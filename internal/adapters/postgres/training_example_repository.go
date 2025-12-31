package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// TrainingExampleRepository implements ports.TrainingExampleRepository
type TrainingExampleRepository struct {
	BaseRepository
	idGenerator ports.IDGenerator
}

// NewTrainingExampleRepository creates a new training example repository
func NewTrainingExampleRepository(pool *pgxpool.Pool, idGenerator ports.IDGenerator) *TrainingExampleRepository {
	return &TrainingExampleRepository{
		BaseRepository: NewBaseRepository(pool),
		idGenerator:    idGenerator,
	}
}

// Create creates a new training example
func (r *TrainingExampleRepository) Create(ctx context.Context, example *models.TrainingExample) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Marshal JSON fields
	inputs, err := json.Marshal(example.Inputs)
	if err != nil {
		return err
	}

	outputs, err := json.Marshal(example.Outputs)
	if err != nil {
		return err
	}

	var voteMetadata []byte
	if example.VoteMetadata != nil {
		voteMetadata, err = json.Marshal(example.VoteMetadata)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO gepa_training_examples (
			id, task_type, vote_id, is_positive, inputs, outputs, vote_metadata, source, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		example.ID,
		example.TaskType,
		example.VoteID,
		example.IsPositive,
		inputs,
		outputs,
		voteMetadata,
		example.Source,
		example.CreatedAt,
	)

	return err
}

// GetByID retrieves a training example by ID
func (r *TrainingExampleRepository) GetByID(ctx context.Context, id string) (*models.TrainingExample, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, task_type, vote_id, is_positive, inputs, outputs, vote_metadata, source, created_at, deleted_at
		FROM gepa_training_examples
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanExample(r.conn(ctx).QueryRow(ctx, query, id))
}

// ListByTaskType retrieves training examples by task type with pagination
func (r *TrainingExampleRepository) ListByTaskType(ctx context.Context, taskType string, limit, offset int) ([]*models.TrainingExample, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, task_type, vote_id, is_positive, inputs, outputs, vote_metadata, source, created_at, deleted_at
		FROM gepa_training_examples
		WHERE task_type = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.conn(ctx).Query(ctx, query, taskType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanExamples(rows)
}

// CountByTaskType returns count of training examples for a task type
func (r *TrainingExampleRepository) CountByTaskType(ctx context.Context, taskType string) (int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `SELECT COUNT(*) FROM gepa_training_examples WHERE task_type = $1 AND deleted_at IS NULL`

	var count int
	err := r.conn(ctx).QueryRow(ctx, query, taskType).Scan(&count)
	return count, err
}

// CountPositiveByTaskType returns count of positive training examples for a task type
func (r *TrainingExampleRepository) CountPositiveByTaskType(ctx context.Context, taskType string) (int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `SELECT COUNT(*) FROM gepa_training_examples WHERE task_type = $1 AND is_positive = true AND deleted_at IS NULL`

	var count int
	err := r.conn(ctx).QueryRow(ctx, query, taskType).Scan(&count)
	return count, err
}

// Delete soft-deletes a training example by ID
func (r *TrainingExampleRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE gepa_training_examples
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.conn(ctx).Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("training example not found")
	}

	return nil
}

// DeleteByVoteID soft-deletes training examples by vote ID
func (r *TrainingExampleRepository) DeleteByVoteID(ctx context.Context, voteID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE gepa_training_examples
		SET deleted_at = NOW()
		WHERE vote_id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, voteID)
	return err
}

// scanExample scans a single training example from a row
func (r *TrainingExampleRepository) scanExample(row pgx.Row) (*models.TrainingExample, error) {
	var example models.TrainingExample
	var voteID sql.NullString
	var inputs, outputs, voteMetadata []byte
	var deletedAt sql.NullTime

	err := row.Scan(
		&example.ID,
		&example.TaskType,
		&voteID,
		&example.IsPositive,
		&inputs,
		&outputs,
		&voteMetadata,
		&example.Source,
		&example.CreatedAt,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	// Unmarshal JSON fields
	if len(inputs) > 0 {
		if err := json.Unmarshal(inputs, &example.Inputs); err != nil {
			example.Inputs = make(map[string]any)
		}
	} else {
		example.Inputs = make(map[string]any)
	}

	if len(outputs) > 0 {
		if err := json.Unmarshal(outputs, &example.Outputs); err != nil {
			example.Outputs = make(map[string]any)
		}
	} else {
		example.Outputs = make(map[string]any)
	}

	if len(voteMetadata) > 0 {
		var vm models.VoteMetadata
		if err := json.Unmarshal(voteMetadata, &vm); err == nil {
			example.VoteMetadata = &vm
		}
	}

	example.VoteID = getStringPtr(voteID)
	example.DeletedAt = getTimePtr(deletedAt)

	return &example, nil
}

// scanExamples scans multiple training examples from rows
func (r *TrainingExampleRepository) scanExamples(rows pgx.Rows) ([]*models.TrainingExample, error) {
	examples := make([]*models.TrainingExample, 0)

	for rows.Next() {
		var example models.TrainingExample
		var voteID sql.NullString
		var inputs, outputs, voteMetadata []byte
		var deletedAt sql.NullTime

		err := rows.Scan(
			&example.ID,
			&example.TaskType,
			&voteID,
			&example.IsPositive,
			&inputs,
			&outputs,
			&voteMetadata,
			&example.Source,
			&example.CreatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if len(inputs) > 0 {
			if err := json.Unmarshal(inputs, &example.Inputs); err != nil {
				example.Inputs = make(map[string]any)
			}
		} else {
			example.Inputs = make(map[string]any)
		}

		if len(outputs) > 0 {
			if err := json.Unmarshal(outputs, &example.Outputs); err != nil {
				example.Outputs = make(map[string]any)
			}
		} else {
			example.Outputs = make(map[string]any)
		}

		if len(voteMetadata) > 0 {
			var vm models.VoteMetadata
			if err := json.Unmarshal(voteMetadata, &vm); err == nil {
				example.VoteMetadata = &vm
			}
		}

		example.VoteID = getStringPtr(voteID)
		example.DeletedAt = getTimePtr(deletedAt)

		examples = append(examples, &example)
	}

	return examples, rows.Err()
}
