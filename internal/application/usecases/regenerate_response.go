package usecases

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// RegenerateResponse handles the regeneration of an assistant message.
// It deletes the existing assistant message and generates a new one
// from the same user message.
type RegenerateResponse struct {
	messageRepo        ports.MessageRepository
	conversationRepo   ports.ConversationRepository
	generateResponseUC *GenerateResponse
	idGenerator        ports.IDGenerator
}

// NewRegenerateResponse creates a new RegenerateResponse use case
func NewRegenerateResponse(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	generateResponseUC *GenerateResponse,
	idGenerator ports.IDGenerator,
) *RegenerateResponse {
	return &RegenerateResponse{
		messageRepo:        messageRepo,
		conversationRepo:   conversationRepo,
		generateResponseUC: generateResponseUC,
		idGenerator:        idGenerator,
	}
}

// Execute regenerates a response for the given assistant message.
// It deletes the target assistant message and generates a new one
// using the same user message that prompted the original response.
func (uc *RegenerateResponse) Execute(ctx context.Context, input *ports.RegenerateResponseInput) (*ports.RegenerateResponseOutput, error) {
	// 1. Get target message by ID
	targetMessage, err := uc.messageRepo.GetByID(ctx, input.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target message: %w", err)
	}

	if targetMessage == nil {
		return nil, fmt.Errorf("message not found: %s", input.MessageID)
	}

	// 2. Validate it's an assistant message
	if targetMessage.Role != models.MessageRoleAssistant {
		return nil, fmt.Errorf("cannot regenerate: target message is not an assistant message (role: %s)", targetMessage.Role)
	}

	// 3. Get the user message via target.PreviousID
	if targetMessage.PreviousID == "" {
		return nil, fmt.Errorf("cannot regenerate: target message has no previous message reference")
	}

	userMessage, err := uc.messageRepo.GetByID(ctx, targetMessage.PreviousID)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous user message: %w", err)
	}

	if userMessage == nil {
		return nil, fmt.Errorf("previous user message not found: %s", targetMessage.PreviousID)
	}

	// Validate the previous message is a user message
	if userMessage.Role != models.MessageRoleUser {
		return nil, fmt.Errorf("cannot regenerate: previous message is not a user message (role: %s)", userMessage.Role)
	}

	// Store the message ID before deletion for output
	deletedMessageID := targetMessage.ID
	conversationID := targetMessage.ConversationID

	// 4. Delete the target assistant message
	// Note: Related entities (sentences, tool uses, reasoning steps) will be
	// cascade deleted by the database
	if err := uc.messageRepo.Delete(ctx, targetMessage.ID); err != nil {
		return nil, fmt.Errorf("failed to delete target message: %w", err)
	}

	// 5. Generate new message ID
	newMessageID := uc.idGenerator.GenerateMessageID()

	// 6. Call GenerateResponse.Execute() with the user message context
	generateInput := &ports.GenerateResponseInput{
		ConversationID:  conversationID,
		UserMessageID:   userMessage.ID,
		MessageID:       newMessageID,
		EnableTools:     input.EnableTools,
		EnableReasoning: input.EnableReasoning,
		EnableStreaming: input.EnableStreaming,
		PreviousID:      userMessage.ID,
		Notifier:        input.Notifier,
	}

	generateOutput, err := uc.generateResponseUC.Execute(ctx, generateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new response: %w", err)
	}

	// 7. Return DeletedMessageID, NewMessage, StreamChannel
	return &ports.RegenerateResponseOutput{
		DeletedMessageID: deletedMessageID,
		NewMessage:       generateOutput.Message,
		StreamChannel:    generateOutput.StreamChannel,
	}, nil
}
