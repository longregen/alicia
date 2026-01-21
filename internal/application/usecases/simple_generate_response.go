package usecases

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
)

// SimpleGenerateResponse is a minimal implementation of GenerateResponseUseCase
// that directly calls LLM without Pareto optimization, tools, or memory.
// Useful for CLI chat and testing.
type SimpleGenerateResponse struct {
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	llmClient        *llm.Client
	idGenerator      ports.IDGenerator
}

// NewSimpleGenerateResponse creates a new SimpleGenerateResponse use case
func NewSimpleGenerateResponse(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	llmClient *llm.Client,
	idGenerator ports.IDGenerator,
) *SimpleGenerateResponse {
	return &SimpleGenerateResponse{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		llmClient:        llmClient,
		idGenerator:      idGenerator,
	}
}

// Execute generates a response by calling the LLM and streaming the result
func (uc *SimpleGenerateResponse) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Get conversation history
	messages, err := uc.messageRepo.GetByConversation(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Convert to LLM format
	llmMessages := make([]llm.ChatMessage, 0, len(messages))
	for _, msg := range messages {
		llmMessages = append(llmMessages, llm.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Contents,
		})
	}

	// Get next sequence number
	seqNum, err := uc.messageRepo.GetNextSequenceNumber(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequence number: %w", err)
	}

	// Generate message ID
	messageID := input.MessageID
	if messageID == "" {
		messageID = uc.idGenerator.GenerateMessageID()
	}

	// Create stream channel for output
	streamChan := make(chan *ports.ResponseStreamChunk, 100)

	// Start streaming in goroutine
	go func() {
		defer close(streamChan)

		// Call LLM with streaming
		responseChan, err := uc.llmClient.ChatStream(ctx, llmMessages)
		if err != nil {
			streamChan <- &ports.ResponseStreamChunk{Error: err}
			return
		}

		var fullContent string
		sequence := 0

		for chunk := range responseChan {
			if chunk.Error != nil {
				streamChan <- &ports.ResponseStreamChunk{Error: chunk.Error}
				return
			}

			if chunk.Content != "" {
				fullContent += chunk.Content
				streamChan <- &ports.ResponseStreamChunk{
					Sequence: sequence,
					Text:     chunk.Content,
					IsFinal:  false,
				}
				sequence++
			}
		}

		// Create and save the assistant message
		assistantMessage := models.NewAssistantMessage(messageID, input.ConversationID, seqNum, fullContent)
		if input.PreviousID != "" {
			assistantMessage.SetPreviousMessage(input.PreviousID)
		}

		if err := uc.messageRepo.Create(ctx, assistantMessage); err != nil {
			streamChan <- &ports.ResponseStreamChunk{Error: fmt.Errorf("failed to save message: %w", err)}
			return
		}

		// Update conversation tip
		_ = uc.conversationRepo.UpdateTip(ctx, input.ConversationID, messageID)

		// Send final chunk
		streamChan <- &ports.ResponseStreamChunk{
			Sequence: sequence,
			Text:     "",
			IsFinal:  true,
		}
	}()

	return &ports.GenerateResponseOutput{
		StreamChannel: streamChan,
	}, nil
}

// Ensure SimpleGenerateResponse implements ports.GenerateResponseUseCase
var _ ports.GenerateResponseUseCase = (*SimpleGenerateResponse)(nil)
