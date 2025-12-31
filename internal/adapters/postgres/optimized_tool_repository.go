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

// OptimizedToolRepository implements ports.OptimizedToolRepository
type OptimizedToolRepository struct {
	BaseRepository
}

// NewOptimizedToolRepository creates a new optimized tool repository
func NewOptimizedToolRepository(pool *pgxpool.Pool) *OptimizedToolRepository {
	return &OptimizedToolRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

// Create stores a new optimized tool version
func (r *OptimizedToolRepository) Create(ctx context.Context, tool *ports.OptimizedTool) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	schema, err := json.Marshal(tool.OptimizedSchema)
	if err != nil {
		return err
	}

	examples, err := json.Marshal(tool.Examples)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO optimized_tools (
			id, tool_id, optimized_description, optimized_schema, result_template,
			examples, version, score, optimized_at, active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		tool.ID,
		tool.ToolID,
		tool.OptimizedDescription,
		schema,
		tool.ResultTemplate,
		examples,
		tool.Version,
		tool.Score,
		tool.OptimizedAt,
		tool.Active,
	)

	return err
}

// GetByID retrieves an optimized tool by ID
func (r *OptimizedToolRepository) GetByID(ctx context.Context, id string) (*ports.OptimizedTool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, tool_id, optimized_description, optimized_schema, result_template,
		       examples, version, score, optimized_at, active, deleted_at
		FROM optimized_tools
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanTool(r.conn(ctx).QueryRow(ctx, query, id))
}

// GetByToolID retrieves all versions for a tool
func (r *OptimizedToolRepository) GetByToolID(ctx context.Context, toolID string) ([]*ports.OptimizedTool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, tool_id, optimized_description, optimized_schema, result_template,
		       examples, version, score, optimized_at, active, deleted_at
		FROM optimized_tools
		WHERE tool_id = $1 AND deleted_at IS NULL
		ORDER BY version DESC`

	rows, err := r.conn(ctx).Query(ctx, query, toolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTools(rows)
}

// GetActiveByToolID retrieves the active optimized version for a tool
func (r *OptimizedToolRepository) GetActiveByToolID(ctx context.Context, toolID string) (*ports.OptimizedTool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, tool_id, optimized_description, optimized_schema, result_template,
		       examples, version, score, optimized_at, active, deleted_at
		FROM optimized_tools
		WHERE tool_id = $1 AND active = true AND deleted_at IS NULL`

	return r.scanTool(r.conn(ctx).QueryRow(ctx, query, toolID))
}

// GetLatestByToolID retrieves the most recent version for a tool
func (r *OptimizedToolRepository) GetLatestByToolID(ctx context.Context, toolID string) (*ports.OptimizedTool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, tool_id, optimized_description, optimized_schema, result_template,
		       examples, version, score, optimized_at, active, deleted_at
		FROM optimized_tools
		WHERE tool_id = $1 AND deleted_at IS NULL
		ORDER BY version DESC
		LIMIT 1`

	return r.scanTool(r.conn(ctx).QueryRow(ctx, query, toolID))
}

// SetActive marks a specific version as active (and deactivates others)
func (r *OptimizedToolRepository) SetActive(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// First get the tool to find its tool_id
	tool, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Deactivate all versions for this tool
	deactivateQuery := `
		UPDATE optimized_tools
		SET active = false
		WHERE tool_id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, deactivateQuery, tool.ToolID)
	if err != nil {
		return err
	}

	// Activate the specified version
	activateQuery := `
		UPDATE optimized_tools
		SET active = true
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.conn(ctx).Exec(ctx, activateQuery, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("optimized tool not found")
	}

	return nil
}

// Delete soft-deletes an optimized tool
func (r *OptimizedToolRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `UPDATE optimized_tools SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.conn(ctx).Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("optimized tool not found")
	}

	return nil
}

func (r *OptimizedToolRepository) scanTool(row pgx.Row) (*ports.OptimizedTool, error) {
	var tool ports.OptimizedTool
	var schema, examples []byte
	var score sql.NullFloat64
	var resultTemplate sql.NullString
	var deletedAt sql.NullTime

	err := row.Scan(
		&tool.ID,
		&tool.ToolID,
		&tool.OptimizedDescription,
		&schema,
		&resultTemplate,
		&examples,
		&tool.Version,
		&score,
		&tool.OptimizedAt,
		&tool.Active,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	if resultTemplate.Valid {
		tool.ResultTemplate = resultTemplate.String
	}

	if score.Valid {
		tool.Score = &score.Float64
	}

	if len(schema) > 0 {
		if err := json.Unmarshal(schema, &tool.OptimizedSchema); err != nil {
			tool.OptimizedSchema = make(map[string]any)
		}
	} else {
		tool.OptimizedSchema = make(map[string]any)
	}

	if len(examples) > 0 {
		if err := json.Unmarshal(examples, &tool.Examples); err != nil {
			tool.Examples = make([]map[string]any, 0)
		}
	} else {
		tool.Examples = make([]map[string]any, 0)
	}

	if deletedAt.Valid {
		tool.DeletedAt = &deletedAt.Time
	}

	return &tool, nil
}

func (r *OptimizedToolRepository) scanTools(rows pgx.Rows) ([]*ports.OptimizedTool, error) {
	tools := make([]*ports.OptimizedTool, 0)

	for rows.Next() {
		var tool ports.OptimizedTool
		var schema, examples []byte
		var score sql.NullFloat64
		var resultTemplate sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&tool.ID,
			&tool.ToolID,
			&tool.OptimizedDescription,
			&schema,
			&resultTemplate,
			&examples,
			&tool.Version,
			&score,
			&tool.OptimizedAt,
			&tool.Active,
			&deletedAt,
		)
		if err != nil {
			return nil, err
		}

		if resultTemplate.Valid {
			tool.ResultTemplate = resultTemplate.String
		}

		if score.Valid {
			tool.Score = &score.Float64
		}

		if len(schema) > 0 {
			if err := json.Unmarshal(schema, &tool.OptimizedSchema); err != nil {
				tool.OptimizedSchema = make(map[string]any)
			}
		} else {
			tool.OptimizedSchema = make(map[string]any)
		}

		if len(examples) > 0 {
			if err := json.Unmarshal(examples, &tool.Examples); err != nil {
				tool.Examples = make([]map[string]any, 0)
			}
		} else {
			tool.Examples = make([]map[string]any, 0)
		}

		if deletedAt.Valid {
			tool.DeletedAt = &deletedAt.Time
		}

		tools = append(tools, &tool)
	}

	return tools, rows.Err()
}
