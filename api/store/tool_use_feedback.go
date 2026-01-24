package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateToolUseFeedback inserts a new tool use feedback.
func (s *Store) CreateToolUseFeedback(ctx context.Context, fb *domain.ToolUseFeedback) error {
	query := `
		INSERT INTO tool_use_feedback (id, tool_use_id, rating, note, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.conn(ctx).Exec(ctx, query,
		fb.ID, fb.ToolUseID, fb.Rating, fb.Note, fb.CreatedAt)
	if err != nil {
		return fmt.Errorf("create tool use feedback: %w", err)
	}
	return nil
}

// GetToolUseFeedback retrieves a tool use feedback by ID.
func (s *Store) GetToolUseFeedback(ctx context.Context, id string) (*domain.ToolUseFeedback, error) {
	query := `
		SELECT id, tool_use_id, rating, note, created_at
		FROM tool_use_feedback
		WHERE id = $1`

	fb := &domain.ToolUseFeedback{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&fb.ID, &fb.ToolUseID, &fb.Rating, &fb.Note, &fb.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get tool use feedback: %w", err)
	}
	return fb, nil
}

// UpdateToolUseFeedback updates a tool use feedback's rating and note.
func (s *Store) UpdateToolUseFeedback(ctx context.Context, fb *domain.ToolUseFeedback) error {
	query := `
		UPDATE tool_use_feedback
		SET rating = $2, note = $3
		WHERE id = $1`

	_, err := s.conn(ctx).Exec(ctx, query, fb.ID, fb.Rating, fb.Note)
	if err != nil {
		return fmt.Errorf("update tool use feedback: %w", err)
	}
	return nil
}

// DeleteToolUseFeedback deletes a tool use feedback.
func (s *Store) DeleteToolUseFeedback(ctx context.Context, id string) error {
	query := `DELETE FROM tool_use_feedback WHERE id = $1`
	result, err := s.conn(ctx).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tool use feedback: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetToolUseFeedbackByToolUse returns all feedback for a tool use.
func (s *Store) GetToolUseFeedbackByToolUse(ctx context.Context, toolUseID string) ([]*domain.ToolUseFeedback, error) {
	query := `
		SELECT id, tool_use_id, rating, note, created_at
		FROM tool_use_feedback
		WHERE tool_use_id = $1
		ORDER BY created_at DESC`

	rows, err := s.conn(ctx).Query(ctx, query, toolUseID)
	if err != nil {
		return nil, fmt.Errorf("get tool use feedback: %w", err)
	}
	defer rows.Close()

	var fbs []*domain.ToolUseFeedback
	for rows.Next() {
		fb := &domain.ToolUseFeedback{}
		if err := rows.Scan(&fb.ID, &fb.ToolUseID, &fb.Rating, &fb.Note, &fb.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tool use feedback: %w", err)
		}
		fbs = append(fbs, fb)
	}
	return fbs, rows.Err()
}

// ListToolUseFeedback returns all tool use feedback with pagination.
func (s *Store) ListToolUseFeedback(ctx context.Context, limit, offset int) ([]*domain.ToolUseFeedback, int, error) {
	countQuery := `SELECT COUNT(*) FROM tool_use_feedback`
	var total int
	if err := s.conn(ctx).QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tool use feedback: %w", err)
	}

	query := `
		SELECT id, tool_use_id, rating, note, created_at
		FROM tool_use_feedback
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tool use feedback: %w", err)
	}
	defer rows.Close()

	var fbs []*domain.ToolUseFeedback
	for rows.Next() {
		fb := &domain.ToolUseFeedback{}
		if err := rows.Scan(&fb.ID, &fb.ToolUseID, &fb.Rating, &fb.Note, &fb.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan tool use feedback: %w", err)
		}
		fbs = append(fbs, fb)
	}
	return fbs, total, rows.Err()
}
