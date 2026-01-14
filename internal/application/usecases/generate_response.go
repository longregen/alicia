package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type GenerateResponse struct {
	messageRepo          ports.MessageRepository
	sentenceRepo         ports.SentenceRepository
	toolUseRepo          ports.ToolUseRepository
	toolRepo             ports.ToolRepository
	reasoningStepRepo    ports.ReasoningStepRepository
	conversationRepo     ports.ConversationRepository
	llmService           ports.LLMService
	toolService          ports.ToolService
	memoryService        ports.MemoryService
	promptVersionService ports.PromptVersionService
	idGenerator          ports.IDGenerator
	txManager            ports.TransactionManager
	titleGenerator       *GenerateConversationTitle // Optional: auto-generates conversation titles
}

func NewGenerateResponse(
	messageRepo ports.MessageRepository,
	sentenceRepo ports.SentenceRepository,
	toolUseRepo ports.ToolUseRepository,
	toolRepo ports.ToolRepository,
	reasoningStepRepo ports.ReasoningStepRepository,
	conversationRepo ports.ConversationRepository,
	llmService ports.LLMService,
	toolService ports.ToolService,
	memoryService ports.MemoryService,
	promptVersionService ports.PromptVersionService,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
	broadcaster ports.ConversationBroadcaster,
) *GenerateResponse {
	gr := &GenerateResponse{
		messageRepo:          messageRepo,
		sentenceRepo:         sentenceRepo,
		toolUseRepo:          toolUseRepo,
		toolRepo:             toolRepo,
		reasoningStepRepo:    reasoningStepRepo,
		conversationRepo:     conversationRepo,
		llmService:           llmService,
		toolService:          toolService,
		memoryService:        memoryService,
		promptVersionService: promptVersionService,
		idGenerator:          idGenerator,
		txManager:            txManager,
	}

	// Initialize title generator with shared dependencies
	gr.titleGenerator = NewGenerateConversationTitle(conversationRepo, messageRepo, llmService, broadcaster)

	return gr
}

func (uc *GenerateResponse) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	// Get conversation for prompt version tracking
	conversation, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	messages, err := uc.messageRepo.GetLatestByConversation(ctx, input.ConversationID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Retrieve relevant memories if services are available
	relevantMemories := input.RelevantMemories
	if uc.memoryService != nil {
		searchResults, err := uc.retrieveRelevantMemories(ctx, input.ConversationID, input.UserMessageID)
		if err != nil {
			// Log but don't fail - memory is optional
			log.Printf("warning: failed to retrieve memories: %v\n", err)
		} else if len(searchResults) > 0 {
			// Extract memories from search results
			retrievedMemories := make([]*models.Memory, len(searchResults))
			for i, result := range searchResults {
				retrievedMemories[i] = result.Memory
			}

			// Merge with any memories already provided in input
			relevantMemories = uc.mergeMemories(relevantMemories, retrievedMemories)

			// Track memory usage with actual similarity scores
			uc.trackMemoryUsageWithScores(ctx, input.ConversationID, input.UserMessageID, searchResults)
		}
	}

	llmMessages := uc.buildLLMContext(ctx, conversation, messages, relevantMemories)

	var tools []*models.Tool
	if input.EnableTools {
		tools, err = uc.toolRepo.ListEnabled(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get enabled tools: %w", err)
		}
	}

	sequenceNumber, err := uc.messageRepo.GetNextSequenceNumber(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next sequence number: %w", err)
	}

	// Use pre-generated message ID if provided, otherwise generate a new one
	messageID := input.MessageID
	if messageID == "" {
		messageID = uc.idGenerator.GenerateMessageID()
	}
	message := models.NewAssistantMessage(messageID, input.ConversationID, sequenceNumber, "")
	if input.PreviousID != "" {
		message.SetPreviousMessage(input.PreviousID)
	}

	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Update conversation tip to point to the new assistant message
	if err := uc.conversationRepo.UpdateTip(ctx, input.ConversationID, message.ID); err != nil {
		return nil, fmt.Errorf("failed to update conversation tip: %w", err)
	}

	if input.EnableStreaming {
		return uc.executeStreaming(ctx, input, message, llmMessages, tools)
	}

	return uc.executeNonStreaming(ctx, input, message, llmMessages, tools)
}

