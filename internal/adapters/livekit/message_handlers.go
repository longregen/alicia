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

	if err := d.protocolHandler.HandleConfiguration(ctx, config); err != nil {
		return err
	}

	// After successful configuration, send server info and session stats to the client
	// These provide the frontend with initial state about the server connection
	if err := d.SendServerInfo(ctx); err != nil {
		log.Printf("Failed to send ServerInfo after configuration: %v", err)
		// Non-fatal - continue even if this fails
	}

	if err := d.SendSessionStats(ctx); err != nil {
		log.Printf("Failed to send SessionStats after configuration: %v", err)
		// Non-fatal - continue even if this fails
	}

	// Send available elite solutions (if any exist from optimization runs)
	if err := d.SendEliteOptions(ctx); err != nil {
		log.Printf("Failed to send EliteOptions after configuration: %v", err)
		// Non-fatal - continue even if this fails
	}

	return nil
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

	// Use SendMessageUseCase if available
	if d.sendMessageUseCase != nil {
		input := &ports.SendMessageInput{
			ConversationID:  userMsg.ConversationID,
			TextContent:     userMsg.Content,
			PreviousID:      userMsg.PreviousID,
			LocalID:         userMsg.ID,
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: true,
		}

		// Execute asynchronously to avoid blocking
		go func() {
			// 5 minute timeout for LLM generation
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			output, err := d.sendMessageUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to send message: %v", err)
				if genCtx.Err() == nil {
					_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
						fmt.Sprintf("Failed to send message: %v", err), true)
				}
				return
			}

			// Send memory traces for retrieved memories
			if output.UserMessage != nil && len(output.RelevantMemories) > 0 {
				d.sendMemoryTraces(genCtx, output.UserMessage.ID, output.RelevantMemories)
			}

			// Handle streaming response
			if output.StreamChannel != nil {
				// Build a GenerateResponseOutput-like structure for processStreamingResponse
				streamOutput := &ports.GenerateResponseOutput{
					Message:       output.AssistantMessage,
					StreamChannel: output.StreamChannel,
				}
				// skipStartAnswer=false because SendMessageUseCase doesn't use a Notifier
				_, err = d.processStreamingResponse(genCtx, userMsg.ConversationID, streamOutput, false)
				if err != nil {
					log.Printf("Error processing streaming response: %v", err)
				}
			} else if output.AssistantMessage != nil {
				// Non-streaming response
				streamOutput := &ports.GenerateResponseOutput{
					Message: output.AssistantMessage,
				}
				d.sendNonStreamingResponse(genCtx, userMsg.ConversationID, streamOutput)
			}

			log.Printf("Completed message handling for user message: %s", userMsg.ID)
		}()

		return nil
	}

	// Fallback to old behavior if use case not available
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
			MessageID:      messageID,
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

	// Fetch conversation to get the current tip for proper message chaining
	conversation, err := d.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		log.Printf("Failed to get conversation: %v", err)
		return nil, d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to get conversation: %v", err), true)
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

	// Use conversation tip as previous_id for proper message chaining
	if conversation.TipMessageID != nil && *conversation.TipMessageID != "" {
		message.PreviousID = *conversation.TipMessageID
	}

	if err := d.messageRepo.Create(ctx, message); err != nil {
		log.Printf("Failed to store user message: %v", err)
		return nil, d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to store message: %v", err), true)
	}

	// Update conversation tip to point to the new message
	if err := d.conversationRepo.UpdateTip(ctx, conversationID, message.ID); err != nil {
		log.Printf("Failed to update conversation tip: %v", err)
		// Non-fatal, continue
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
	_ context.Context, // unused - we create our own context for async operation
	conversationID string,
	userMessageID string,
	processOutput *ports.ProcessUserMessageOutput,
) {
	if d.generateResponseUseCase == nil {
		return
	}

	// Pre-generate the message ID so we can register it for cancellation
	assistantMsgID := d.idGenerator.GenerateMessageID()

	// Generate response asynchronously
	// Use context.Background() so generation continues even if user disconnects
	// The response will be saved to DB and can be retrieved when user reconnects
	go func() {
		// 5 minute timeout for LLM generation to prevent indefinite hangs
		genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Create a notifier to send real-time progress updates to the client
		notifier := NewProtocolNotifier(genCtx, d.protocolHandler, d.idGenerator)

		input := &ports.GenerateResponseInput{
			ConversationID:   conversationID,
			UserMessageID:    processOutput.Message.ID,
			MessageID:        assistantMsgID,
			RelevantMemories: processOutput.RelevantMemories,
			EnableTools:      true,
			EnableReasoning:  true,
			EnableStreaming:  true,
			PreviousID:       processOutput.Message.ID,
			Notifier:         notifier,
		}

		// Register the generation for cancellation with the correct ID
		d.generationManager.RegisterGeneration(assistantMsgID, cancel)
		defer d.generationManager.UnregisterGeneration(assistantMsgID)

		output, err := d.generateResponseUseCase.Execute(genCtx, input)
		if err != nil {
			log.Printf("Failed to generate response: %v", err)
			// Don't send error if context was cancelled (user disconnected)
			if genCtx.Err() == nil {
				_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
					fmt.Sprintf("Failed to generate response: %v", err), true)
			}
			return
		}

		log.Printf("Generated response for user message: %s", userMessageID)

		// Handle streaming vs non-streaming responses
		if input.EnableStreaming && output.StreamChannel != nil {
			// skipStartAnswer=true because we use a Notifier which already sent StartAnswer
			_, err = d.processStreamingResponse(genCtx, conversationID, output, true)
			if err != nil {
				return
			}
		} else if !input.EnableStreaming && output.Message != nil {
			d.sendNonStreamingResponse(genCtx, conversationID, output)
		}

		// Note: For streaming responses, TTS is synthesized per-sentence in processStreamingResponse
		// For non-streaming responses, TTS synthesis should be added in sendNonStreamingResponse
	}()
}

