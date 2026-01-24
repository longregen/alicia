package services

import (
	"context"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
)

// MessageService handles message operations.
type MessageService struct {
	store *store.Store
}

// NewMessageService creates a new message service.
func NewMessageService(s *store.Store) *MessageService {
	return &MessageService{store: s}
}

// CreateUserMessage creates a user message and broadcasts a generation request.
func (svc *MessageService) CreateUserMessage(ctx context.Context, convID, content string, previousID *string) (*domain.Message, error) {
	msg := &domain.Message{
		ID:             store.NewMessageID(),
		ConversationID: convID,
		PreviousID:     previousID,
		Role:           domain.RoleUser,
		Content:        content,
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}

	err := svc.store.WithTx(ctx, func(ctx context.Context) error {
		if err := svc.store.CreateMessage(ctx, msg); err != nil {
			return err
		}
		return svc.store.UpdateConversationTip(ctx, convID, msg.ID)
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// CreateAssistantMessage creates an assistant message (called by agent).
func (svc *MessageService) CreateAssistantMessage(ctx context.Context, convID, content, reasoning string, previousID *string) (*domain.Message, error) {
	msg := &domain.Message{
		ID:             store.NewMessageID(),
		ConversationID: convID,
		PreviousID:     previousID,
		Role:           domain.RoleAssistant,
		Content:        content,
		Reasoning:      reasoning,
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}

	err := svc.store.WithTx(ctx, func(ctx context.Context) error {
		if err := svc.store.CreateMessage(ctx, msg); err != nil {
			return err
		}
		return svc.store.UpdateConversationTip(ctx, convID, msg.ID)
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// GetMessage retrieves a message by ID.
func (svc *MessageService) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	return svc.store.GetMessage(ctx, id)
}

// GetMessageChain retrieves the message chain from root to tip.
func (svc *MessageService) GetMessageChain(ctx context.Context, tipID string) ([]*domain.Message, error) {
	return svc.store.GetMessageChain(ctx, tipID)
}

// GetMessageSiblings retrieves sibling messages (same parent).
func (svc *MessageService) GetMessageSiblings(ctx context.Context, messageID string) ([]*domain.Message, error) {
	return svc.store.GetMessageSiblings(ctx, messageID)
}

// ListMessages retrieves messages for a conversation.
func (svc *MessageService) ListMessages(ctx context.Context, convID string, limit int) ([]*domain.Message, error) {
	return svc.store.ListMessages(ctx, convID, limit)
}