func (uc *GenerateResponse) buildLLMContext(ctx context.Context, conversation *models.Conversation, messages []*models.Message, memories []*models.Memory) []ports.LLMMessage {
	llmMessages := []ports.LLMMessage{}

	// Build system prompt
	var systemPrompt string
	if len(memories) > 0 {
		var memoryContent strings.Builder
		memoryContent.WriteString("You are Alicia, a helpful AI assistant. Here are some relevant memories from our previous conversations:\n\n")
		for i, memory := range memories {
			memoryContent.WriteString(fmt.Sprintf("%d. %s\n", i+1, memory.Content))
		}
		systemPrompt = memoryContent.String()
	} else {
		systemPrompt = "You are Alicia, a helpful AI assistant."
	}

	// Track the prompt version if promptVersionService is available
	if uc.promptVersionService != nil && conversation.SystemPromptVersionID == "" {
		versionID, err := uc.promptVersionService.GetOrCreateForConversation(ctx, systemPrompt)
		if err == nil {
			// Update conversation with version ID
			if err := uc.conversationRepo.UpdatePromptVersion(ctx, conversation.ID, versionID); err != nil {
				log.Printf("warning: failed to update conversation prompt version: %v\n", err)
			} else {
				conversation.SystemPromptVersionID = versionID
			}
		} else {
			log.Printf("warning: failed to create prompt version: %v\n", err)
		}
	}

	llmMessages = append(llmMessages, ports.LLMMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	for _, msg := range messages {
		role := string(msg.Role)
		if role == "system" {
			continue // Skip system messages from history
		}
		llmMessages = append(llmMessages, ports.LLMMessage{
			Role:    role,
			Content: msg.Contents,
		})
	}

	return llmMessages
}

func (uc *GenerateResponse) executeNonStreaming(
	ctx context.Context,
	input *ports.GenerateResponseInput,
	message *models.Message,
	llmMessages []ports.LLMMessage,
	tools []*models.Tool,
) (*ports.GenerateResponseOutput, error) {
	output := &ports.GenerateResponseOutput{
		Message:        message,
		Sentences:      []*models.Sentence{},
		ToolUses:       []*models.ToolUse{},
		ReasoningSteps: []*models.ReasoningStep{},
	}

	// If tools are enabled, use the tool execution loop
	if len(tools) > 0 {
		return uc.executeWithToolLoop(ctx, input, message, llmMessages, tools, output)
	}

	// Otherwise, simple non-streaming response
	response, err := uc.llmService.Chat(ctx, llmMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Wrap message update and reasoning steps in a transaction
	err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		message.Contents = strings.TrimSpace(response.Content)
		if err := uc.messageRepo.Update(txCtx, message); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}

		if input.EnableReasoning && response.Reasoning != "" {
			stepID := uc.idGenerator.GenerateReasoningStepID()
			step := &models.ReasoningStep{
				ID:             stepID,
				MessageID:      message.ID,
				SequenceNumber: 0,
				Content:        response.Reasoning,
			}

			if err := uc.reasoningStepRepo.Create(txCtx, step); err != nil {
				return fmt.Errorf("failed to create reasoning step: %w", err)
			}

			output.ReasoningSteps = append(output.ReasoningSteps, step)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Extract and store important memories from the response (async, non-blocking)
	// Use detached context with timeout to avoid being cancelled when parent completes
	go func() {
		memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		uc.extractAndStoreMemories(memCtx, message, input.ConversationID)
	}()

	// Generate conversation title if needed (async, non-blocking)
	uc.titleGenerator.ExecuteAsync(ctx, input.ConversationID)

	return output, nil
}

// executeWithToolLoop handles the tool execution loop for non-streaming mode
func (uc *GenerateResponse) executeWithToolLoop(
	ctx context.Context,
	input *ports.GenerateResponseInput,
	message *models.Message,
	llmMessages []ports.LLMMessage,
	tools []*models.Tool,
	output *ports.GenerateResponseOutput,
) (*ports.GenerateResponseOutput, error) {
	const maxToolIterations = 5
	currentMessages := llmMessages

	for iteration := 0; iteration < maxToolIterations; iteration++ {
		// Call LLM with current message history
		response, err := uc.llmService.ChatWithTools(ctx, currentMessages, tools)
		if err != nil {
			return nil, fmt.Errorf("failed to generate response: %w", err)
		}

		// Add assistant response to message history
		currentMessages = append(currentMessages, ports.LLMMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			// Update message with final content
			err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
				message.Contents = strings.TrimSpace(response.Content)
				if err := uc.messageRepo.Update(txCtx, message); err != nil {
					return fmt.Errorf("failed to update message: %w", err)
				}

				if input.EnableReasoning && response.Reasoning != "" {
					stepID := uc.idGenerator.GenerateReasoningStepID()
					step := &models.ReasoningStep{
						ID:             stepID,
						MessageID:      message.ID,
						SequenceNumber: 0,
						Content:        response.Reasoning,
					}

					if err := uc.reasoningStepRepo.Create(txCtx, step); err != nil {
						return fmt.Errorf("failed to create reasoning step: %w", err)
					}

					output.ReasoningSteps = append(output.ReasoningSteps, step)
				}

				return nil
			})

			if err != nil {
				return nil, err
			}

			// Extract and store important memories from the response (async, non-blocking)
			// Use detached context with timeout to avoid being cancelled when parent completes
			go func() {
				memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
				defer cancel()
				uc.extractAndStoreMemories(memCtx, message, input.ConversationID)
			}()

			// Generate conversation title if needed (async, non-blocking)
			uc.titleGenerator.ExecuteAsync(ctx, input.ConversationID)

			return output, nil
		}

		// Execute each tool call within a transaction to ensure atomicity
		var iterationToolUses []*models.ToolUse
		err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
			for _, toolCall := range response.ToolCalls {
				// Create tool use
				toolUse, err := uc.toolService.CreateToolUse(txCtx, message.ID, toolCall.Name, toolCall.Arguments)
				if err != nil {
					return fmt.Errorf("failed to create tool use for %s: %w", toolCall.Name, err)
				}

				// Execute the tool
				toolUse, err = uc.toolService.ExecuteToolUse(txCtx, toolUse.ID)
				if err != nil {
					return fmt.Errorf("failed to execute tool %s: %w", toolCall.Name, err)
				}

				iterationToolUses = append(iterationToolUses, toolUse)
			}
			return nil
		})

		if err != nil {
			// On error, add error message to conversation and continue to next iteration
			currentMessages = append(currentMessages, ports.LLMMessage{
				Role:    "tool",
				Content: fmt.Sprintf("Error processing tools: %s", err.Error()),
			})
			continue
		}

		// Add successfully processed tool uses to output and message history
		for _, toolUse := range iterationToolUses {
			output.ToolUses = append(output.ToolUses, toolUse)

			// Add tool result to message history
			resultContent := fmt.Sprintf("%v", toolUse.Result)
			currentMessages = append(currentMessages, ports.LLMMessage{
				Role:    "tool",
				Content: resultContent,
			})
		}
	}

	// If we hit max iterations, update message with the last response
	err := uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		message.Contents = "Max tool execution iterations reached. Last response: " + strings.TrimSpace(currentMessages[len(currentMessages)-1].Content)
		return uc.messageRepo.Update(txCtx, message)
	})

	if err != nil {
		return nil, err
	}

	return output, nil
}

