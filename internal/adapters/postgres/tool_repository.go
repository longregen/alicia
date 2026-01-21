package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type ToolRepository struct {
	BaseRepository
}

func NewToolRepository(pool *pgxpool.Pool) *ToolRepository {
	return &ToolRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *ToolRepository) Create(ctx context.Context, tool *models.Tool) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	schema, err := json.Marshal(tool.Schema)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO alicia_tools (
			id, name, description, schema, enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		tool.ID,
		tool.Name,
		tool.Description,
		schema,
		tool.Enabled,
		tool.CreatedAt,
		tool.UpdatedAt,
	)

	return err
}

func (r *ToolRepository) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at, deleted_at
		FROM alicia_tools
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanTool(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *ToolRepository) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at, deleted_at
		FROM alicia_tools
		WHERE name = $1 AND deleted_at IS NULL`

	return r.scanTool(r.conn(ctx).QueryRow(ctx, query, name))
}

func (r *ToolRepository) Update(ctx context.Context, tool *models.Tool) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	schema, err := json.Marshal(tool.Schema)
	if err != nil {
		return err
	}

	query := `
		UPDATE alicia_tools
		SET name = $2,
			description = $3,
			schema = $4,
			enabled = $5,
			updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, query,
		tool.ID,
		tool.Name,
		tool.Description,
		schema,
		tool.Enabled,
		tool.UpdatedAt,
	)

	return err
}

func (r *ToolRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_tools
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *ToolRepository) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at, deleted_at
		FROM alicia_tools
		WHERE enabled = true AND deleted_at IS NULL
		ORDER BY name ASC`

	rows, err := r.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTools(rows)
}

func (r *ToolRepository) ListAll(ctx context.Context) ([]*models.Tool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at, deleted_at
		FROM alicia_tools
		WHERE deleted_at IS NULL
		ORDER BY name ASC`

	rows, err := r.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTools(rows)
}

func (r *ToolRepository) scanTool(row pgx.Row) (*models.Tool, error) {
	var t models.Tool
	var schema []byte

	err := row.Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&schema,
		&t.Enabled,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.DeletedAt,
	)

	if err != nil {
		if checkNoRows(err) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	if err := unmarshalJSONField(schema, &t.Schema); err != nil {
		return nil, err
	}

	return &t, nil
}

func (r *ToolRepository) scanTools(rows pgx.Rows) ([]*models.Tool, error) {
	var tools []*models.Tool

	for rows.Next() {
		var t models.Tool
		var schema []byte

		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&schema,
			&t.Enabled,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := unmarshalJSONField(schema, &t.Schema); err != nil {
			return nil, err
		}

		tools = append(tools, &t)
	}

	return tools, rows.Err()
}