// processStreamingResponse handles streaming response chunks and sends protocol messages.
// If skipStartAnswer is true, the StartAnswer message is not sent (useful when the caller
// already sent it via a Notifier).
func (d *DefaultMessageDispatcher) processStreamingResponse(
	ctx context.Context,
	conversationID string,
	output *ports.GenerateResponseOutput,
	skipStartAnswer bool,
) (string, error) {
	// Validate output.Message is not nil before accessing it
	if output.Message == nil {
		log.Printf("Received nil output.Message in streaming response")
		_ = d.sendError(ctx, protocol.ErrCodeInternalError,
			"Received nil output message in streaming response", true)
		return "", fmt.Errorf("received nil output message")
	}

	// Send StartAnswer message to indicate streaming response is starting
	// (unless it was already sent by the Notifier)
	if !skipStartAnswer {
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

			// Synthesize and send audio for this sentence
			d.synthesizeAndSendAudioChunk(ctx, conversationID, output.Message.ID, chunk.SentenceID, chunk.Text, int32(chunk.Sequence), chunk.IsFinal)

			previousID = chunk.SentenceID
		}

		// Send ToolCall messages if present
		if chunk.ToolCall != nil && chunk.ToolUseID != "" {
			// Get the ToolUse from the database to access full details
			toolUse, err := d.protocolHandler.GetToolUseRepo().GetByID(ctx, chunk.ToolUseID)
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

// synthesizeAndSendAudioChunk synthesizes speech for a sentence and sends it as an AudioChunk protocol message
// isLast indicates whether this is the final sentence of the response
func (d *DefaultMessageDispatcher) synthesizeAndSendAudioChunk(ctx context.Context, conversationID string, messageID string, sentenceID string, text string, sequence int32, isLast bool) {
	if d.synthesizeSpeechUseCase == nil || text == "" {
		return
	}

	log.Printf("Synthesizing speech for sentence %d: %d characters (isLast: %v)", sequence, len(text), isLast)

	// Use the SynthesizeSpeech use case to synthesize and store audio
	input := &ports.SynthesizeSpeechInput{
		Text:            text,
		MessageID:       messageID,
		SentenceID:      sentenceID,
		OutputFormat:    "opus", // Use opus for protocol messages (better compression)
		EnableStreaming: false,
	}

	output, err := d.synthesizeSpeechUseCase.Execute(ctx, input)
	if err != nil {
		log.Printf("Failed to synthesize speech for sentence %d: %v", sequence, err)
		// Don't fail the whole operation, just skip audio for this sentence
		return
	}

	if output == nil || len(output.AudioData) == 0 {
		log.Printf("No audio data received for sentence %d", sequence)
		return
	}

	log.Printf("Synthesized %d bytes of audio for sentence %d (duration: %dms)", len(output.AudioData), sequence, output.DurationMs)

	// Send AudioChunk protocol message
	audioChunk := &protocol.AudioChunk{
		ConversationID: conversationID,
		Format:         output.Format,
		Sequence:       sequence,
		DurationMs:     int32(output.DurationMs),
		Data:           output.AudioData,
		IsLast:         isLast,
		Timestamp:      uint64(time.Now().UnixMilli()),
	}

	if err := d.protocolHandler.SendEnvelope(ctx, &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeAudioChunk,
		Body:           audioChunk,
	}); err != nil {
		log.Printf("Failed to send AudioChunk for sentence %d: %v", sequence, err)
		// Don't fail, just log the error
		return
	}

	log.Printf("Sent AudioChunk for sentence %d", sequence)

	// Also send to LiveKit audio track for voice conversations
	if err := d.protocolHandler.SendAudio(ctx, output.AudioData, output.Format); err != nil {
		log.Printf("Failed to send audio to LiveKit track for sentence %d: %v", sequence, err)
		// Non-fatal, audio chunk was already sent via protocol
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
	return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
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

	// Handle different variation types based on message role
	switch varMsg.Mode {
	case protocol.VariationTypeEdit:
		// Edit mode supports both user and assistant messages
		if targetMessage.IsFromAssistant() {
			return d.handleAssistantEdit(ctx, envelope, targetMessage, varMsg.NewContent)
		} else if targetMessage.IsFromUser() {
			return d.handleUserEdit(ctx, envelope, targetMessage, varMsg.NewContent)
		} else {
			return d.sendError(ctx, protocol.ErrCodeInvalidState,
				"Can only edit user or assistant messages", true)
		}

	case protocol.VariationTypeRegenerate:
		// Regenerate only applies to assistant messages
		if !targetMessage.IsFromAssistant() {
			return d.sendError(ctx, protocol.ErrCodeInvalidState,
				"Can only regenerate assistant messages", true)
		}
		return d.handleRegenerate(ctx, envelope, targetMessage)

	case protocol.VariationTypeContinue:
		// Continue only applies to assistant messages
		if !targetMessage.IsFromAssistant() {
			return d.sendError(ctx, protocol.ErrCodeInvalidState,
				"Can only continue assistant messages", true)
		}
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

	// Use RegenerateResponseUseCase if available
	if d.regenerateResponseUseCase != nil {
		input := &ports.RegenerateResponseInput{
			MessageID:       targetMessage.ID,
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: true,
		}

		// Execute asynchronously
		go func() {
			// 5 minute timeout for LLM generation
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			output, err := d.regenerateResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to regenerate response: %v", err)
				if genCtx.Err() == nil {
					_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
						fmt.Sprintf("Failed to regenerate response: %v", err), true)
				}
				return
			}

			log.Printf("Regenerated response, deleted message: %s, new message: %s",
				output.DeletedMessageID, output.NewMessage.ID)

			// Handle streaming response
			if output.StreamChannel != nil {
				streamOutput := &ports.GenerateResponseOutput{
					Message:       output.NewMessage,
					StreamChannel: output.StreamChannel,
				}
				// skipStartAnswer=false because RegenerateResponseUseCase doesn't use a Notifier
				_, err = d.processStreamingResponse(genCtx, targetMessage.ConversationID, streamOutput, false)
				if err != nil {
					log.Printf("Error processing streaming response: %v", err)
				}
			} else if output.NewMessage != nil {
				// Non-streaming response
				assistantMsg := &protocol.AssistantMessage{
					ID:             output.NewMessage.ID,
					PreviousID:     output.NewMessage.PreviousID,
					ConversationID: output.NewMessage.ConversationID,
					Content:        output.NewMessage.Contents,
				}

				responseEnvelope := &protocol.Envelope{
					ConversationID: targetMessage.ConversationID,
					Type:           protocol.TypeAssistantMessage,
					Body:           assistantMsg,
				}

				if err := d.protocolHandler.SendEnvelope(genCtx, responseEnvelope); err != nil {
					log.Printf("Failed to send regenerated message: %v", err)
				}
			}
		}()

		// Send acknowledgement
		return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
	}

	// Fallback to old behavior if use case not available
	// Delete the existing assistant message and related data
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
		conversationID := targetMessage.ConversationID
		previousID := targetMessage.PreviousID

		go func() {
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Create a notifier to send real-time progress updates to the client
			notifier := NewProtocolNotifier(genCtx, d.protocolHandler, d.idGenerator)

			input := &ports.GenerateResponseInput{
				ConversationID:  conversationID,
				UserMessageID:   previousID,
				MessageID:       assistantMsgID,
				EnableTools:     true,
				EnableReasoning: true,
				EnableStreaming: true,
				PreviousID:      previousID,
				Notifier:        notifier,
			}

			d.generationManager.RegisterGeneration(assistantMsgID, cancel)
			defer d.generationManager.UnregisterGeneration(assistantMsgID)

			output, err := d.generateResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to regenerate response: %v", err)
				if genCtx.Err() == nil {
					_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
						fmt.Sprintf("Failed to regenerate response: %v", err), true)
				}
				return
			}

			log.Printf("Regenerated response for user message: %s", previousID)

			if output.StreamChannel != nil {
				// skipStartAnswer=true because we use a Notifier which already sent StartAnswer
				_, _ = d.processStreamingResponse(genCtx, conversationID, output, true)
			} else if output.Message != nil {
				d.sendNonStreamingResponse(genCtx, conversationID, output)
			}
		}()
	}

	// Send acknowledgement
	return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
}