func (uc *GenerateResponse) executeStreaming(
	ctx context.Context,
	input *ports.GenerateResponseInput,
	message *models.Message,
	llmMessages []ports.LLMMessage,
	tools []*models.Tool,
) (*ports.GenerateResponseOutput, error) {
	var streamChan <-chan ports.LLMStreamChunk
	var err error

	if len(tools) > 0 {
		streamChan, err = uc.llmService.ChatStreamWithTools(ctx, llmMessages, tools)
	} else {
		streamChan, err = uc.llmService.ChatStream(ctx, llmMessages)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start streaming response: %w", err)
	}

	// Mark message as streaming before we start
	message.MarkAsStreaming()
	if err := uc.messageRepo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to mark message as streaming: %w", err)
	}

	outputChan := make(chan *ports.ResponseStreamChunk, 10)

	go uc.processStream(ctx, message, streamChan, outputChan, input.EnableReasoning)

	return &ports.GenerateResponseOutput{
		Message:        message,
		StreamChannel:  outputChan,
		Sentences:      []*models.Sentence{},
		ToolUses:       []*models.ToolUse{},
		ReasoningSteps: []*models.ReasoningStep{},
	}, nil
}

func (uc *GenerateResponse) processStream(
	ctx context.Context,
	message *models.Message,
	inputChan <-chan ports.LLMStreamChunk,
	outputChan chan<- *ports.ResponseStreamChunk,
	enableReasoning bool,
) {
	// Track created sentence IDs for cleanup on failure
	createdSentenceIDs := make([]string, 0, 50)
	streamingSucceeded := false

	// Ensure cleanup happens on any exit path (normal return, panic, or error)
	defer func() {
		close(outputChan)

		// Handle panic recovery
		if r := recover(); r != nil {
			log.Printf("PANIC in processStream: %v\n", r)

			// Mark message as failed
			message.MarkAsFailed()
			if err := uc.messageRepo.Update(ctx, message); err != nil {
				log.Printf("Failed to mark message as failed after panic: %v\n", err)
			}

			// Mark all created sentences as failed
			for _, sentenceID := range createdSentenceIDs {
				sentence, err := uc.sentenceRepo.GetByID(ctx, sentenceID)
				if err == nil && sentence != nil {
					sentence.MarkAsFailed()
					if err := uc.sentenceRepo.Update(ctx, sentence); err != nil {
						log.Printf("ERROR: failed to mark sentence %s as failed: %v", sentenceID, err)
					}
				}
			}

			// Send error to output channel if still open
			select {
			case outputChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("panic during streaming: %v", r)}:
			default:
			}
		} else if !streamingSucceeded {
			// Stream ended abnormally without panic (e.g., error or context cancellation)
			message.MarkAsFailed()
			if err := uc.messageRepo.Update(ctx, message); err != nil {
				log.Printf("Failed to mark message as failed: %v\n", err)
			}

			// Mark all created sentences as failed
			for _, sentenceID := range createdSentenceIDs {
				sentence, err := uc.sentenceRepo.GetByID(ctx, sentenceID)
				if err == nil && sentence != nil {
					sentence.MarkAsFailed()
					if err := uc.sentenceRepo.Update(ctx, sentence); err != nil {
						log.Printf("ERROR: failed to mark sentence %s as failed: %v", sentenceID, err)
					}
				}
			}
		}
	}()

	var fullContent strings.Builder
	var textBuffer strings.Builder
	sentenceSequence := 0
	reasoningSequence := 0

	const maxSentenceLength = 10000 // Maximum sentence length to prevent memory issues

	for chunk := range inputChan {
		if chunk.Error != nil {
			outputChan <- &ports.ResponseStreamChunk{Error: chunk.Error}
			return
		}

		if chunk.Content != "" {
			fullContent.WriteString(chunk.Content)
			textBuffer.WriteString(chunk.Content)

			// Extract complete sentences from the buffer
			bufferText := textBuffer.String()
			for {
				sentenceText, remaining := uc.extractNextSentence(bufferText, maxSentenceLength)
				if sentenceText == "" {
					// No complete sentence found, keep accumulating
					break
				}

				// Create and save the sentence with streaming status
				sentenceID := uc.idGenerator.GenerateSentenceID()
				sentence := models.NewSentence(sentenceID, message.ID, sentenceSequence, sentenceText)
				sentence.MarkAsStreaming()

				if err := uc.sentenceRepo.Create(ctx, sentence); err != nil {
					outputChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("failed to create sentence: %w", err)}
					return
				}

				// Track sentence for cleanup
				createdSentenceIDs = append(createdSentenceIDs, sentenceID)

				outputChan <- &ports.ResponseStreamChunk{
					SentenceID: sentenceID,
					Sequence:   sentenceSequence,
					Text:       sentenceText,
					IsFinal:    false,
				}

				sentenceSequence++
				bufferText = remaining
			}

			// Update buffer with remaining text
			textBuffer.Reset()
			textBuffer.WriteString(bufferText)
		}

		if chunk.ToolCall != nil {
			toolUseID := uc.idGenerator.GenerateToolUseID()
			toolUse := models.NewToolUse(
				toolUseID,
				message.ID,
				chunk.ToolCall.Name,
				sentenceSequence,
				chunk.ToolCall.Arguments,
			)

			if err := uc.toolUseRepo.Create(ctx, toolUse); err != nil {
				outputChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("failed to create tool use: %w", err)}
				return
			}

			// Send ToolUseRequest notification before execution
			outputChan <- &ports.ResponseStreamChunk{
				ToolCall:              chunk.ToolCall,
				ToolUseID:             toolUseID,
				IsToolExecutionResult: false,
			}

			// Execute the tool
			if uc.toolService != nil {
				executedToolUse, err := uc.toolService.ExecuteToolUse(ctx, toolUseID)
				if err != nil {
					log.Printf("Tool execution failed for %s: %v", chunk.ToolCall.Name, err)
					// Tool execution failure is already recorded in the ToolUse status
					// Send the failed ToolUse result to client
					outputChan <- &ports.ResponseStreamChunk{
						ToolCall:              chunk.ToolCall,
						ToolUseID:             toolUseID,
						IsToolExecutionResult: true,
					}
				} else {
					// Send ToolUseResult notification after successful execution
					outputChan <- &ports.ResponseStreamChunk{
						ToolCall:              chunk.ToolCall,
						ToolUseID:             executedToolUse.ID,
						IsToolExecutionResult: true,
					}
				}
			}
		}

		if enableReasoning && chunk.Reasoning != "" {
			stepID := uc.idGenerator.GenerateReasoningStepID()
			step := &models.ReasoningStep{
				ID:             stepID,
				MessageID:      message.ID,
				SequenceNumber: reasoningSequence,
				Content:        chunk.Reasoning,
			}

			if err := uc.reasoningStepRepo.Create(ctx, step); err != nil {
				outputChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("failed to create reasoning step: %w", err)}
				return
			}

			outputChan <- &ports.ResponseStreamChunk{
				Reasoning: chunk.Reasoning,
			}
			reasoningSequence++
		}

		if chunk.Done {
			// Process any remaining text in the buffer as the final sentence
			if textBuffer.Len() > 0 {
				remainingText := strings.TrimSpace(textBuffer.String())
				if remainingText != "" {
					sentenceID := uc.idGenerator.GenerateSentenceID()
					sentence := models.NewSentence(sentenceID, message.ID, sentenceSequence, remainingText)
					sentence.MarkAsCompleted() // Final sentence is completed immediately

					if err := uc.sentenceRepo.Create(ctx, sentence); err != nil {
						outputChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("failed to create final sentence: %w", err)}
						return
					}

					// Track sentence for cleanup
					createdSentenceIDs = append(createdSentenceIDs, sentenceID)

					outputChan <- &ports.ResponseStreamChunk{
						SentenceID: sentenceID,
						Sequence:   sentenceSequence,
						Text:       remainingText,
						IsFinal:    true,
					}
				}
			}

			// Mark all sentences as completed
			for _, sentenceID := range createdSentenceIDs {
				sentence, err := uc.sentenceRepo.GetByID(ctx, sentenceID)
				if err == nil && sentence != nil && !sentence.IsCompleted() {
					sentence.MarkAsCompleted()
					if err := uc.sentenceRepo.Update(ctx, sentence); err != nil {
						log.Printf("ERROR: failed to mark sentence %s as completed: %v", sentenceID, err)
					}
				}
			}

			// Update message with final content and mark as completed
			message.Contents = strings.TrimSpace(fullContent.String())
			message.MarkAsCompleted()
			if err := uc.messageRepo.Update(ctx, message); err != nil {
				outputChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("failed to update message: %w", err)}
				return
			}

			streamingSucceeded = true

			// Extract and store important memories from the response (async, non-blocking)
			// Use detached context with timeout to avoid being cancelled when parent completes
			go func() {
				memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
				defer cancel()
				uc.extractAndStoreMemories(memCtx, message, message.ConversationID)
			}()

			// Generate conversation title if needed (async, non-blocking)
			uc.titleGenerator.ExecuteAsync(ctx, message.ConversationID)

			return
		}
	}
}

