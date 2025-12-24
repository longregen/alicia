package livekit

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// handleConfiguration processes Configuration messages
func (d *DefaultMessageDispatcher) handleConfiguration(ctx context.Context, envelope *protocol.Envelope) error {
	config, ok := envelope.Body.(*protocol.Configuration)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid Configuration message", true)
		return fmt.Errorf("invalid Configuration message type")
	}

	return d.protocolHandler.HandleConfiguration(ctx, config)
}

// handleUserMessage processes user text messages
func (d *DefaultMessageDispatcher) handleUserMessage(ctx context.Context, envelope *protocol.Envelope) error {
	userMsg, ok := envelope.Body.(*protocol.UserMessage)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid UserMessage", true)
		return fmt.Errorf("invalid UserMessage type")
	}

	log.Printf("Received user message: %s (conversation: %s)", userMsg.ID, userMsg.ConversationID)

	// Validate conversation ID
	if userMsg.ConversationID != d.conversationID {
		_ = d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, userMsg.ConversationID),
			true)
		return fmt.Errorf("conversation ID mismatch: expected %s, got %s", d.conversationID, userMsg.ConversationID)
	}

	// Process the user message
	processOutput, err := d.processUserInput(ctx, userMsg.ConversationID, userMsg.ID, userMsg.Content, userMsg.PreviousID)
	if err != nil {
		return err
	}

	// Trigger response generation
	d.generateResponseAsync(ctx, userMsg.ConversationID, userMsg.ID, processOutput)

	return nil
}

// processUserInput processes user message text, creates message, retrieves memories
func (d *DefaultMessageDispatcher) processUserInput(
	ctx context.Context,
	conversationID string,
	messageID string,
	textContent string,
	previousID string,
) (*ports.ProcessUserMessageOutput, error) {
	if d.processUserMessageUseCase != nil {
		processInput := &ports.ProcessUserMessageInput{
			ConversationID: conversationID,
			TextContent:    textContent,
			PreviousID:     previousID,
		}

		processOutput, err := d.processUserMessageUseCase.Execute(ctx, processInput)
		if err != nil {
			log.Printf("Failed to process user message: %v", err)
			_ = d.sendError(ctx, protocol.ErrCodeInternalError,
				fmt.Sprintf("Failed to process message: %v", err), true)
			return nil, fmt.Errorf("failed to process message: %w", err)
		}

		log.Printf("Processed user message: %s (sequence: %d, memories: %d)",
			processOutput.Message.ID, processOutput.Message.SequenceNumber, len(processOutput.RelevantMemories))

		// Send MemoryTrace messages for each retrieved memory
		d.sendMemoryTraces(ctx, processOutput.Message.ID, processOutput.RelevantMemories)
		return processOutput, nil
	}

	// Fallback to direct message creation if usecase not available
	sequenceNumber, err := d.messageRepo.GetNextSequenceNumber(ctx, conversationID)
	if err != nil {
		log.Printf("Failed to get sequence number: %v", err)
		return nil, d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to get sequence number: %v", err), true)
	}

	now := time.Now()
	message := &models.Message{
		ID:             messageID,
		ConversationID: conversationID,
		SequenceNumber: sequenceNumber,
		PreviousID:     previousID,
		Role:           models.MessageRoleUser,
		Contents:       textContent,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := d.messageRepo.Create(ctx, message); err != nil {
		log.Printf("Failed to store user message: %v", err)
		return nil, d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to store message: %v", err), true)
	}

	processOutput := &ports.ProcessUserMessageOutput{
		Message:          message,
		RelevantMemories: []*models.Memory{},
	}

	log.Printf("Stored user message: %s (sequence: %d)", message.ID, message.SequenceNumber)
	return processOutput, nil
}

// generateResponseAsync triggers async response generation
func (d *DefaultMessageDispatcher) generateResponseAsync(
	ctx context.Context,
	conversationID string,
	userMessageID string,
	processOutput *ports.ProcessUserMessageOutput,
) {
	if d.generateResponseUseCase == nil {
		return
	}

	// Pre-generate the message ID so we can register it for cancellation
	assistantMsgID := d.idGenerator.GenerateMessageID()

	input := &ports.GenerateResponseInput{
		ConversationID:   conversationID,
		UserMessageID:    processOutput.Message.ID,
		MessageID:        assistantMsgID,
		RelevantMemories: processOutput.RelevantMemories,
		EnableTools:      true,
		EnableReasoning:  true,
		EnableStreaming:  true,
		PreviousID:       processOutput.Message.ID,
	}

	// Generate response asynchronously
	go func() {
		// 5 minute timeout for LLM generation to prevent indefinite hangs
		genCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// Register the generation for cancellation with the correct ID
		d.generationManager.RegisterGeneration(assistantMsgID, cancel)
		defer d.generationManager.UnregisterGeneration(assistantMsgID)

		output, err := d.generateResponseUseCase.Execute(genCtx, input)
		if err != nil {
			log.Printf("Failed to generate response: %v", err)
			_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
				fmt.Sprintf("Failed to generate response: %v", err), true)
			return
		}

		log.Printf("Generated response for user message: %s", userMessageID)

		// Handle streaming vs non-streaming responses
		var fullResponseText string

		if input.EnableStreaming && output.StreamChannel != nil {
			var err error
			fullResponseText, err = d.processStreamingResponse(genCtx, conversationID, output)
			if err != nil {
				return
			}
		} else if !input.EnableStreaming && output.Message != nil {
			fullResponseText = d.sendNonStreamingResponse(genCtx, conversationID, output)
		}

		// Synthesize speech for voice response
		d.synthesizeAndSendSpeech(genCtx, fullResponseText)
	}()
}