// handleAssistantEdit updates an existing assistant message with new content (no regeneration)
func (d *DefaultMessageDispatcher) handleAssistantEdit(ctx context.Context, envelope *protocol.Envelope, targetMessage *models.Message, newContent string) error {
	log.Printf("Editing assistant message: %s", targetMessage.ID)

	// Validate that new content is provided
	if newContent == "" {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			"NewContent is required for edit variation", true)
	}

	// First, cancel any active generation for this message
	_ = d.generationManager.CancelGeneration(targetMessage.ID)

	// Use EditAssistantMessageUseCase if available
	if d.editAssistantMessageUseCase != nil {
		input := &ports.EditAssistantMessageInput{
			ConversationID:  targetMessage.ConversationID,
			TargetMessageID: targetMessage.ID,
			NewContent:      newContent,
		}

		output, err := d.editAssistantMessageUseCase.Execute(ctx, input)
		if err != nil {
			log.Printf("Failed to edit assistant message: %v", err)
			return d.sendError(ctx, protocol.ErrCodeInternalError,
				fmt.Sprintf("Failed to edit assistant message: %v", err), true)
		}

		log.Printf("Edited assistant message: %s", output.UpdatedMessage.ID)

		// Send updated AssistantMessage back to the client
		assistantMsg := &protocol.AssistantMessage{
			ID:             output.UpdatedMessage.ID,
			PreviousID:     output.UpdatedMessage.PreviousID,
			ConversationID: output.UpdatedMessage.ConversationID,
			Content:        output.UpdatedMessage.Contents,
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

		log.Printf("Sent updated assistant message to client: %s", output.UpdatedMessage.ID)

		// Send acknowledgement
		return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
	}

	// Fallback to old behavior if use case not available
	// Update the message content
	targetMessage.Contents = newContent
	targetMessage.UpdatedAt = time.Now()
	if err := d.messageRepo.Update(ctx, targetMessage); err != nil {
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to update message: %v", err), true)
	}

	log.Printf("Updated assistant message content: %s", targetMessage.ID)

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

	log.Printf("Sent updated assistant message to client: %s", targetMessage.ID)

	// Send acknowledgement
	return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
}

// handleUserEdit updates a user message and triggers a new assistant response
func (d *DefaultMessageDispatcher) handleUserEdit(ctx context.Context, envelope *protocol.Envelope, targetMessage *models.Message, newContent string) error {
	log.Printf("Editing user message: %s", targetMessage.ID)

	// Validate that new content is provided
	if newContent == "" {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			"NewContent is required for user message edit", true)
	}

	// Cancel any ongoing generation
	d.generationManager.CancelAll()

	// Use EditUserMessageUseCase if available
	if d.editUserMessageUseCase != nil {
		input := &ports.EditUserMessageInput{
			ConversationID:  targetMessage.ConversationID,
			TargetMessageID: targetMessage.ID,
			NewContent:      newContent,
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: true,
		}

		// Send acknowledgement first
		if err := d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true); err != nil {
			log.Printf("Failed to send acknowledgement: %v", err)
			return err
		}

		// Execute asynchronously
		go func() {
			// 5 minute timeout for LLM generation
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			output, err := d.editUserMessageUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to edit user message: %v", err)
				if genCtx.Err() == nil {
					_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
						fmt.Sprintf("Failed to edit user message: %v", err), true)
				}
				return
			}

			log.Printf("Edited user message: %s, deleted %d subsequent messages",
				output.UpdatedMessage.ID, output.DeletedCount)

			// Send updated UserMessage back to client
			userMsg := &protocol.UserMessage{
				ID:             output.UpdatedMessage.ID,
				ConversationID: output.UpdatedMessage.ConversationID,
				Content:        output.UpdatedMessage.Contents,
				PreviousID:     output.UpdatedMessage.PreviousID,
			}
			userEnvelope := &protocol.Envelope{
				ConversationID: targetMessage.ConversationID,
				Type:           protocol.TypeUserMessage,
				Body:           userMsg,
			}
			if err := d.protocolHandler.SendEnvelope(genCtx, userEnvelope); err != nil {
				log.Printf("Failed to send updated user message: %v", err)
			}

			// Send BranchUpdate notification so frontend can update the branch navigator
			d.sendBranchUpdate(genCtx, output.UpdatedMessage)

			// Send memory traces for retrieved memories
			if len(output.RelevantMemories) > 0 {
				d.sendMemoryTraces(genCtx, output.UpdatedMessage.ID, output.RelevantMemories)
			}

			// Handle streaming response
			if output.StreamChannel != nil {
				streamOutput := &ports.GenerateResponseOutput{
					Message:       output.AssistantMessage,
					StreamChannel: output.StreamChannel,
				}
				// skipStartAnswer=false because EditUserMessageUseCase doesn't use a Notifier
				_, err = d.processStreamingResponse(genCtx, targetMessage.ConversationID, streamOutput, false)
				if err != nil {
					log.Printf("Error processing streaming response: %v", err)
				}
			} else if output.AssistantMessage != nil {
				// Non-streaming response
				streamOutput := &ports.GenerateResponseOutput{
					Message: output.AssistantMessage,
				}
				d.sendNonStreamingResponse(genCtx, targetMessage.ConversationID, streamOutput)
			}

			log.Printf("Completed handling edited user message: %s", targetMessage.ID)
		}()

		return nil
	}

	// Fallback to old behavior if use case not available
	// Update the user message content
	targetMessage.Contents = newContent
	targetMessage.UpdatedAt = time.Now()
	if err := d.messageRepo.Update(ctx, targetMessage); err != nil {
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to update user message: %v", err), true)
	}

	log.Printf("Updated user message content: %s", targetMessage.ID)

	// Delete all messages that came AFTER this user message
	if err := d.messageRepo.DeleteAfterSequence(ctx, targetMessage.ConversationID, targetMessage.SequenceNumber); err != nil {
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to clear subsequent messages: %v", err), true)
	}

	log.Printf("Deleted messages after sequence %d in conversation %s", targetMessage.SequenceNumber, targetMessage.ConversationID)

	// Send acknowledgement for the edit
	if err := d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true); err != nil {
		log.Printf("Failed to send acknowledgement: %v", err)
		return err
	}

	// Send updated UserMessage back to client
	userMsg := &protocol.UserMessage{
		ID:             targetMessage.ID,
		ConversationID: targetMessage.ConversationID,
		Content:        newContent,
		PreviousID:     targetMessage.PreviousID,
	}
	userEnvelope := &protocol.Envelope{
		ConversationID: targetMessage.ConversationID,
		Type:           protocol.TypeUserMessage,
		Body:           userMsg,
	}
	if err := d.protocolHandler.SendEnvelope(ctx, userEnvelope); err != nil {
		log.Printf("Failed to send updated user message: %v", err)
	}

	// Trigger new assistant response generation
	processOutput := &ports.ProcessUserMessageOutput{
		Message:          targetMessage,
		RelevantMemories: []*models.Memory{},
	}

	if d.processUserMessageUseCase != nil {
		processInput := &ports.ProcessUserMessageInput{
			ConversationID: targetMessage.ConversationID,
			TextContent:    newContent,
			PreviousID:     targetMessage.PreviousID,
		}
		output, err := d.processUserMessageUseCase.Execute(ctx, processInput)
		if err != nil {
			log.Printf("Failed to retrieve memories for edited message: %v", err)
		} else {
			processOutput.RelevantMemories = output.RelevantMemories
			d.sendMemoryTraces(ctx, targetMessage.ID, output.RelevantMemories)
		}
	}

	log.Printf("Triggering assistant response for edited user message: %s", targetMessage.ID)
	d.generateResponseAsync(ctx, targetMessage.ConversationID, targetMessage.ID, processOutput)

	return nil
}

