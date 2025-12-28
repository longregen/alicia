package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type NoteRepository struct {
	BaseRepository
}

func NewNoteRepository(pool *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *NoteRepository) Create(ctx context.Context, note *models.Note) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO alicia_notes (
			id, message_id, content, category, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	category := note.Category
	if category == "" {
		category = models.NoteCategoryGeneral
	}

	_, err := r.conn(ctx).Exec(ctx, query,
		note.ID,
		note.MessageID,
		note.Content,
		category,
		note.CreatedAt,
		note.UpdatedAt,
	)

	return err
}

func (r *NoteRepository) Update(ctx context.Context, id string, content string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_notes
		SET content = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.conn(ctx).Exec(ctx, query, content, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *NoteRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_notes
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *NoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Note, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, content, category, created_at, updated_at
		FROM alicia_notes
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanNotes(rows)
}

func (r *NoteRepository) GetByID(ctx context.Context, id string) (*models.Note, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, content, category, created_at, updated_at
		FROM alicia_notes
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanNote(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *NoteRepository) scanNote(row pgx.Row) (*models.Note, error) {
	var note models.Note

	err := row.Scan(
		&note.ID,
		&note.MessageID,
		&note.Content,
		&note.Category,
		&note.CreatedAt,
		&note.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	return &note, nil
}

func (r *NoteRepository) scanNotes(rows pgx.Rows) ([]*models.Note, error) {
	var notes []*models.Note

	for rows.Next() {
		var note models.Note

		err := rows.Scan(
			&note.ID,
			&note.MessageID,
			&note.Content,
			&note.Category,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		notes = append(notes, &note)
	}

	return notes, rows.Err()
}