// processStreamingResponse handles streaming response chunks and sends protocol messages
func (d *DefaultMessageDispatcher) processStreamingResponse(
	ctx context.Context,
	conversationID string,
	output *ports.GenerateResponseOutput,
) (string, error) {
	// Validate output.Message is not nil before accessing it
	if output.Message == nil {
		log.Printf("Received nil output.Message in streaming response")
		_ = d.sendError(ctx, protocol.ErrCodeInternalError,
			"Received nil output message in streaming response", true)
		return "", fmt.Errorf("received nil output message")
	}

	// Send StartAnswer message to indicate streaming response is starting
	startAnswer := &protocol.StartAnswer{
		ID:             output.Message.ID,
		PreviousID:     output.Message.PreviousID,
		ConversationID: conversationID,
		AnswerType:     protocol.AnswerTypeText,
	}

	if err := d.protocolHandler.SendEnvelope(ctx, &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeStartAnswer,
		Body:           startAnswer,
	}); err != nil {
		log.Printf("Failed to send StartAnswer: %v", err)
		return "", err
	}

	// Process the streaming response chunks and send AssistantSentence messages
	previousID := output.Message.ID
	var sentences []string
	var reasoningSequence int32
	for chunk := range output.StreamChannel {
		if chunk == nil {
			log.Printf("Received nil chunk in response stream")
			_ = d.sendError(ctx, protocol.ErrCodeInternalError,
				"Received nil chunk in response stream", true)
			return "", fmt.Errorf("received nil chunk")
		}
		if chunk.Error != nil {
			log.Printf("Error in response stream: %v", chunk.Error)
			_ = d.sendError(ctx, protocol.ErrCodeInternalError,
				fmt.Sprintf("Streaming error: %v", chunk.Error), true)
			return "", chunk.Error
		}

		// Send ReasoningStep messages if present
		if chunk.Reasoning != "" {
			reasoningStep := &protocol.ReasoningStep{
				ID:             d.idGenerator.GenerateMessageID(),
				MessageID:      output.Message.ID,
				ConversationID: conversationID,
				Sequence:       reasoningSequence,
				Content:        chunk.Reasoning,
			}
			reasoningSequence++

			if err := d.protocolHandler.SendEnvelope(ctx, &protocol.Envelope{
				ConversationID: conversationID,
				Type:           protocol.TypeReasoningStep,
				Body:           reasoningStep,
			}); err != nil {
				log.Printf("Failed to send ReasoningStep: %v", err)
				// Don't fail the whole operation if reasoning step sending fails
			}
		}

		// Send AssistantSentence messages for sentence chunks
		if chunk.SentenceID != "" && chunk.Text != "" {
			sentences = append(sentences, chunk.Text)

			sentenceMsg := &protocol.AssistantSentence{
				ID:             chunk.SentenceID,
				PreviousID:     previousID,
				ConversationID: conversationID,
				Sequence:       int32(chunk.Sequence),
				Text:           chunk.Text,
				IsFinal:        chunk.IsFinal,
			}

			if err := d.protocolHandler.SendEnvelope(ctx, &protocol.Envelope{
				ConversationID: conversationID,
				Type:           protocol.TypeAssistantSentence,
				Body:           sentenceMsg,
			}); err != nil {
				log.Printf("Failed to send AssistantSentence: %v", err)
				return "", err
			}

			previousID = chunk.SentenceID
		}

		// Send ToolCall messages if present
		if chunk.ToolCall != nil && chunk.ToolUseID != "" {
			// Get the ToolUse from the database to access full details
			toolUse, err := d.protocolHandler.toolUseRepo.GetByID(ctx, chunk.ToolUseID)
			if err != nil {
				log.Printf("Failed to get tool use %s: %v", chunk.ToolUseID, err)
				continue
			}

			if !chunk.IsToolExecutionResult {
				// Send ToolUseRequest (Type 6) before execution
				if err := d.protocolHandler.SendToolUseRequest(ctx, toolUse); err != nil {
					log.Printf("Failed to send ToolUseRequest: %v", err)
				} else {
					log.Printf("Sent ToolUseRequest for tool %s (ID: %s)", toolUse.ToolName, toolUse.ID)
				}
			} else {
				// Send ToolUseResult (Type 7) after execution
				if err := d.protocolHandler.SendToolUseResult(ctx, toolUse); err != nil {
					log.Printf("Failed to send ToolUseResult: %v", err)
				} else {
					log.Printf("Sent ToolUseResult for tool %s (ID: %s, Status: %s)", toolUse.ToolName, toolUse.ID, toolUse.Status)
				}
			}
		}
	}

	// Combine all sentences for return
	fullResponseText := ""
	for i, s := range sentences {
		if i > 0 {
			fullResponseText += " "
		}
		fullResponseText += s
	}

	log.Printf("Completed streaming response for message: %s", output.Message.ID)
	return fullResponseText, nil
}