// handleContinue extends an existing message
func (d *DefaultMessageDispatcher) handleContinue(ctx context.Context, envelope *protocol.Envelope, targetMessage *models.Message) error {
	log.Printf("Continuing message: %s", targetMessage.ID)

	// Get the user message that prompted this response
	if targetMessage.PreviousID == "" {
		return d.sendError(ctx, protocol.ErrCodeInvalidState,
			"Cannot continue: target message has no previous message reference", true)
	}

	// Use ContinueResponseUseCase if available
	if d.continueResponseUseCase != nil {
		input := &ports.ContinueResponseInput{
			TargetMessageID: targetMessage.ID,
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: true,
		}

		// Execute asynchronously
		go func() {
			// 5 minute timeout for LLM generation
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			output, err := d.continueResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to continue response: %v", err)
				if genCtx.Err() == nil {
					_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
						fmt.Sprintf("Failed to continue response: %v", err), true)
				}
				return
			}

			log.Printf("Continued response for message: %s, appended content length: %d",
				targetMessage.ID, len(output.AppendedContent))

			// Handle streaming response
			if output.StreamChannel != nil {
				streamOutput := &ports.GenerateResponseOutput{
					Message:       output.TargetMessage,
					StreamChannel: output.StreamChannel,
				}
				// skipStartAnswer=false because ContinueResponseUseCase doesn't use a Notifier
				_, err = d.processStreamingResponse(genCtx, targetMessage.ConversationID, streamOutput, false)
				if err != nil {
					log.Printf("Error processing streaming response: %v", err)
				}
			} else if output.TargetMessage != nil {
				// Non-streaming response - send the updated message
				assistantMsg := &protocol.AssistantMessage{
					ID:             output.TargetMessage.ID,
					PreviousID:     output.TargetMessage.PreviousID,
					ConversationID: output.TargetMessage.ConversationID,
					Content:        output.TargetMessage.Contents,
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

		// Send acknowledgement
		return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
	}

	// Fallback to old behavior if use case not available
	if d.generateResponseUseCase != nil {
		continuationMsgID := d.idGenerator.GenerateMessageID()
		conversationID := targetMessage.ConversationID
		previousID := targetMessage.PreviousID
		targetMsgID := targetMessage.ID

		go func() {
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Create a notifier to send real-time progress updates to the client
			notifier := NewProtocolNotifier(genCtx, d.protocolHandler, d.idGenerator)

			input := &ports.GenerateResponseInput{
				ConversationID:  conversationID,
				UserMessageID:   previousID,
				MessageID:       continuationMsgID,
				EnableTools:     true,
				EnableReasoning: true,
				EnableStreaming: true,
				PreviousID:      targetMsgID,
				Notifier:        notifier,
			}

			d.generationManager.RegisterGeneration(continuationMsgID, cancel)
			defer d.generationManager.UnregisterGeneration(continuationMsgID)

			output, err := d.generateResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to continue response: %v", err)
				if genCtx.Err() == nil {
					_ = d.sendError(genCtx, protocol.ErrCodeInternalError,
						fmt.Sprintf("Failed to continue response: %v", err), true)
				}
				return
			}

			log.Printf("Continued response for message: %s", targetMsgID)

			if output.Message != nil {
				targetMessage.Contents += "\n\n" + output.Message.Contents
				targetMessage.UpdatedAt = time.Now()

				if err := d.messageRepo.Update(genCtx, targetMessage); err != nil {
					log.Printf("Failed to update continued message: %v", err)
					return
				}

				assistantMsg := &protocol.AssistantMessage{
					ID:             targetMessage.ID,
					PreviousID:     targetMessage.PreviousID,
					ConversationID: conversationID,
					Content:        targetMessage.Contents,
				}

				responseEnvelope := &protocol.Envelope{
					ConversationID: conversationID,
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
	return d.protocolHandler.SendAcknowledgement(ctx, envelope.StanzaID, true)
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

// sendBranchUpdate sends a BranchUpdate notification when a new sibling message is created.
// This allows the frontend to update the branch navigator UI without requiring a manual refresh.
func (d *DefaultMessageDispatcher) sendBranchUpdate(ctx context.Context, newMessage *models.Message) {
	if newMessage == nil || newMessage.PreviousID == "" {
		// No parent message means no siblings to notify about
		return
	}

	// Fetch all siblings (messages with the same PreviousID)
	siblings, err := d.messageRepo.GetSiblings(ctx, newMessage.ID)
	if err != nil {
		log.Printf("Failed to get siblings for branch update: %v", err)
		return
	}

	// If there's only one message (the new one), no need to send branch update
	// (no other messages exist that need to know about siblings)
	if len(siblings) <= 1 {
		return
	}

	// Convert siblings to protocol format
	allSiblings := make([]protocol.SiblingInfo, 0, len(siblings))
	for _, sibling := range siblings {
		allSiblings = append(allSiblings, protocol.SiblingInfo{
			ID:        sibling.ID,
			Content:   sibling.Contents,
			CreatedAt: sibling.CreatedAt.Format(time.RFC3339),
		})
	}

	// Create the BranchUpdate message
	branchUpdate := &protocol.BranchUpdate{
		ConversationID:  newMessage.ConversationID,
		ParentMessageID: newMessage.PreviousID,
		NewSibling: protocol.SiblingInfo{
			ID:        newMessage.ID,
			Content:   newMessage.Contents,
			CreatedAt: newMessage.CreatedAt.Format(time.RFC3339),
		},
		AllSiblings: allSiblings,
		TotalCount:  len(siblings),
		Timestamp:   time.Now().UnixMilli(),
	}

	// Send via protocol handler
	envelope := &protocol.Envelope{
		ConversationID: newMessage.ConversationID,
		Type:           protocol.TypeBranchUpdate,
		Body:           branchUpdate,
	}

	if err := d.protocolHandler.SendEnvelope(ctx, envelope); err != nil {
		log.Printf("Failed to send branch update: %v", err)
		// Non-fatal - continue even if branch update fails
		return
	}

	log.Printf("Sent branch update: conversation=%s, parent=%s, newSibling=%s, totalSiblings=%d",
		newMessage.ConversationID, newMessage.PreviousID, newMessage.ID, len(siblings))
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
		msg := fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, result.ConversationID)
		_ = d.sendError(ctx, protocol.ErrCodeConversationNotFound, msg, true)
		return fmt.Errorf("%s", msg)
	}

	// Fetch the ToolUse from the repository
	toolUse, err := d.toolUseRepo.GetByID(ctx, result.RequestID)
	if err != nil || toolUse == nil {
		msg := fmt.Sprintf("Failed to find tool use with ID %s: %v", result.RequestID, err)
		_ = d.sendError(ctx, protocol.ErrCodeInternalError, msg, true)
		return fmt.Errorf("%s", msg)
	}

	// Update the ToolUse with the result
	if result.Success {
		toolUse.Complete(result.Result)
		log.Printf("Tool execution succeeded for request %s", result.RequestID)
	} else {
		errorMsg := result.ErrorMessage
		if result.ErrorCode != "" {
			errorMsg = fmt.Sprintf("%s: %s", result.ErrorCode, result.ErrorMessage)
		}
		toolUse.Fail(errorMsg)
		log.Printf("Tool execution failed for request %s: %s - %s",
			result.RequestID, result.ErrorCode, result.ErrorMessage)
	}

	// Persist the updated ToolUse
	if err := d.toolUseRepo.Update(ctx, toolUse); err != nil {
		log.Printf("Failed to update tool use %s: %v", result.RequestID, err)
		return d.sendError(ctx, protocol.ErrCodeInternalError,
			fmt.Sprintf("Failed to store tool result: %v", err),
			true)
	}

	log.Printf("Stored tool result for %s (status: %s)", result.RequestID, toolUse.Status)

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

// handleFeedback processes Feedback messages (votes on messages, tools, memories, reasoning)
func (d *DefaultMessageDispatcher) handleFeedback(ctx context.Context, envelope *protocol.Envelope) error {
	feedback, ok := envelope.Body.(*protocol.Feedback)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid Feedback message", true)
		return fmt.Errorf("invalid Feedback message type")
	}

	log.Printf("Received feedback: targetType=%s, targetID=%s, vote=%s",
		feedback.TargetType, feedback.TargetID, feedback.Vote)

	// Check if voteRepo is available
	if d.voteRepo == nil {
		log.Printf("Warning: voteRepo not available, cannot process feedback")
		return d.sendFeedbackConfirmation(ctx, feedback, nil)
	}

	// Map protocol vote string to model value
	var voteValue int
	switch feedback.Vote {
	case "up":
		voteValue = models.VoteValueUp
	case "down":
		voteValue = models.VoteValueDown
	case "remove":
		// Handle vote removal
		if err := d.voteRepo.Delete(ctx, feedback.TargetType, feedback.TargetID); err != nil {
			log.Printf("Failed to remove vote: %v", err)
		}
		return d.sendFeedbackConfirmation(ctx, feedback, nil)
	default:
		// For special votes like "critical", treat as downvote with metadata
		voteValue = models.VoteValueDown
	}

	// Create the vote
	vote := models.NewVote(
		d.idGenerator.GenerateVoteID(),
		feedback.TargetType,
		feedback.TargetID,
		feedback.MessageID,
		voteValue,
	)

	// Save the vote
	if err := d.voteRepo.Create(ctx, vote); err != nil {
		log.Printf("Failed to create vote: %v", err)
		// Don't fail the whole request, just log the error
	}

	// Get updated aggregates for confirmation
	aggregates, err := d.voteRepo.GetAggregates(ctx, feedback.TargetType, feedback.TargetID)
	if err != nil {
		log.Printf("Failed to get vote aggregates: %v", err)
		// Still send confirmation with zero aggregates
	}

	return d.sendFeedbackConfirmation(ctx, feedback, aggregates)
}

// sendFeedbackConfirmation sends a FeedbackConfirmation message back to the client
func (d *DefaultMessageDispatcher) sendFeedbackConfirmation(ctx context.Context, feedback *protocol.Feedback, aggregates *models.VoteAggregates) error {
	confirmation := &protocol.FeedbackConfirmation{
		FeedbackID: feedback.ID,
		TargetType: feedback.TargetType,
		TargetID:   feedback.TargetID,
		UserVote:   feedback.Vote,
		Aggregates: protocol.FeedbackAggregates{
			Upvotes:   0,
			Downvotes: 0,
		},
	}

	if aggregates != nil {
		confirmation.Aggregates.Upvotes = aggregates.Upvotes
		confirmation.Aggregates.Downvotes = aggregates.Downvotes
	}

	envelope := &protocol.Envelope{
		ConversationID: d.conversationID,
		Type:           protocol.TypeFeedbackConfirmation,
		Body:           confirmation,
	}

	return d.protocolHandler.SendEnvelope(ctx, envelope)
}

// handleUserNote processes UserNote messages (create, update, delete notes)
func (d *DefaultMessageDispatcher) handleUserNote(ctx context.Context, envelope *protocol.Envelope) error {
	note, ok := envelope.Body.(*protocol.UserNote)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid UserNote message", true)
		return fmt.Errorf("invalid UserNote message type")
	}

	log.Printf("Received user note: messageID=%s, action=%s, category=%s",
		note.MessageID, note.Action, note.Category)

	// Check if noteRepo is available
	if d.noteRepo == nil {
		log.Printf("Warning: noteRepo not available, cannot process user note")
		return d.sendNoteConfirmation(ctx, note, false)
	}

	var success bool
	var noteID string

	switch note.Action {
	case "create":
		noteID = d.idGenerator.GenerateNoteID()
		category := note.Category
		if category == "" {
			category = models.NoteCategoryGeneral
		}
		modelNote := models.NewNote(noteID, note.MessageID, note.Content, category)
		if err := d.noteRepo.Create(ctx, modelNote); err != nil {
			log.Printf("Failed to create note: %v", err)
		} else {
			success = true
		}
	case "update":
		noteID = note.ID
		if err := d.noteRepo.Update(ctx, note.ID, note.Content); err != nil {
			log.Printf("Failed to update note: %v", err)
		} else {
			success = true
		}
	case "delete":
		noteID = note.ID
		if err := d.noteRepo.Delete(ctx, note.ID); err != nil {
			log.Printf("Failed to delete note: %v", err)
		} else {
			success = true
		}
	default:
		log.Printf("Unknown note action: %s", note.Action)
	}

	return d.sendNoteConfirmation(ctx, &protocol.UserNote{ID: noteID, MessageID: note.MessageID}, success)
}

// sendNoteConfirmation sends a NoteConfirmation message back to the client
func (d *DefaultMessageDispatcher) sendNoteConfirmation(ctx context.Context, note *protocol.UserNote, success bool) error {
	confirmation := &protocol.NoteConfirmation{
		NoteID:    note.ID,
		MessageID: note.MessageID,
		Success:   success,
	}

	envelope := &protocol.Envelope{
		ConversationID: d.conversationID,
		Type:           protocol.TypeNoteConfirmation,
		Body:           confirmation,
	}

	return d.protocolHandler.SendEnvelope(ctx, envelope)
}

// handleMemoryAction processes MemoryAction messages (create, update, delete, pin, archive memories)
func (d *DefaultMessageDispatcher) handleMemoryAction(ctx context.Context, envelope *protocol.Envelope) error {
	memAction, ok := envelope.Body.(*protocol.MemoryAction)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid MemoryAction message", true)
		return fmt.Errorf("invalid MemoryAction message type")
	}

	log.Printf("Received memory action: id=%s, action=%s", memAction.ID, memAction.Action)

	// Check if memoryService is available
	if d.memoryService == nil {
		log.Printf("Warning: memoryService not available, cannot process memory action")
		return d.sendMemoryConfirmation(ctx, memAction.ID, memAction.Action, false)
	}

	var success bool
	memoryID := memAction.ID

	switch memAction.Action {
	case "create":
		if memAction.Memory != nil {
			memory, err := d.memoryService.CreateWithEmbeddings(ctx, memAction.Memory.Content)
			if err != nil {
				log.Printf("Failed to create memory: %v", err)
			} else {
				memoryID = memory.ID
				success = true
			}
		}
	case "update":
		memory, err := d.memoryService.GetByID(ctx, memAction.ID)
		if err != nil {
			log.Printf("Failed to get memory for update: %v", err)
		} else if memAction.Memory != nil {
			memory.Content = memAction.Memory.Content
			if err := d.memoryService.Update(ctx, memory); err != nil {
				log.Printf("Failed to update memory: %v", err)
			} else {
				success = true
			}
		}
	case "delete":
		if err := d.memoryService.Delete(ctx, memAction.ID); err != nil {
			log.Printf("Failed to delete memory: %v", err)
		} else {
			success = true
		}
	case "pin":
		// Set high importance for pinned memories
		if _, err := d.memoryService.SetImportance(ctx, memAction.ID, 1.0); err != nil {
			log.Printf("Failed to pin memory: %v", err)
		} else {
			success = true
		}
	case "archive":
		// Set low importance for archived memories
		if _, err := d.memoryService.SetImportance(ctx, memAction.ID, 0.1); err != nil {
			log.Printf("Failed to archive memory: %v", err)
		} else {
			success = true
		}
	default:
		log.Printf("Unknown memory action: %s", memAction.Action)
	}

	return d.sendMemoryConfirmation(ctx, memoryID, memAction.Action, success)
}

// sendMemoryConfirmation sends a MemoryConfirmation message back to the client
func (d *DefaultMessageDispatcher) sendMemoryConfirmation(ctx context.Context, memoryID, action string, success bool) error {
	confirmation := &protocol.MemoryConfirmation{
		MemoryID: memoryID,
		Action:   action,
		Success:  success,
	}

	envelope := &protocol.Envelope{
		ConversationID: d.conversationID,
		Type:           protocol.TypeMemoryConfirmation,
		Body:           confirmation,
	}

	return d.protocolHandler.SendEnvelope(ctx, envelope)
}

// handleDimensionPreference processes DimensionPreference messages (adjust optimization weights)
func (d *DefaultMessageDispatcher) handleDimensionPreference(ctx context.Context, envelope *protocol.Envelope) error {
	pref, ok := envelope.Body.(*protocol.DimensionPreference)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid DimensionPreference message", true)
		return fmt.Errorf("invalid DimensionPreference message type")
	}

	log.Printf("Received dimension preference: conversationID=%s, preset=%s",
		pref.ConversationID, pref.Preset)

	// Validate conversation ID
	if pref.ConversationID != d.conversationID {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, pref.ConversationID),
			true)
	}

	// Check if optimizationService is available
	if d.optimizationService == nil {
		log.Printf("Warning: optimizationService not available, cannot process dimension preference")
		return nil
	}

	// Convert protocol weights to map for interface compatibility
	weights := map[string]float64{
		"successRate":    pref.Weights.SuccessRate,
		"quality":        pref.Weights.Quality,
		"efficiency":     pref.Weights.Efficiency,
		"robustness":     pref.Weights.Robustness,
		"generalization": pref.Weights.Generalization,
		"diversity":      pref.Weights.Diversity,
		"innovation":     pref.Weights.Innovation,
	}

	// Apply the dimension weights to the optimization service
	d.optimizationService.SetDimensionWeights(weights)

	log.Printf("Applied dimension weights: successRate=%.2f, quality=%.2f, efficiency=%.2f",
		pref.Weights.SuccessRate, pref.Weights.Quality, pref.Weights.Efficiency)

	return nil
}

