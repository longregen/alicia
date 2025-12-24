package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type ReasoningStepRepository struct {
	BaseRepository
}

func NewReasoningStepRepository(pool *pgxpool.Pool) *ReasoningStepRepository {
	return &ReasoningStepRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *ReasoningStepRepository) Create(ctx context.Context, step *models.ReasoningStep) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO alicia_reasoning_steps (
			id, message_id, content, sequence_number, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	_, err := r.conn(ctx).Exec(ctx, query,
		step.ID,
		step.MessageID,
		step.Content,
		step.SequenceNumber,
		step.CreatedAt,
		step.UpdatedAt,
	)

	return err
}

func (r *ReasoningStepRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.ReasoningStep, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, content, sequence_number, created_at, updated_at, deleted_at
		FROM alicia_reasoning_steps
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY sequence_number ASC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*models.ReasoningStep

	for rows.Next() {
		var s models.ReasoningStep

		err := rows.Scan(
			&s.ID,
			&s.MessageID,
			&s.Content,
			&s.SequenceNumber,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		steps = append(steps, &s)
	}

	return steps, rows.Err()
}

func (r *ReasoningStepRepository) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT COALESCE(MAX(sequence_number), 0) + 1 as next_sequence
		FROM alicia_reasoning_steps
		WHERE message_id = $1 AND deleted_at IS NULL`

	var nextSeq int
	err := r.conn(ctx).QueryRow(ctx, query, messageID).Scan(&nextSeq)
	if err != nil {
		return 0, err
	}

	return nextSeq, nil
}
