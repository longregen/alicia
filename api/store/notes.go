package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
	pgvector "github.com/pgvector/pgvector-go"
)

func (s *Store) CreateNote(ctx context.Context, note *domain.Note) error {
	query := `
		INSERT INTO notes (id, user_id, title, content, embedding, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	var embedding *pgvector.Vector
	if len(note.Embedding) > 0 {
		v := pgvector.NewVector(note.Embedding)
		embedding = &v
	}

	_, err := s.conn(ctx).Exec(ctx, query,
		note.ID, note.UserID, note.Title, note.Content, embedding,
		note.CreatedAt, note.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	return nil
}

func (s *Store) GetNote(ctx context.Context, id string) (*domain.Note, error) {
	query := `
		SELECT id, user_id, title, content, created_at, updated_at
		FROM notes
		WHERE id = $1 AND deleted_at IS NULL`

	note := &domain.Note{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&note.ID, &note.UserID, &note.Title, &note.Content,
		&note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	return note, nil
}

func (s *Store) ListNotesByUser(ctx context.Context, userID string) ([]*domain.Note, error) {
	query := `
		SELECT id, user_id, title, content, created_at, updated_at
		FROM notes
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC`

	rows, err := s.conn(ctx).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []*domain.Note
	for rows.Next() {
		note := &domain.Note{}
		if err := rows.Scan(&note.ID, &note.UserID, &note.Title, &note.Content,
			&note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, note)
	}
	return notes, nil
}

func (s *Store) UpdateNote(ctx context.Context, note *domain.Note) error {
	query := `
		UPDATE notes
		SET title = $1, content = $2, embedding = $3, updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL`

	var embedding *pgvector.Vector
	if len(note.Embedding) > 0 {
		v := pgvector.NewVector(note.Embedding)
		embedding = &v
	}

	tag, err := s.conn(ctx).Exec(ctx, query,
		note.Title, note.Content, embedding, note.UpdatedAt, note.ID)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("note not found")
	}
	return nil
}

func (s *Store) DeleteNote(ctx context.Context, id string) error {
	query := `UPDATE notes SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	tag, err := s.conn(ctx).Exec(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("note not found")
	}
	return nil
}