// handleEliteSelect processes EliteSelect messages (select an elite solution)
func (d *DefaultMessageDispatcher) handleEliteSelect(ctx context.Context, envelope *protocol.Envelope) error {
	selection, ok := envelope.Body.(*protocol.EliteSelect)
	if !ok {
		_ = d.sendError(ctx, protocol.ErrCodeMalformedData, "Invalid EliteSelect message", true)
		return fmt.Errorf("invalid EliteSelect message type")
	}

	log.Printf("Received elite selection: conversationID=%s, eliteID=%s",
		selection.ConversationID, selection.EliteID)

	// Validate conversation ID
	if selection.ConversationID != d.conversationID {
		return d.sendError(ctx, protocol.ErrCodeConversationNotFound,
			fmt.Sprintf("Conversation ID mismatch: expected %s, got %s", d.conversationID, selection.ConversationID),
			true)
	}

	// Validate elite ID is provided
	if selection.EliteID == "" {
		return d.sendError(ctx, protocol.ErrCodeMalformedData,
			"EliteID is required for elite selection", true)
	}

	// For now, just log the selection
	// In the future, this could:
	// - Update the active elite solution for the conversation
	// - Apply the elite's dimension weights
	// - Store the selection for tracking user preferences
	log.Printf("User selected elite solution: %s", selection.EliteID)

	return nil
}

