package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateMemoryUseFeedback inserts a new memory use feedback.
func (s *Store) CreateMemoryUseFeedback(ctx context.Context, fb *domain.MemoryUseFeedback) error {
	query := `
		INSERT INTO memory_use_feedback (id, memory_use_id, rating, note, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.conn(ctx).Exec(ctx, query,
		fb.ID, fb.MemoryUseID, fb.Rating, fb.Note, fb.CreatedAt)
	if err != nil {
		return fmt.Errorf("create memory use feedback: %w", err)
	}
	return nil
}

// GetMemoryUseFeedback retrieves a memory use feedback by ID.
func (s *Store) GetMemoryUseFeedback(ctx context.Context, id string) (*domain.MemoryUseFeedback, error) {
	query := `
		SELECT id, memory_use_id, rating, note, created_at
		FROM memory_use_feedback
		WHERE id = $1`

	fb := &domain.MemoryUseFeedback{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&fb.ID, &fb.MemoryUseID, &fb.Rating, &fb.Note, &fb.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get memory use feedback: %w", err)
	}
	return fb, nil
}

// UpdateMemoryUseFeedback updates a memory use feedback's rating and note.
func (s *Store) UpdateMemoryUseFeedback(ctx context.Context, fb *domain.MemoryUseFeedback) error {
	query := `
		UPDATE memory_use_feedback
		SET rating = $2, note = $3
		WHERE id = $1`

	_, err := s.conn(ctx).Exec(ctx, query, fb.ID, fb.Rating, fb.Note)
	if err != nil {
		return fmt.Errorf("update memory use feedback: %w", err)
	}
	return nil
}

// DeleteMemoryUseFeedback deletes a memory use feedback.
func (s *Store) DeleteMemoryUseFeedback(ctx context.Context, id string) error {
	query := `DELETE FROM memory_use_feedback WHERE id = $1`
	result, err := s.conn(ctx).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete memory use feedback: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetMemoryUseFeedbackByMemoryUse returns all feedback for a memory use.
func (s *Store) GetMemoryUseFeedbackByMemoryUse(ctx context.Context, memoryUseID string) ([]*domain.MemoryUseFeedback, error) {
	query := `
		SELECT id, memory_use_id, rating, note, created_at
		FROM memory_use_feedback
		WHERE memory_use_id = $1
		ORDER BY created_at DESC`

	rows, err := s.conn(ctx).Query(ctx, query, memoryUseID)
	if err != nil {
		return nil, fmt.Errorf("get memory use feedback: %w", err)
	}
	defer rows.Close()

	var fbs []*domain.MemoryUseFeedback
	for rows.Next() {
		fb := &domain.MemoryUseFeedback{}
		if err := rows.Scan(&fb.ID, &fb.MemoryUseID, &fb.Rating, &fb.Note, &fb.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory use feedback: %w", err)
		}
		fbs = append(fbs, fb)
	}
	return fbs, rows.Err()
}

// ListMemoryUseFeedback returns all memory use feedback with pagination.
func (s *Store) ListMemoryUseFeedback(ctx context.Context, limit, offset int) ([]*domain.MemoryUseFeedback, int, error) {
	countQuery := `SELECT COUNT(*) FROM memory_use_feedback`
	var total int
	if err := s.conn(ctx).QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count memory use feedback: %w", err)
	}

	query := `
		SELECT id, memory_use_id, rating, note, created_at
		FROM memory_use_feedback
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list memory use feedback: %w", err)
	}
	defer rows.Close()

	var fbs []*domain.MemoryUseFeedback
	for rows.Next() {
		fb := &domain.MemoryUseFeedback{}
		if err := rows.Scan(&fb.ID, &fb.MemoryUseID, &fb.Rating, &fb.Note, &fb.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan memory use feedback: %w", err)
		}
		fbs = append(fbs, fb)
	}
	return fbs, total, rows.Err()
}
