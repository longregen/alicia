package usecases

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// ParetoRegenerateResponse regenerates an assistant response using the Pareto search approach.
// This wraps the ParetoResponseGenerator for the regenerate use case.
type ParetoRegenerateResponse struct {
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	paretoGenerator  *ParetoResponseGenerator
	idGenerator      ports.IDGenerator
}

// NewParetoRegenerateResponse creates a new ParetoRegenerateResponse use case.
func NewParetoRegenerateResponse(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	paretoGenerator *ParetoResponseGenerator,
	idGenerator ports.IDGenerator,
) *ParetoRegenerateResponse {
	return &ParetoRegenerateResponse{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		paretoGenerator:  paretoGenerator,
		idGenerator:      idGenerator,
	}
}

// Execute regenerates an assistant message using the Pareto search approach.
// It finds the user message that preceded the assistant message and generates a new response.
func (r *ParetoRegenerateResponse) Execute(ctx context.Context, input *ports.RegenerateResponseInput) (*ports.RegenerateResponseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.MessageID == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	// Get the assistant message to regenerate
	assistantMessage, err := r.messageRepo.GetByID(ctx, input.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant message: %w", err)
	}

	if assistantMessage.Role != models.MessageRoleAssistant {
		return nil, fmt.Errorf("can only regenerate assistant messages")
	}

	// Get the conversation to find history
	conversation, err := r.conversationRepo.GetByID(ctx, assistantMessage.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Find the user message that this assistant message was responding to
	var userMessage *models.Message
	if assistantMessage.PreviousID != "" {
		userMessage, err = r.messageRepo.GetByID(ctx, assistantMessage.PreviousID)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous user message: %w", err)
		}
	} else {
		// Fall back to finding by sequence number
		messages, err := r.messageRepo.GetByConversation(ctx, assistantMessage.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversation messages: %w", err)
		}

		for i, msg := range messages {
			if msg.ID == assistantMessage.ID && i > 0 {
				userMessage = messages[i-1]
				break
			}
		}
	}

	if userMessage == nil {
		return nil, fmt.Errorf("could not find user message to regenerate from")
	}

	// Soft-delete the old assistant message by marking it as deleted
	assistantMessage.SoftDelete()
	if err := r.messageRepo.Update(ctx, assistantMessage); err != nil {
		return nil, fmt.Errorf("failed to delete old assistant message: %w", err)
	}

	// Generate a new response using Pareto search
	newMessageID := r.idGenerator.GenerateMessageID()
	paretoInput := &ParetoResponseInput{
		ConversationID:  conversation.ID,
		UserMessageID:   userMessage.ID,
		MessageID:       newMessageID,
		PreviousID:      userMessage.ID, // Link to user message
		EnableTools:     input.EnableTools,
		EnableReasoning: input.EnableReasoning,
		EnableStreaming: input.EnableStreaming,
		Notifier:        input.Notifier,
	}

	paretoOutput, err := r.paretoGenerator.Execute(ctx, paretoInput)
	if err != nil {
		return nil, fmt.Errorf("pareto response generation failed: %w", err)
	}

	return &ports.RegenerateResponseOutput{
		DeletedMessageID: assistantMessage.ID,
		NewMessage:       paretoOutput.Message,
		StreamChannel:    paretoOutput.StreamChannel,
	}, nil
}

// Ensure ParetoRegenerateResponse implements ports.RegenerateResponseUseCase
var _ ports.RegenerateResponseUseCase = (*ParetoRegenerateResponse)(nil)
