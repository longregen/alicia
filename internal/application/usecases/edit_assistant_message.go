package usecases

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// EditAssistantMessage handles editing an assistant message's content in place
type EditAssistantMessage struct {
	messageRepo ports.MessageRepository
}

// NewEditAssistantMessage creates a new EditAssistantMessage use case
func NewEditAssistantMessage(messageRepo ports.MessageRepository) *EditAssistantMessage {
	return &EditAssistantMessage{
		messageRepo: messageRepo,
	}
}

// Execute edits an assistant message's content
func (uc *EditAssistantMessage) Execute(ctx context.Context, input *ports.EditAssistantMessageInput) (*ports.EditAssistantMessageOutput, error) {
	// 1. Get target message
	message, err := uc.messageRepo.GetByID(ctx, input.TargetMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// 2. Validate it's an assistant message
	if message.Role != models.MessageRoleAssistant {
		return nil, fmt.Errorf("cannot edit message: expected assistant role, got %s", message.Role)
	}

	// 3. Update content and mark as user-edited (valuable training data)
	message.Contents = input.NewContent
	message.MarkAsUserEdited()

	// 4. Save to repository
	if err := uc.messageRepo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	// 5. Return updated message
	return &ports.EditAssistantMessageOutput{
		UpdatedMessage: message,
	}, nil
}