// sendNonStreamingResponse sends complete non-streaming response
func (d *DefaultMessageDispatcher) sendNonStreamingResponse(
	ctx context.Context,
	conversationID string,
	output *ports.GenerateResponseOutput,
) string {
	if output.Message == nil {
		return ""
	}

	// Send complete non-streaming response
	assistantMsg := &protocol.AssistantMessage{
		ID:             output.Message.ID,
		PreviousID:     output.Message.PreviousID,
		ConversationID: output.Message.ConversationID,
		Content:        output.Message.Contents,
	}

	responseEnvelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeAssistantMessage,
		Body:           assistantMsg,
	}

	if err := d.protocolHandler.SendEnvelope(ctx, responseEnvelope); err != nil {
		log.Printf("Failed to send assistant message: %v", err)
	}

	// Send ReasoningStep messages for non-streaming responses
	for _, step := range output.ReasoningSteps {
		reasoningStep := &protocol.ReasoningStep{
			ID:             step.ID,
			MessageID:      step.MessageID,
			ConversationID: conversationID,
			Sequence:       int32(step.SequenceNumber),
			Content:        step.Content,
		}

		if err := d.protocolHandler.SendEnvelope(ctx, &protocol.Envelope{
			ConversationID: conversationID,
			Type:           protocol.TypeReasoningStep,
			Body:           reasoningStep,
		}); err != nil {
			log.Printf("Failed to send ReasoningStep: %v", err)
			// Don't fail the whole operation if reasoning step sending fails
		}
	}

	return output.Message.Contents
}

// synthesizeAndSendSpeech synthesizes and sends audio response
func (d *DefaultMessageDispatcher) synthesizeAndSendSpeech(ctx context.Context, text string) {
	if d.ttsService == nil || text == "" {
		return
	}

	log.Printf("Synthesizing speech for response: %d characters", len(text))

	// Get TTS options from conversation preferences if available
	ttsOptions := &ports.TTSOptions{
		OutputFormat: "pcm",
	}

	result, err := d.ttsService.Synthesize(ctx, text, ttsOptions)
	if err != nil {
		log.Printf("Failed to synthesize speech: %v", err)
		// Send error to client but don't fail the whole operation
		_ = d.sendError(ctx, protocol.ErrCodeServiceUnavailable,
			fmt.Sprintf("Failed to synthesize speech: %v", err), false)
	} else if result != nil && len(result.Audio) > 0 {
		log.Printf("Synthesized %d bytes of audio", len(result.Audio))

		// Send audio via the protocol handler's agent
		if err := d.protocolHandler.SendAudio(ctx, result.Audio, result.Format); err != nil {
			log.Printf("Failed to send synthesized audio: %v", err)
			_ = d.sendError(ctx, protocol.ErrCodeServiceUnavailable,
				fmt.Sprintf("Failed to send audio: %v", err), false)
		} else {
			log.Printf("Sent synthesized audio to client")
		}
	}
}

