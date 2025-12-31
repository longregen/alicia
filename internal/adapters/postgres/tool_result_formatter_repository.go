package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/ports"
)

// ToolResultFormatterRepository implements ports.ToolResultFormatterRepository
type ToolResultFormatterRepository struct {
	BaseRepository
}

// NewToolResultFormatterRepository creates a new tool result formatter repository
func NewToolResultFormatterRepository(pool *pgxpool.Pool) *ToolResultFormatterRepository {
	return &ToolResultFormatterRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

// Create stores a new result formatter
func (r *ToolResultFormatterRepository) Create(ctx context.Context, formatter *ports.ToolResultFormatter) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	keyFields, err := json.Marshal(formatter.KeyFields)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO tool_result_formatters (
			id, tool_name, template, max_length, summarize_at, summary_prompt, key_fields, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		formatter.ID,
		formatter.ToolName,
		formatter.Template,
		formatter.MaxLength,
		formatter.SummarizeAt,
		formatter.SummaryPrompt,
		keyFields,
		formatter.CreatedAt,
	)

	return err
}

// GetByToolName retrieves the formatter for a specific tool
func (r *ToolResultFormatterRepository) GetByToolName(ctx context.Context, toolName string) (*ports.ToolResultFormatter, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, tool_name, template, max_length, summarize_at, summary_prompt, key_fields, created_at, deleted_at
		FROM tool_result_formatters
		WHERE tool_name = $1 AND deleted_at IS NULL`

	return r.scanFormatter(r.conn(ctx).QueryRow(ctx, query, toolName))
}

// Update updates an existing formatter
func (r *ToolResultFormatterRepository) Update(ctx context.Context, formatter *ports.ToolResultFormatter) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	keyFields, err := json.Marshal(formatter.KeyFields)
	if err != nil {
		return err
	}

	query := `
		UPDATE tool_result_formatters
		SET template = $1,
		    max_length = $2,
		    summarize_at = $3,
		    summary_prompt = $4,
		    key_fields = $5
		WHERE tool_name = $6 AND deleted_at IS NULL`

	result, err := r.conn(ctx).Exec(ctx, query,
		formatter.Template,
		formatter.MaxLength,
		formatter.SummarizeAt,
		formatter.SummaryPrompt,
		keyFields,
		formatter.ToolName,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("tool result formatter not found")
	}

	return nil
}

// Delete soft-deletes a formatter
func (r *ToolResultFormatterRepository) Delete(ctx context.Context, toolName string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `UPDATE tool_result_formatters SET deleted_at = NOW() WHERE tool_name = $1 AND deleted_at IS NULL`
	result, err := r.conn(ctx).Exec(ctx, query, toolName)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("tool result formatter not found")
	}

	return nil
}

// List retrieves all active formatters
func (r *ToolResultFormatterRepository) List(ctx context.Context) ([]*ports.ToolResultFormatter, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, tool_name, template, max_length, summarize_at, summary_prompt, key_fields, created_at, deleted_at
		FROM tool_result_formatters
		WHERE deleted_at IS NULL
		ORDER BY tool_name`

	rows, err := r.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFormatters(rows)
}

func (r *ToolResultFormatterRepository) scanFormatter(row pgx.Row) (*ports.ToolResultFormatter, error) {
	var formatter ports.ToolResultFormatter
	var keyFields []byte
	var summaryPrompt sql.NullString
	var deletedAt sql.NullTime

	err := row.Scan(
		&formatter.ID,
		&formatter.ToolName,
		&formatter.Template,
		&formatter.MaxLength,
		&formatter.SummarizeAt,
		&summaryPrompt,
		&keyFields,
		&formatter.CreatedAt,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	if summaryPrompt.Valid {
		formatter.SummaryPrompt = summaryPrompt.String
	}

	if len(keyFields) > 0 {
		if err := json.Unmarshal(keyFields, &formatter.KeyFields); err != nil {
			formatter.KeyFields = make([]string, 0)
		}
	} else {
		formatter.KeyFields = make([]string, 0)
	}

	if deletedAt.Valid {
		formatter.DeletedAt = &deletedAt.Time
	}

	return &formatter, nil
}

func (r *ToolResultFormatterRepository) scanFormatters(rows pgx.Rows) ([]*ports.ToolResultFormatter, error) {
	formatters := make([]*ports.ToolResultFormatter, 0)

	for rows.Next() {
		var formatter ports.ToolResultFormatter
		var keyFields []byte
		var summaryPrompt sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&formatter.ID,
			&formatter.ToolName,
			&formatter.Template,
			&formatter.MaxLength,
			&formatter.SummarizeAt,
			&summaryPrompt,
			&keyFields,
			&formatter.CreatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, err
		}

		if summaryPrompt.Valid {
			formatter.SummaryPrompt = summaryPrompt.String
		}

		if len(keyFields) > 0 {
			if err := json.Unmarshal(keyFields, &formatter.KeyFields); err != nil {
				formatter.KeyFields = make([]string, 0)
			}
		} else {
			formatter.KeyFields = make([]string, 0)
		}

		if deletedAt.Valid {
			formatter.DeletedAt = &deletedAt.Time
		}

		formatters = append(formatters, &formatter)
	}

	return formatters, rows.Err()
}