// SendServerInfo sends server information to the client
// This should be called on initial connection and periodically for status updates
func (d *DefaultMessageDispatcher) SendServerInfo(ctx context.Context) error {
	serverInfo := &protocol.ServerInfo{
		Connection: protocol.ConnectionInfo{
			Status:  "connected",
			Latency: 0, // Will be updated with actual latency when available
		},
		Model: protocol.ModelInfo{
			Name:     "claude-3-5-sonnet", // Default model name
			Provider: "anthropic",
		},
		MCPServers: []protocol.MCPServerInfo{},
	}

	envelope := &protocol.Envelope{
		ConversationID: d.conversationID,
		Type:           protocol.TypeServerInfo,
		Body:           serverInfo,
	}

	return d.protocolHandler.SendEnvelope(ctx, envelope)
}

// SendSessionStats sends session statistics to the client
func (d *DefaultMessageDispatcher) SendSessionStats(ctx context.Context) error {
	// Get message count for this conversation
	var messageCount int
	if d.messageRepo != nil {
		messages, err := d.messageRepo.GetByConversation(ctx, d.conversationID)
		if err == nil {
			messageCount = len(messages)
		}
	}

	// Get tool call count by querying all messages and their tool uses
	var toolCallCount int
	if d.messageRepo != nil && d.toolUseRepo != nil {
		messages, err := d.messageRepo.GetByConversation(ctx, d.conversationID)
		if err == nil {
			for _, msg := range messages {
				toolUses, err := d.toolUseRepo.GetByMessage(ctx, msg.ID)
				if err == nil {
					toolCallCount += len(toolUses)
				}
			}
		}
	}

	// Get memory usage count for this conversation
	var memoriesUsed int
	if d.memoryUsageRepo != nil {
		memoryUsages, err := d.memoryUsageRepo.GetByConversation(ctx, d.conversationID)
		if err == nil {
			memoriesUsed = len(memoryUsages)
		}
	}

	// Calculate session duration from conversation creation time
	var sessionDuration int
	if d.conversationRepo != nil {
		conversation, err := d.conversationRepo.GetByID(ctx, d.conversationID)
		if err == nil {
			sessionDuration = int(time.Since(conversation.CreatedAt).Milliseconds())
		}
	}

	stats := &protocol.SessionStats{
		MessageCount:    messageCount,
		ToolCallCount:   toolCallCount,
		MemoriesUsed:    memoriesUsed,
		SessionDuration: sessionDuration,
	}

	envelope := &protocol.Envelope{
		ConversationID: d.conversationID,
		Type:           protocol.TypeSessionStats,
		Body:           stats,
	}

	return d.protocolHandler.SendEnvelope(ctx, envelope)
}