// handleToolUseRequest processes tool execution requests
func (d *DefaultMessageDispatcher) handleToolUseRequest(ctx context.Context, envelope *protocol.Envelope) error {
	req, ok := envelope.Body.(*protocol.ToolUseRequest)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid ToolUseRequest", true)
		return fmt.Errorf("invalid ToolUseRequest type")
	}

	log.Printf("Received tool use request: %s (tool: %s, conversation: %s)", req.ID, req.ToolName, req.ConversationID)

	// Validate conversation ID matches
	if req.ConversationID != d.conversationID {
		_ = d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, req.ConversationID),
			true)
		return fmt.Errorf("conversation ID mismatch: expected %s, got %s", d.conversationID, req.ConversationID)
	}

	// Validate conversation ID format
	if err := services.ValidateConversationIDFormat(req.ConversationID); err != nil {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			fmt.Sprintf("Invalid conversation ID format: %v", err), true)
	}

	// Validate tool name length (max 100 characters)
	if err := services.ValidateStringLength(req.ToolName, "tool name", 1, 100); err != nil {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			fmt.Sprintf("Invalid tool name: %v", err), true)
	}

	// Validate arguments size (max 1MB)
	const maxArgumentsSize = 1024 * 1024 // 1MB
	if err := services.ValidateJSONSize(req.Parameters, "tool arguments", maxArgumentsSize); err != nil {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			fmt.Sprintf("Invalid tool arguments: %v", err), true)
	}

	// Set timeout from request or use default
	timeoutMs := req.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = protocol.DefaultToolTimeout
	}

	// Validate timeout range (must be positive, max 5 minutes = 300000ms)
	const maxTimeout = 300000 // 5 minutes in milliseconds
	if err := services.ValidateRange(int(timeoutMs), "timeout", 1, maxTimeout); err != nil {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			fmt.Sprintf("Invalid timeout: %v", err), true)
	}

	// Execute the tool via HandleToolCall use case
	input := &ports.HandleToolInput{
		ToolUseID:      req.ID,
		ToolName:       req.ToolName,
		Arguments:      req.Parameters,
		TimeoutMs:      int(timeoutMs),
		MessageID:      req.MessageID, // Use the message ID from the request
		ConversationID: req.ConversationID,
	}

	var output *ports.HandleToolOutput
	var err error

	if d.handleToolUseCase != nil {
		output, err = d.handleToolUseCase.Execute(ctx, input)
	} else {
		err = fmt.Errorf("tool execution not available: handleToolUseCase is nil")
		log.Printf("Tool execution unavailable for %s", req.ToolName)
	}

	// Prepare result message
	result := &protocol.ToolUseResult{
		ID:             d.idGenerator.GenerateToolUseID(),
		RequestID:      req.ID,
		ConversationID: req.ConversationID,
		Success:        err == nil && output != nil,
	}

	if err != nil {
		result.ErrorCode = "EXECUTION_ERROR"
		result.ErrorMessage = err.Error()
		log.Printf("Tool execution failed for %s: %v", req.ToolName, err)
	} else if output != nil {
		result.Result = output.Result
		log.Printf("Tool execution succeeded for %s", req.ToolName)
	}

	// Send result back via protocol handler
	resultEnvelope := &protocol.Envelope{
		ConversationID: req.ConversationID,
		Type:           protocol.TypeToolUseResult,
		Body:           result,
	}

	if err := d.protocolHandler.SendEnvelope(ctx, resultEnvelope); err != nil {
		log.Printf("Failed to send tool result: %v", err)
		return fmt.Errorf("failed to send tool result: %w", err)
	}

	return nil
}

