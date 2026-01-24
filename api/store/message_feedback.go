package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateMessageFeedback inserts a new message feedback.
func (s *Store) CreateMessageFeedback(ctx context.Context, fb *domain.MessageFeedback) error {
	query := `
		INSERT INTO message_feedback (id, message_id, rating, note, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.conn(ctx).Exec(ctx, query,
		fb.ID, fb.MessageID, fb.Rating, fb.Note, fb.CreatedAt)
	if err != nil {
		return fmt.Errorf("create message feedback: %w", err)
	}
	return nil
}

// GetMessageFeedback retrieves a message feedback by ID.
func (s *Store) GetMessageFeedback(ctx context.Context, id string) (*domain.MessageFeedback, error) {
	query := `
		SELECT id, message_id, rating, note, created_at
		FROM message_feedback
		WHERE id = $1`

	fb := &domain.MessageFeedback{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&fb.ID, &fb.MessageID, &fb.Rating, &fb.Note, &fb.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get message feedback: %w", err)
	}
	return fb, nil
}

// UpdateMessageFeedback updates a message feedback's rating and note.
func (s *Store) UpdateMessageFeedback(ctx context.Context, fb *domain.MessageFeedback) error {
	query := `
		UPDATE message_feedback
		SET rating = $2, note = $3
		WHERE id = $1`

	_, err := s.conn(ctx).Exec(ctx, query, fb.ID, fb.Rating, fb.Note)
	if err != nil {
		return fmt.Errorf("update message feedback: %w", err)
	}
	return nil
}

// DeleteMessageFeedback deletes a message feedback.
func (s *Store) DeleteMessageFeedback(ctx context.Context, id string) error {
	query := `DELETE FROM message_feedback WHERE id = $1`
	result, err := s.conn(ctx).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete message feedback: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetMessageFeedbackByMessage returns all feedback for a message.
func (s *Store) GetMessageFeedbackByMessage(ctx context.Context, messageID string) ([]*domain.MessageFeedback, error) {
	query := `
		SELECT id, message_id, rating, note, created_at
		FROM message_feedback
		WHERE message_id = $1
		ORDER BY created_at DESC`

	rows, err := s.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("get message feedback: %w", err)
	}
	defer rows.Close()

	var fbs []*domain.MessageFeedback
	for rows.Next() {
		fb := &domain.MessageFeedback{}
		if err := rows.Scan(&fb.ID, &fb.MessageID, &fb.Rating, &fb.Note, &fb.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message feedback: %w", err)
		}
		fbs = append(fbs, fb)
	}
	return fbs, rows.Err()
}

// ListMessageFeedback returns all message feedback with pagination.
func (s *Store) ListMessageFeedback(ctx context.Context, limit, offset int) ([]*domain.MessageFeedback, int, error) {
	countQuery := `SELECT COUNT(*) FROM message_feedback`
	var total int
	if err := s.conn(ctx).QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count message feedback: %w", err)
	}

	query := `
		SELECT id, message_id, rating, note, created_at
		FROM message_feedback
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list message feedback: %w", err)
	}
	defer rows.Close()

	var fbs []*domain.MessageFeedback
	for rows.Next() {
		fb := &domain.MessageFeedback{}
		if err := rows.Scan(&fb.ID, &fb.MessageID, &fb.Rating, &fb.Note, &fb.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan message feedback: %w", err)
		}
		fbs = append(fbs, fb)
	}
	return fbs, total, rows.Err()
}
