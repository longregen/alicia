package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateConversation inserts a new conversation.
func (s *Store) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	query := `
		INSERT INTO conversations (id, user_id, title, status, tip_message_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.conn(ctx).Exec(ctx, query,
		conv.ID, conv.UserID, conv.Title, conv.Status,
		conv.TipMessageID, conv.CreatedAt, conv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	return nil
}

// GetConversation retrieves a conversation by ID.
func (s *Store) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	query := `
		SELECT id, user_id, title, status, tip_message_id, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND deleted_at IS NULL`

	conv := &domain.Conversation{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.UserID, &conv.Title, &conv.Status,
		&conv.TipMessageID, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return conv, nil
}

// GetConversationByUser retrieves a conversation by ID and user ID.
func (s *Store) GetConversationByUser(ctx context.Context, id, userID string) (*domain.Conversation, error) {
	query := `
		SELECT id, user_id, title, status, tip_message_id, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	conv := &domain.Conversation{}
	err := s.conn(ctx).QueryRow(ctx, query, id, userID).Scan(
		&conv.ID, &conv.UserID, &conv.Title, &conv.Status,
		&conv.TipMessageID, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get conversation by user: %w", err)
	}
	return conv, nil
}

// UpdateConversation updates a conversation's mutable fields.
func (s *Store) UpdateConversation(ctx context.Context, conv *domain.Conversation) error {
	query := `
		UPDATE conversations
		SET title = $2, status = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL`

	conv.UpdatedAt = time.Now().UTC()
	_, err := s.conn(ctx).Exec(ctx, query,
		conv.ID, conv.Title, conv.Status, conv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update conversation: %w", err)
	}
	return nil
}

// UpdateConversationTip updates the tip message ID.
func (s *Store) UpdateConversationTip(ctx context.Context, convID, messageID string) error {
	query := `
		UPDATE conversations
		SET tip_message_id = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := s.conn(ctx).Exec(ctx, query, convID, messageID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update conversation tip: %w", err)
	}
	return nil
}

// DeleteConversation soft-deletes a conversation.
func (s *Store) DeleteConversation(ctx context.Context, id string) error {
	query := `UPDATE conversations SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	result, err := s.conn(ctx).Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListConversations returns conversations for a user with total count.
func (s *Store) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, int, error) {
	return s.listConversations(ctx, userID, false, limit, offset)
}

// ListActiveConversations returns active conversations for a user with total count.
func (s *Store) ListActiveConversations(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, int, error) {
	return s.listConversations(ctx, userID, true, limit, offset)
}

func (s *Store) listConversations(ctx context.Context, userID string, activeOnly bool, limit, offset int) ([]*domain.Conversation, int, error) {
	statusFilter := ""
	if activeOnly {
		statusFilter = " AND status = 'active'"
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM conversations WHERE user_id = $1` + statusFilter + ` AND deleted_at IS NULL`
	var total int
	if err := s.conn(ctx).QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count conversations: %w", err)
	}

	query := `
		SELECT id, user_id, title, status, tip_message_id, created_at, updated_at
		FROM conversations
		WHERE user_id = $1` + statusFilter + ` AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.conn(ctx).Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var convs []*domain.Conversation
	for rows.Next() {
		conv := &domain.Conversation{}
		if err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.Title, &conv.Status,
			&conv.TipMessageID, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan conversation: %w", err)
		}
		convs = append(convs, conv)
	}
	return convs, total, rows.Err()
}