// handleControlStop processes stop control messages
func (d *DefaultMessageDispatcher) handleControlStop(ctx context.Context, envelope *protocol.Envelope) error {
	stopMsg, ok := envelope.Body.(*protocol.ControlStop)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid ControlStop message", true)
		return fmt.Errorf("invalid ControlStop message type")
	}

	log.Printf("Received stop control: type=%s, target=%s, reason=%s",
		stopMsg.StopType, stopMsg.TargetID, stopMsg.Reason)

	// Validate conversation ID
	if stopMsg.ConversationID != d.conversationID {
		_ = d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s",
				d.conversationID, stopMsg.ConversationID),
			true)
		return fmt.Errorf("conversation ID mismatch: expected %s, got %s", d.conversationID, stopMsg.ConversationID)
	}

	// Determine which operations to stop based on StopType
	switch stopMsg.StopType {
	case protocol.StopTypeGeneration:
		// Cancel LLM generation only
		if err := d.generationManager.CancelGeneration(stopMsg.TargetID); err != nil {
			log.Printf("Failed to cancel generation: %v", err)
			return d.sendError(ctx, protocol.ErrCodeInternalError,
				fmt.Sprintf("Failed to cancel generation: %v", err), true)
		}

	case protocol.StopTypeSpeech:
		// Cancel TTS synthesis only
		if err := d.generationManager.CancelTTS(stopMsg.TargetID); err != nil {
			log.Printf("Failed to cancel TTS: %v", err)
			return d.sendError(ctx, protocol.ErrCodeInternalError,
				fmt.Sprintf("Failed to cancel TTS: %v", err), true)
		}

	case protocol.StopTypeAll, "":
		// Default: cancel everything
		if err := d.generationManager.CancelGeneration(stopMsg.TargetID); err != nil {
			log.Printf("Failed to cancel generation: %v", err)
		}
		if err := d.generationManager.CancelTTS(stopMsg.TargetID); err != nil {
			log.Printf("Failed to cancel TTS: %v", err)
		}

	default:
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData,
			fmt.Sprintf("Unknown stop type: %s", stopMsg.StopType), true)
		return fmt.Errorf("unknown stop type: %s", stopMsg.StopType)
	}

	// Send acknowledgement
	return d.protocolHandler.sendAcknowledgement(ctx, envelope.StanzaID, true)
}

// handleControlVariation processes variation control messages
func (d *DefaultMessageDispatcher) handleControlVariation(ctx context.Context, envelope *protocol.Envelope) error {
	varMsg, ok := envelope.Body.(*protocol.ControlVariation)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid ControlVariation message", true)
		return fmt.Errorf("invalid ControlVariation message type")
	}

	log.Printf("Received variation control: type=%s, target=%s", varMsg.Mode, varMsg.TargetID)

	// Validate conversation ID
	if varMsg.ConversationID != d.conversationID {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s",
				d.conversationID, varMsg.ConversationID),
			true)
	}

	// Validate target ID is provided
	if varMsg.TargetID == "" {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			"TargetID is required for variation requests", true)
	}

	// Get the target message to ensure it exists
	targetMessage, err := d.messageRepo.GetByID(ctx, varMsg.TargetID)
	if err != nil {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Target message not found: %s", varMsg.TargetID), true)
	}

	// Only allow variations on assistant messages
	if !targetMessage.IsFromAssistant() {
		return d.sendError(ctx, protocol.ErrCodeInvalidState,
			"Can only create variations of assistant messages", true)
	}

	// Handle different variation types
	switch varMsg.Mode {
	case protocol.VariationTypeRegenerate:
		return d.handleRegenerate(ctx, envelope, targetMessage)

	case protocol.VariationTypeEdit:
		return d.handleEdit(ctx, envelope, targetMessage, varMsg.NewContent)

	case protocol.VariationTypeContinue:
		return d.handleContinue(ctx, envelope, targetMessage)

	default:
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			fmt.Sprintf("Unknown variation type: %s", varMsg.Mode), true)
	}
}

// handleAudioChunk processes incoming audio data
func (d *DefaultMessageDispatcher) handleAudioChunk(ctx context.Context, envelope *protocol.Envelope) error {
	audioChunk, ok := envelope.Body.(*protocol.AudioChunk)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid AudioChunk message", true)
		return fmt.Errorf("invalid AudioChunk message type")
	}

	log.Printf("Received audio chunk: sequence=%d, duration=%dms", audioChunk.Sequence, audioChunk.DurationMs)

	// Send audio to ASR service for transcription
	if d.asrService != nil && len(audioChunk.Data) > 0 {
		result, err := d.asrService.Transcribe(ctx, audioChunk.Data, audioChunk.Format)
		if err != nil {
			log.Printf("ASR transcription failed for chunk %d: %v", audioChunk.Sequence, err)
			return fmt.Errorf("ASR transcription failed: %w", err)
		}

		if result != nil && result.Text != "" {
			// Send transcription result back to client
			transcription := &protocol.Transcription{
				ID:             d.idGenerator.GenerateMessageID(),
				ConversationID: audioChunk.ConversationID,
				Text:           result.Text,
				Confidence:     result.Confidence,
				Language:       result.Language,
				Final:          true,
			}

			transcriptionEnvelope := &protocol.Envelope{
				ConversationID: audioChunk.ConversationID,
				Type:           protocol.TypeTranscription,
				Body:           transcription,
			}

			if err := d.protocolHandler.SendEnvelope(ctx, transcriptionEnvelope); err != nil {
				log.Printf("Failed to send transcription: %v", err)
				return fmt.Errorf("failed to send transcription: %w", err)
			}

			log.Printf("Sent transcription from chunk %d: %s (confidence: %.2f)",
				audioChunk.Sequence, result.Text, result.Confidence)
		}
	}

	return nil
}

