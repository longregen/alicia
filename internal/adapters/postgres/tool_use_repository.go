package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type ToolUseRepository struct {
	BaseRepository
}

func NewToolUseRepository(pool *pgxpool.Pool) *ToolUseRepository {
	return &ToolUseRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *ToolUseRepository) Create(ctx context.Context, toolUse *models.ToolUse) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	arguments, err := json.Marshal(toolUse.Arguments)
	if err != nil {
		return err
	}

	var result []byte
	if toolUse.Result != nil {
		result, err = json.Marshal(toolUse.Result)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO alicia_tool_uses (
			id, message_id, tool_name, tool_arguments, tool_result, status,
			error_message, sequence_number, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		toolUse.ID,
		toolUse.MessageID,
		toolUse.ToolName,
		arguments,
		result,
		toolUse.Status,
		nullString(toolUse.ErrorMessage),
		toolUse.SequenceNumber,
		toolUse.CompletedAt,
		toolUse.CreatedAt,
		toolUse.UpdatedAt,
	)

	return err
}

func (r *ToolUseRepository) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, tool_name, tool_arguments, tool_result, status,
			   error_message, sequence_number, completed_at, created_at, updated_at, deleted_at
		FROM alicia_tool_uses
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanToolUse(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *ToolUseRepository) Update(ctx context.Context, toolUse *models.ToolUse) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var result []byte
	var err error
	if toolUse.Result != nil {
		result, err = json.Marshal(toolUse.Result)
		if err != nil {
			return err
		}
	}

	query := `
		UPDATE alicia_tool_uses
		SET tool_result = $2,
			status = $3,
			error_message = $4,
			completed_at = $5,
			updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, query,
		toolUse.ID,
		result,
		toolUse.Status,
		nullString(toolUse.ErrorMessage),
		toolUse.CompletedAt,
		toolUse.UpdatedAt,
	)

	return err
}

func (r *ToolUseRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, tool_name, tool_arguments, tool_result, status,
			   error_message, sequence_number, completed_at, created_at, updated_at, deleted_at
		FROM alicia_tool_uses
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY sequence_number ASC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanToolUses(rows)
}

func (r *ToolUseRepository) GetPending(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, tool_name, tool_arguments, tool_result, status,
			   error_message, sequence_number, completed_at, created_at, updated_at, deleted_at
		FROM alicia_tool_uses
		WHERE status = 'pending' AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.conn(ctx).Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanToolUses(rows)
}

func (r *ToolUseRepository) scanToolUse(row pgx.Row) (*models.ToolUse, error) {
	var tu models.ToolUse
	var arguments, result []byte
	var errorMessage sql.NullString

	err := row.Scan(
		&tu.ID,
		&tu.MessageID,
		&tu.ToolName,
		&arguments,
		&result,
		&tu.Status,
		&errorMessage,
		&tu.SequenceNumber,
		&tu.CompletedAt,
		&tu.CreatedAt,
		&tu.UpdatedAt,
		&tu.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &tu.Arguments); err != nil {
			return nil, err
		}
	}

	if len(result) > 0 {
		var res any
		if err := json.Unmarshal(result, &res); err != nil {
			return nil, err
		}
		tu.Result = res
	}

	if errorMessage.Valid {
		tu.ErrorMessage = errorMessage.String
	}

	return &tu, nil
}

func (r *ToolUseRepository) scanToolUses(rows pgx.Rows) ([]*models.ToolUse, error) {
	var toolUses []*models.ToolUse

	for rows.Next() {
		var tu models.ToolUse
		var arguments, result []byte
		var errorMessage sql.NullString

		err := rows.Scan(
			&tu.ID,
			&tu.MessageID,
			&tu.ToolName,
			&arguments,
			&result,
			&tu.Status,
			&errorMessage,
			&tu.SequenceNumber,
			&tu.CompletedAt,
			&tu.CreatedAt,
			&tu.UpdatedAt,
			&tu.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(arguments) > 0 {
			if err := json.Unmarshal(arguments, &tu.Arguments); err != nil {
				return nil, err
			}
		}

		if len(result) > 0 {
			var res any
			if err := json.Unmarshal(result, &res); err != nil {
				return nil, err
			}
			tu.Result = res
		}

		if errorMessage.Valid {
			tu.ErrorMessage = errorMessage.String
		}

		toolUses = append(toolUses, &tu)
	}

	return toolUses, rows.Err()
}