// SendEliteOptions sends available elite solutions to the client
func (d *DefaultMessageDispatcher) SendEliteOptions(ctx context.Context) error {
	// Elite solutions are populated when optimization runs complete
	// For now, send an empty list - UI will show "no elites available" state
	// When optimization service has completed runs, this can be enhanced
	// to query the most recent optimization run and extract its elites
	eliteOptions := &protocol.EliteOptions{
		ConversationID: d.conversationID,
		Elites:         []protocol.EliteSummary{},
		CurrentEliteID: "", // Will be set when user selects one
		Timestamp:      time.Now().UnixMilli(),
	}

	envelope := &protocol.Envelope{
		ConversationID: d.conversationID,
		Type:           protocol.TypeEliteOptions,
		Body:           eliteOptions,
	}

	return d.protocolHandler.SendEnvelope(ctx, envelope)
}

// handleResponseGenerationRequest handles ResponseGenerationRequest messages from the serve process
// This is called when the agent receives a request to generate a response for a user message
func (d *DefaultMessageDispatcher) handleResponseGenerationRequest(ctx context.Context, envelope *protocol.Envelope) error {
	req, ok := envelope.Body.(*protocol.ResponseGenerationRequest)
	if !ok {
		// Try to decode from map if body is a generic interface
		bodyMap, isMap := envelope.Body.(map[string]interface{})
		if !isMap {
			log.Printf("Invalid ResponseGenerationRequest message type: %T", envelope.Body)
			return fmt.Errorf("invalid ResponseGenerationRequest message type")
		}

		req = &protocol.ResponseGenerationRequest{}
		if id, ok := bodyMap["id"].(string); ok {
			req.ID = id
		}
		if messageID, ok := bodyMap["messageId"].(string); ok {
			req.MessageID = messageID
		}
		if conversationID, ok := bodyMap["conversationId"].(string); ok {
			req.ConversationID = conversationID
		}
		if requestType, ok := bodyMap["requestType"].(string); ok {
			req.RequestType = requestType
		}
		if enableTools, ok := bodyMap["enableTools"].(bool); ok {
			req.EnableTools = enableTools
		}
		if enableReasoning, ok := bodyMap["enableReasoning"].(bool); ok {
			req.EnableReasoning = enableReasoning
		}
		if enableStreaming, ok := bodyMap["enableStreaming"].(bool); ok {
			req.EnableStreaming = enableStreaming
		}
		if previousID, ok := bodyMap["previousId"].(string); ok {
			req.PreviousID = previousID
		}
	}

	log.Printf("Received ResponseGenerationRequest (type: %s, messageID: %s, conversationID: %s)",
		req.RequestType, req.MessageID, req.ConversationID)

	// Execute based on request type
	switch req.RequestType {
	case "send":
		return d.handleGenerateForUserMessage(ctx, req)
	case "regenerate":
		return d.handleRegenerateFromRequest(ctx, req)
	case "continue":
		return d.handleContinueFromRequest(ctx, req)
	case "edit":
		return d.handleGenerateForEditedMessage(ctx, req)
	default:
		log.Printf("Unknown request type: %s", req.RequestType)
		return fmt.Errorf("unknown request type: %s", req.RequestType)
	}
}