// handleTranscription processes transcription messages
func (d *DefaultMessageDispatcher) handleTranscription(ctx context.Context, envelope *protocol.Envelope) error {
	transcription, ok := envelope.Body.(*protocol.Transcription)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid Transcription message", true)
		return fmt.Errorf("invalid Transcription message type")
	}

	log.Printf("Received transcription: %s (final: %v)", transcription.Text, transcription.Final)

	// Only process final transcriptions
	if !transcription.Final {
		// Intermediate transcriptions are just echoed back or logged
		return nil
	}

	// Validate conversation ID
	if transcription.ConversationID != d.conversationID {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, transcription.ConversationID),
			true)
	}

	// Process the transcribed message
	processOutput, err := d.processUserInput(ctx, transcription.ConversationID, transcription.ID, transcription.Text, transcription.PreviousID)
	if err != nil {
		return err
	}

	// Trigger response generation
	d.generateResponseAsync(ctx, transcription.ConversationID, transcription.ID, processOutput)

	return nil
}

// handleRegenerate regenerates a response for the same user message
func (d *DefaultMessageDispatcher) handleRegenerate(ctx context.Context, envelope *protocol.Envelope, targetMessage *models.Message) error {
	log.Printf("Regenerating response for message: %s", targetMessage.ID)

	// First, cancel any active generation for this message
	_ = d.generationManager.CancelGeneration(targetMessage.ID)

	// Delete the existing assistant message and related data
	// This will be handled by cascade delete in the database for related entities
	if err := d.messageRepo.Delete(ctx, targetMessage.ID); err != nil {
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to delete old message: %v", err), true)
	}

	// Get the user message that prompted this response (PreviousID)
	if targetMessage.PreviousID == "" {
		return d.sendError(ctx, protocol.ErrCodeInvalidState,
			"Cannot regenerate: target message has no previous message reference", true)
	}

	// Trigger a new response generation using the GenerateResponseUseCase
	if d.generateResponseUseCase != nil {
		// Pre-generate the message ID so we can register it for cancellation
		assistantMsgID := d.idGenerator.GenerateMessageID()

		input := &ports.GenerateResponseInput{
			ConversationID:  targetMessage.ConversationID,
			UserMessageID:   targetMessage.PreviousID,
			MessageID:       assistantMsgID, // Pass pre-generated ID to avoid race condition
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: true,
			PreviousID:      targetMessage.PreviousID,
		}

		// Generate response asynchronously
		go func() {
			// 5 minute timeout for LLM generation to prevent indefinite hangs
			// Use parent context for proper cancellation propagation
			genCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			// Register the generation for cancellation with the correct ID
			d.generationManager.RegisterGeneration(assistantMsgID, cancel)
			defer d.generationManager.UnregisterGeneration(assistantMsgID)

			output, err := d.generateResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to regenerate response: %v", err)
				_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
					fmt.Sprintf("Failed to regenerate response: %v", err), true)
				return
			}

			log.Printf("Regenerated response for user message: %s", targetMessage.PreviousID)

			// If not streaming, send the complete response
			if !input.EnableStreaming && output.Message != nil {
				assistantMsg := &protocol.AssistantMessage{
					ID:             output.Message.ID,
					PreviousID:     output.Message.PreviousID,
					ConversationID: output.Message.ConversationID,
					Content:        output.Message.Contents,
				}

				responseEnvelope := &protocol.Envelope{
					ConversationID: targetMessage.ConversationID,
					Type:           protocol.TypeAssistantMessage,
					Body:           assistantMsg,
				}

				if err := d.protocolHandler.SendEnvelope(genCtx, responseEnvelope); err != nil {
					log.Printf("Failed to send regenerated message: %v", err)
				}

				// Send ReasoningStep messages for non-streaming responses
				for _, step := range output.ReasoningSteps {
					reasoningStep := &protocol.ReasoningStep{
						ID:             step.ID,
						MessageID:      step.MessageID,
						ConversationID: targetMessage.ConversationID,
						Sequence:       int32(step.SequenceNumber),
						Content:        step.Content,
					}

					if err := d.protocolHandler.SendEnvelope(genCtx, &protocol.Envelope{
						ConversationID: targetMessage.ConversationID,
						Type:           protocol.TypeReasoningStep,
						Body:           reasoningStep,
					}); err != nil {
						log.Printf("Failed to send ReasoningStep: %v", err)
						// Don't fail the whole operation if reasoning step sending fails
					}
				}
			}
		}()
	}

	// Send acknowledgement
	return d.protocolHandler.sendAcknowledgement(ctx, envelope.StanzaID, true)
}