// extractNextSentence extracts the next complete sentence from the text buffer
// Returns the sentence text and the remaining buffer text
func (uc *GenerateResponse) extractNextSentence(text string, maxLength int) (sentence string, remaining string) {
	if text == "" {
		return "", ""
	}

	// Find the first occurrence of sentence-ending punctuation
	var endIdx int = -1
	var endChar rune

	for i, char := range text {
		if char == '.' || char == '!' || char == '?' {
			endIdx = i
			endChar = char
			break
		}
	}

	// No sentence boundary found
	if endIdx == -1 {
		// If buffer exceeds max length, force a sentence break at the last space
		if len(text) > maxLength {
			lastSpace := strings.LastIndex(text[:maxLength], " ")
			if lastSpace > 0 {
				sentence = strings.TrimSpace(text[:lastSpace])
				remaining = strings.TrimSpace(text[lastSpace:])
				return sentence, remaining
			}
			// No space found, just break at max length
			sentence = strings.TrimSpace(text[:maxLength])
			remaining = strings.TrimSpace(text[maxLength:])
			return sentence, remaining
		}
		return "", text
	}

	// Check for common abbreviations to avoid false sentence breaks
	// Common patterns: "Dr.", "Mr.", "Mrs.", "Ms.", "etc.", "e.g.", "i.e."
	if endChar == '.' && endIdx > 0 && endIdx < len(text)-1 {
		// Check if it's likely an abbreviation
		beforeDot := ""
		if endIdx >= 5 {
			beforeDot = strings.ToLower(text[endIdx-5 : endIdx])
		} else {
			beforeDot = strings.ToLower(text[:endIdx])
		}

		isAbbreviation := strings.HasSuffix(beforeDot, "dr") ||
			strings.HasSuffix(beforeDot, "mr") ||
			strings.HasSuffix(beforeDot, "mrs") ||
			strings.HasSuffix(beforeDot, "ms") ||
			strings.HasSuffix(beforeDot, "etc") ||
			strings.HasSuffix(beforeDot, "e.g") ||
			strings.HasSuffix(beforeDot, "i.e") ||
			strings.HasSuffix(beforeDot, "inc") ||
			strings.HasSuffix(beforeDot, "ltd") ||
			strings.HasSuffix(beforeDot, "co")

		// If it's an abbreviation and followed by lowercase or another abbreviation, skip this period
		if isAbbreviation && endIdx < len(text)-2 {
			nextChar := text[endIdx+1]
			if nextChar == ' ' && endIdx < len(text)-3 {
				charAfterSpace := text[endIdx+2]
				if charAfterSpace >= 'a' && charAfterSpace <= 'z' {
					// This is likely an abbreviation in the middle of a sentence
					// Continue searching from the next position
					nextSentence, nextRemaining := uc.extractNextSentence(text[endIdx+1:], maxLength)
					if nextSentence != "" {
						return text[:endIdx+1] + nextSentence, nextRemaining
					}
					return "", text
				}
			}
		}
	}

	// Extract the sentence including the punctuation
	sentence = strings.TrimSpace(text[:endIdx+1])

	// The remaining text starts after the punctuation and any whitespace
	if endIdx+1 < len(text) {
		remaining = strings.TrimSpace(text[endIdx+1:])
	} else {
		remaining = ""
	}

	return sentence, remaining
}

