package services

import (
	"context"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
)

type ConversationService struct {
	store *store.Store
}

func NewConversationService(s *store.Store) *ConversationService {
	return &ConversationService{store: s}
}

func (svc *ConversationService) Create(ctx context.Context, userID, title string) (*domain.Conversation, error) {
	conv := &domain.Conversation{
		ID:        store.NewConversationID(),
		UserID:    userID,
		Title:     title,
		Status:    domain.ConversationStatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := svc.store.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (svc *ConversationService) Get(ctx context.Context, id string) (*domain.Conversation, error) {
	return svc.store.GetConversation(ctx, id)
}

func (svc *ConversationService) GetByUser(ctx context.Context, id, userID string) (*domain.Conversation, error) {
	return svc.store.GetConversationByUser(ctx, id, userID)
}

func (svc *ConversationService) List(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, int, error) {
	return svc.store.ListConversations(ctx, userID, limit, offset)
}

func (svc *ConversationService) ListActive(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, int, error) {
	return svc.store.ListActiveConversations(ctx, userID, limit, offset)
}

func (svc *ConversationService) Update(ctx context.Context, conv *domain.Conversation) error {
	return svc.store.UpdateConversation(ctx, conv)
}

func (svc *ConversationService) Delete(ctx context.Context, id string) error {
	return svc.store.DeleteConversation(ctx, id)
}

func (svc *ConversationService) UpdateTip(ctx context.Context, convID, messageID string) error {
	return svc.store.UpdateConversationTip(ctx, convID, messageID)
}