// handleEdit updates an existing message with new content
func (d *DefaultMessageDispatcher) handleEdit(ctx context.Context, envelope *protocol.Envelope, targetMessage *models.Message, newContent string) error {
	log.Printf("Editing message: %s", targetMessage.ID)

	// Validate that new content is provided
	if newContent == "" {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			"NewContent is required for edit variation", true)
	}

	// First, cancel any active generation for this message
	_ = d.generationManager.CancelGeneration(targetMessage.ID)

	// Update the message content
	targetMessage.Contents = newContent
	targetMessage.UpdatedAt = time.Now()
	if err := d.messageRepo.Update(ctx, targetMessage); err != nil {
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to update message: %v", err), true)
	}

	log.Printf("Updated message content: %s", targetMessage.ID)

	// Send updated AssistantMessage back to the client
	assistantMsg := &protocol.AssistantMessage{
		ID:             targetMessage.ID,
		PreviousID:     targetMessage.PreviousID,
		ConversationID: targetMessage.ConversationID,
		Content:        targetMessage.Contents,
	}

	responseEnvelope := &protocol.Envelope{
		ConversationID: targetMessage.ConversationID,
		Type:           protocol.TypeAssistantMessage,
		Body:           assistantMsg,
	}

	if err := d.protocolHandler.SendEnvelope(ctx, responseEnvelope); err != nil {
		log.Printf("Failed to send updated message: %v", err)
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to send updated message: %v", err), true)
	}

	log.Printf("Sent updated message to client: %s", targetMessage.ID)

	// Send acknowledgement
	return d.protocolHandler.sendAcknowledgement(ctx, envelope.StanzaID, true)
}

// handleContinue extends an existing message
func (d *DefaultMessageDispatcher) handleContinue(ctx context.Context, envelope *protocol.Envelope, targetMessage *models.Message) error {
	log.Printf("Continuing message: %s", targetMessage.ID)

	// Get the user message that prompted this response
	if targetMessage.PreviousID == "" {
		return d.sendError(ctx, protocol.ErrCodeInvalidState,
			"Cannot continue: target message has no previous message reference", true)
	}

	// Trigger continuation by regenerating with the existing content as context
	// Note: This is a simplified implementation. A full implementation would require
	// the GenerateResponseUseCase to support continuation mode where it appends to
	// the existing message rather than creating a new one.
	if d.generateResponseUseCase != nil {
		// Pre-generate the continuation message ID so we can register it for cancellation
		continuationMsgID := d.idGenerator.GenerateMessageID()

		input := &ports.GenerateResponseInput{
			ConversationID:  targetMessage.ConversationID,
			UserMessageID:   targetMessage.PreviousID,
			MessageID:       continuationMsgID, // Pass pre-generated ID to avoid race condition
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: true,
			PreviousID:      targetMessage.ID, // Use current message as previous
		}

		// Generate continuation asynchronously
		go func() {
			// 5 minute timeout for LLM generation to prevent indefinite hangs
			// Use parent context for proper cancellation propagation
			genCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			// Register the generation for cancellation with the correct ID
			d.generationManager.RegisterGeneration(continuationMsgID, cancel)
			defer d.generationManager.UnregisterGeneration(continuationMsgID)

			output, err := d.generateResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to continue response: %v", err)
				_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
					fmt.Sprintf("Failed to continue response: %v", err), true)
				return
			}

			log.Printf("Continued response for message: %s", targetMessage.ID)

			// Update the target message with appended content
			if output.Message != nil {
				targetMessage.Contents += "\n\n" + output.Message.Contents
				targetMessage.UpdatedAt = time.Now()

				if err := d.messageRepo.Update(genCtx, targetMessage); err != nil {
					log.Printf("Failed to update continued message: %v", err)
					return
				}

				// Send the updated message back to client
				assistantMsg := &protocol.AssistantMessage{
					ID:             targetMessage.ID,
					PreviousID:     targetMessage.PreviousID,
					ConversationID: targetMessage.ConversationID,
					Content:        targetMessage.Contents,
				}

				responseEnvelope := &protocol.Envelope{
					ConversationID: targetMessage.ConversationID,
					Type:           protocol.TypeAssistantMessage,
					Body:           assistantMsg,
				}

				if err := d.protocolHandler.SendEnvelope(genCtx, responseEnvelope); err != nil {
					log.Printf("Failed to send continued message: %v", err)
				}
			}
		}()
	}

	// Send acknowledgement
	return d.protocolHandler.sendAcknowledgement(ctx, envelope.StanzaID, true)
}