// retrieveRelevantMemories retrieves memories relevant to the user message using MemoryService
// Returns both the memories and their similarity scores for tracking
func (uc *GenerateResponse) retrieveRelevantMemories(ctx context.Context, conversationID, userMessageID string) ([]*ports.MemorySearchResult, error) {
	// Get the user message to use as query
	userMessage, err := uc.messageRepo.GetByID(ctx, userMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user message: %w", err)
	}

	// Use MemoryService to search for relevant memories with scores
	// This allows us to track the actual similarity scores for analytics
	searchResults, err := uc.memoryService.SearchWithScores(ctx, userMessage.Contents, 0.7, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	return searchResults, nil
}

// mergeMemories merges two memory lists, removing duplicates
func (uc *GenerateResponse) mergeMemories(existing, new []*models.Memory) []*models.Memory {
	if len(existing) == 0 {
		return new
	}
	if len(new) == 0 {
		return existing
	}

	// Use a map to track unique memory IDs
	seen := make(map[string]bool)
	merged := make([]*models.Memory, 0, len(existing)+len(new))

	// Add existing memories
	for _, mem := range existing {
		if !seen[mem.ID] {
			seen[mem.ID] = true
			merged = append(merged, mem)
		}
	}

	// Add new memories that haven't been seen
	for _, mem := range new {
		if !seen[mem.ID] {
			seen[mem.ID] = true
			merged = append(merged, mem)
		}
	}

	return merged
}

// trackMemoryUsageWithScores records which memories were used in the response generation
// along with their similarity scores for analytics
func (uc *GenerateResponse) trackMemoryUsageWithScores(ctx context.Context, conversationID, messageID string, searchResults []*ports.MemorySearchResult) {
	// Note: This is called to track memory usage for analytics
	// We intentionally swallow errors here as tracking is for analytics only
	for _, result := range searchResults {
		// Use MemoryService to track usage with the actual similarity score
		_, err := uc.memoryService.TrackUsage(ctx, result.Memory.ID, conversationID, messageID, result.Similarity)
		if err != nil {
			// Log but don't fail - memory usage tracking is optional
			log.Printf("warning: failed to track memory usage for memory %s: %v\n", result.Memory.ID, err)
		}
	}
}

// extractAndStoreMemories extracts important information from the assistant's response
// and stores it as memories for future retrieval. This runs asynchronously to avoid
// blocking the response flow.
func (uc *GenerateResponse) extractAndStoreMemories(ctx context.Context, message *models.Message, conversationID string) {
	// Skip if memory service is not available
	if uc.memoryService == nil {
		return
	}

	// Skip if message content is too short (less than 50 characters)
	if len(message.Contents) < 50 {
		return
	}

	// Use LLM to extract important information from the response
	extractionPrompt := []ports.LLMMessage{
		{
			Role: "system",
			Content: `You are a memory extraction assistant. Your task is to identify important, factual information from the assistant's response that should be remembered for future conversations.

Extract information that is:
- Factual and specific (names, dates, preferences, facts)
- Likely to be useful in future conversations
- Not trivial or conversational filler

For each piece of important information, output it as a separate line starting with "MEMORY:".
If there's no important information to remember, output "NONE".

Examples:
Input: "I love hiking! My favorite trail is the Pacific Crest Trail. I usually go on weekends."
Output:
MEMORY: User's favorite trail is the Pacific Crest Trail
MEMORY: User usually goes hiking on weekends

Input: "Okay, let me help you with that."
Output: NONE`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Extract important memories from this assistant response:\n\n%s", message.Contents),
		},
	}

	extraction, err := uc.llmService.Chat(ctx, extractionPrompt)
	if err != nil {
		log.Printf("warning: failed to extract memories from response: %v\n", err)
		return
	}

	// Parse the extraction result
	if extraction.Content == "NONE" || extraction.Content == "" {
		return
	}

	// Split by lines and process each memory
	lines := strings.Split(extraction.Content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "MEMORY:") {
			continue
		}

		// Extract the memory content
		memoryContent := strings.TrimSpace(strings.TrimPrefix(line, "MEMORY:"))
		if memoryContent == "" || len(memoryContent) < 10 {
			continue
		}

		// Create memory from conversation with embeddings
		memory, err := uc.memoryService.CreateFromConversation(ctx, memoryContent, conversationID, message.ID)
		if err != nil {
			log.Printf("warning: failed to create memory '%s': %v\n", memoryContent, err)
			continue
		}

		// Set a default importance score based on content length and specificity
		importance := calculateMemoryImportance(memoryContent)
		_, err = uc.memoryService.SetImportance(ctx, memory.ID, importance)
		if err != nil {
			log.Printf("warning: failed to set memory importance: %v\n", err)
		}

		log.Printf("info: created memory from conversation: %s\n", memoryContent)
	}
}

// calculateMemoryImportance calculates an importance score based on content characteristics
func calculateMemoryImportance(content string) float32 {
	// Base importance
	importance := float32(0.5)

	// Longer memories tend to be more detailed and important (up to +0.2)
	if len(content) > 100 {
		importance += 0.2
	} else if len(content) > 50 {
		importance += 0.1
	}

	// Memories with specific markers are often more important
	specificMarkers := []string{"preference", "favorite", "always", "never", "important", "remember"}
	for _, marker := range specificMarkers {
		if strings.Contains(strings.ToLower(content), marker) {
			importance += 0.1
			break
		}
	}

	// Cap at 1.0
	if importance > 1.0 {
		importance = 1.0
	}

	return importance
}