// handleGenerateForUserMessage generates a response for a user message
func (d *DefaultMessageDispatcher) handleGenerateForUserMessage(ctx context.Context, req *protocol.ResponseGenerationRequest) error {
	if d.generateResponseUseCase == nil {
		return fmt.Errorf("generateResponseUseCase not available")
	}

	// Execute response generation asynchronously
	go func() {
		// 5 minute timeout for LLM generation
		genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		input := &ports.GenerateResponseInput{
			ConversationID:  req.ConversationID,
			UserMessageID:   req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
			Notifier:        d.protocolHandler.(ports.GenerationNotifier),
		}

		output, err := d.generateResponseUseCase.Execute(genCtx, input)
		if err != nil {
			log.Printf("Failed to generate response for message %s: %v", req.MessageID, err)
			// Notify failure via protocol handler
			errMsg := &protocol.ErrorMessage{
				ConversationID: req.ConversationID,
				Code:           protocol.ErrCodeInternalError,
				Message:        err.Error(),
				Severity:       protocol.SeverityError,
				Recoverable:    true,
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeErrorMessage,
				Body:           errMsg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
			return
		}

		// If not streaming, send the complete response via AssistantMessage
		if output.Message != nil && output.Message.Contents != "" && !req.EnableStreaming {
			msg := &protocol.AssistantMessage{
				ID:             output.Message.ID,
				PreviousID:     req.MessageID,
				ConversationID: req.ConversationID,
				Content:        output.Message.Contents,
				Timestamp:      time.Now().UnixMilli(),
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeAssistantMessage,
				Body:           msg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
		}
	}()

	return nil
}

// handleRegenerateFromRequest regenerates a response for a message
func (d *DefaultMessageDispatcher) handleRegenerateFromRequest(ctx context.Context, req *protocol.ResponseGenerationRequest) error {
	if d.regenerateResponseUseCase == nil {
		return fmt.Errorf("regenerateResponseUseCase not available")
	}

	// Execute regeneration asynchronously
	go func() {
		genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		input := &ports.RegenerateResponseInput{
			MessageID:       req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
		}

		output, err := d.regenerateResponseUseCase.Execute(genCtx, input)
		if err != nil {
			log.Printf("Failed to regenerate response for message %s: %v", req.MessageID, err)
			errMsg := &protocol.ErrorMessage{
				ConversationID: req.ConversationID,
				Code:           protocol.ErrCodeInternalError,
				Message:        err.Error(),
				Severity:       protocol.SeverityError,
				Recoverable:    true,
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeErrorMessage,
				Body:           errMsg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
			return
		}

		// Send complete response if not streaming
		if output.NewMessage != nil && output.NewMessage.Contents != "" && !req.EnableStreaming {
			msg := &protocol.AssistantMessage{
				ID:             output.NewMessage.ID,
				ConversationID: req.ConversationID,
				Content:        output.NewMessage.Contents,
				Timestamp:      time.Now().UnixMilli(),
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeAssistantMessage,
				Body:           msg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
		}
	}()

	return nil
}

// handleContinueFromRequest continues a response for a message
func (d *DefaultMessageDispatcher) handleContinueFromRequest(ctx context.Context, req *protocol.ResponseGenerationRequest) error {
	if d.continueResponseUseCase == nil {
		return fmt.Errorf("continueResponseUseCase not available")
	}

	// Execute continuation asynchronously
	go func() {
		genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		input := &ports.ContinueResponseInput{
			TargetMessageID: req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
		}

		output, err := d.continueResponseUseCase.Execute(genCtx, input)
		if err != nil {
			log.Printf("Failed to continue response for message %s: %v", req.MessageID, err)
			errMsg := &protocol.ErrorMessage{
				ConversationID: req.ConversationID,
				Code:           protocol.ErrCodeInternalError,
				Message:        err.Error(),
				Severity:       protocol.SeverityError,
				Recoverable:    true,
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeErrorMessage,
				Body:           errMsg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
			return
		}

		// Send complete response if not streaming
		if output.TargetMessage != nil && output.AppendedContent != "" && !req.EnableStreaming {
			msg := &protocol.AssistantMessage{
				ID:             output.TargetMessage.ID,
				ConversationID: req.ConversationID,
				Content:        output.TargetMessage.Contents, // Updated message content
				Timestamp:      time.Now().UnixMilli(),
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeAssistantMessage,
				Body:           msg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
		}
	}()

	return nil
}

// handleGenerateForEditedMessage generates a response for an edited user message
func (d *DefaultMessageDispatcher) handleGenerateForEditedMessage(ctx context.Context, req *protocol.ResponseGenerationRequest) error {
	if d.generateResponseUseCase == nil {
		return fmt.Errorf("generateResponseUseCase not available")
	}

	// Execute response generation asynchronously (same as handleGenerateForUserMessage)
	// The message has already been updated by the HTTP handler
	go func() {
		genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		input := &ports.GenerateResponseInput{
			ConversationID:  req.ConversationID,
			UserMessageID:   req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
			Notifier:        d.protocolHandler.(ports.GenerationNotifier),
		}

		output, err := d.generateResponseUseCase.Execute(genCtx, input)
		if err != nil {
			log.Printf("Failed to generate response for edited message %s: %v", req.MessageID, err)
			errMsg := &protocol.ErrorMessage{
				ConversationID: req.ConversationID,
				Code:           protocol.ErrCodeInternalError,
				Message:        err.Error(),
				Severity:       protocol.SeverityError,
				Recoverable:    true,
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeErrorMessage,
				Body:           errMsg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
			return
		}

		// If not streaming, send the complete response
		if output.Message != nil && output.Message.Contents != "" && !req.EnableStreaming {
			msg := &protocol.AssistantMessage{
				ID:             output.Message.ID,
				PreviousID:     req.MessageID,
				ConversationID: req.ConversationID,
				Content:        output.Message.Contents,
				Timestamp:      time.Now().UnixMilli(),
			}
			envelope := &protocol.Envelope{
				ConversationID: req.ConversationID,
				Type:           protocol.TypeAssistantMessage,
				Body:           msg,
			}
			_ = d.protocolHandler.SendEnvelope(genCtx, envelope)
		}
	}()

	return nil
}