// sendMemoryTraces sends MemoryTrace protocol messages for each retrieved memory
func (d *DefaultMessageDispatcher) sendMemoryTraces(ctx context.Context, messageID string, memories []*models.Memory) {
	if len(memories) == 0 {
		return
	}

	for _, memory := range memories {
		// Create MemoryTrace message
		memoryTrace := &protocol.MemoryTrace{
			ID:             d.idGenerator.GenerateMessageID(),
			MessageID:      messageID,
			ConversationID: d.conversationID,
			MemoryID:       memory.ID,
			Content:        memory.Content,
			Relevance:      memory.Importance, // Use importance as relevance score
		}

		// Send via protocol handler
		envelope := &protocol.Envelope{
			ConversationID: d.conversationID,
			Type:           protocol.TypeMemoryTrace,
			Body:           memoryTrace,
		}

		if err := d.protocolHandler.SendEnvelope(ctx, envelope); err != nil {
			log.Printf("Failed to send memory trace for memory %s: %v", memory.ID, err)
			// Don't fail the whole operation if memory trace sending fails
			continue
		}

		log.Printf("Sent memory trace: memory=%s, message=%s, relevance=%.2f",
			memory.ID, messageID, memory.Importance)
	}
}

// handleErrorMessage processes error messages from clients
func (d *DefaultMessageDispatcher) handleErrorMessage(ctx context.Context, envelope *protocol.Envelope) error {
	errorMsg, ok := envelope.Body.(*protocol.ErrorMessage)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid ErrorMessage", true)
		return fmt.Errorf("invalid ErrorMessage type")
	}

	log.Printf("Received error from client: code=%d, message=%s, severity=%d, recoverable=%v",
		errorMsg.Code, errorMsg.Message, errorMsg.Severity, errorMsg.Recoverable)

	// Log the error for monitoring and debugging purposes
	// In a production system, this might be sent to an error tracking service
	// For now, we just log and acknowledge receipt
	return nil
}

// handleToolUseResult processes tool execution results from clients
func (d *DefaultMessageDispatcher) handleToolUseResult(ctx context.Context, envelope *protocol.Envelope) error {
	result, ok := envelope.Body.(*protocol.ToolUseResult)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid ToolUseResult", true)
		return fmt.Errorf("invalid ToolUseResult type")
	}

	log.Printf("Received tool result from client: requestID=%s, success=%v", result.RequestID, result.Success)

	// Validate conversation ID
	if result.ConversationID != d.conversationID {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, result.ConversationID),
			true)
	}

	// TODO: Store the tool result and potentially trigger continuation of response generation
	// For now, we just log the result
	if result.Success {
		log.Printf("Tool execution succeeded for request %s", result.RequestID)
	} else {
		log.Printf("Tool execution failed for request %s: %s - %s",
			result.RequestID, result.ErrorCode, result.ErrorMessage)
	}

	return nil
}

// handleAcknowledgement processes acknowledgement messages from clients
func (d *DefaultMessageDispatcher) handleAcknowledgement(ctx context.Context, envelope *protocol.Envelope) error {
	ack, ok := envelope.Body.(*protocol.Acknowledgement)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid Acknowledgement", true)
		return fmt.Errorf("invalid Acknowledgement type")
	}

	log.Printf("Received acknowledgement from client: stanzaID=%d, success=%v", ack.AckedStanzaID, ack.Success)

	// Validate conversation ID
	if ack.ConversationID != d.conversationID {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, ack.ConversationID),
			true)
	}

	// Acknowledgements are primarily used for tracking message delivery
	// For now, we just log receipt
	return nil
}

// handleCommentary processes commentary messages
func (d *DefaultMessageDispatcher) handleCommentary(ctx context.Context, envelope *protocol.Envelope) error {
	commentary, ok := envelope.Body.(*protocol.Commentary)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid Commentary message", true)
		return fmt.Errorf("invalid Commentary message type")
	}

	log.Printf("Received commentary: messageID=%s, type=%s, content=%s",
		commentary.MessageID, commentary.CommentaryType, commentary.Content)

	// Commentary messages are internal assistant thoughts/observations
	// They can be logged for debugging but don't require action
	// In the future, they could be stored for analysis or displayed to users
	return nil
}
