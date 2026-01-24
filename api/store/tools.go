package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateTool inserts a new tool.
func (s *Store) CreateTool(ctx context.Context, tool *domain.Tool) error {
	query := `
		INSERT INTO tools (id, name, description, schema, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.conn(ctx).Exec(ctx, query,
		tool.ID, tool.Name, tool.Description, tool.Schema,
		tool.Enabled, tool.CreatedAt, tool.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tool: %w", err)
	}
	return nil
}

// GetTool retrieves a tool by ID.
func (s *Store) GetTool(ctx context.Context, id string) (*domain.Tool, error) {
	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at
		FROM tools
		WHERE id = $1`

	tool := &domain.Tool{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&tool.ID, &tool.Name, &tool.Description, &tool.Schema,
		&tool.Enabled, &tool.CreatedAt, &tool.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get tool: %w", err)
	}
	return tool, nil
}

// GetToolByName retrieves a tool by name.
func (s *Store) GetToolByName(ctx context.Context, name string) (*domain.Tool, error) {
	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at
		FROM tools
		WHERE name = $1`

	tool := &domain.Tool{}
	err := s.conn(ctx).QueryRow(ctx, query, name).Scan(
		&tool.ID, &tool.Name, &tool.Description, &tool.Schema,
		&tool.Enabled, &tool.CreatedAt, &tool.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get tool by name: %w", err)
	}
	return tool, nil
}

// UpdateTool updates a tool.
func (s *Store) UpdateTool(ctx context.Context, tool *domain.Tool) error {
	query := `
		UPDATE tools
		SET description = $2, schema = $3, enabled = $4, updated_at = $5
		WHERE id = $1`

	tool.UpdatedAt = time.Now().UTC()
	_, err := s.conn(ctx).Exec(ctx, query,
		tool.ID, tool.Description, tool.Schema, tool.Enabled, tool.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update tool: %w", err)
	}
	return nil
}

// ListTools returns all tools.
func (s *Store) ListTools(ctx context.Context) ([]*domain.Tool, error) {
	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at
		FROM tools
		ORDER BY name`

	rows, err := s.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	defer rows.Close()

	return scanTools(rows)
}

// ListEnabledTools returns all enabled tools.
func (s *Store) ListEnabledTools(ctx context.Context) ([]*domain.Tool, error) {
	query := `
		SELECT id, name, description, schema, enabled, created_at, updated_at
		FROM tools
		WHERE enabled = true
		ORDER BY name`

	rows, err := s.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list enabled tools: %w", err)
	}
	defer rows.Close()

	return scanTools(rows)
}

func scanTools(rows pgx.Rows) ([]*domain.Tool, error) {
	var tools []*domain.Tool
	for rows.Next() {
		tool := &domain.Tool{}
		if err := rows.Scan(
			&tool.ID, &tool.Name, &tool.Description, &tool.Schema,
			&tool.Enabled, &tool.CreatedAt, &tool.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tool: %w", err)
		}
		tools = append(tools, tool)
	}
	return tools, rows.Err()
}

// --- Tool Uses ---

// CreateToolUse inserts a new tool use.
func (s *Store) CreateToolUse(ctx context.Context, tu *domain.ToolUse) error {
	query := `
		INSERT INTO tool_uses (id, message_id, tool_name, arguments, result, status, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := s.conn(ctx).Exec(ctx, query,
		tu.ID, tu.MessageID, tu.ToolName, tu.Arguments,
		tu.Result, tu.Status, tu.Error, tu.CreatedAt)
	if err != nil {
		return fmt.Errorf("create tool use: %w", err)
	}
	return nil
}

// GetToolUse retrieves a tool use by ID.
func (s *Store) GetToolUse(ctx context.Context, id string) (*domain.ToolUse, error) {
	query := `
		SELECT id, message_id, tool_name, arguments, result, status, error, created_at
		FROM tool_uses
		WHERE id = $1`

	tu := &domain.ToolUse{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&tu.ID, &tu.MessageID, &tu.ToolName, &tu.Arguments,
		&tu.Result, &tu.Status, &tu.Error, &tu.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get tool use: %w", err)
	}
	return tu, nil
}

// UpdateToolUse updates a tool use's result and status.
func (s *Store) UpdateToolUse(ctx context.Context, tu *domain.ToolUse) error {
	query := `
		UPDATE tool_uses
		SET result = $2, status = $3, error = $4
		WHERE id = $1`

	_, err := s.conn(ctx).Exec(ctx, query, tu.ID, tu.Result, tu.Status, tu.Error)
	if err != nil {
		return fmt.Errorf("update tool use: %w", err)
	}
	return nil
}

// GetToolUsesByMessage returns tool uses for a message.
func (s *Store) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*domain.ToolUse, error) {
	query := `
		SELECT id, message_id, tool_name, arguments, result, status, error, created_at
		FROM tool_uses
		WHERE message_id = $1
		ORDER BY created_at`

	rows, err := s.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("get tool uses: %w", err)
	}
	defer rows.Close()

	return scanToolUses(rows)
}

// ListToolUses returns all tool uses with pagination and total count.
func (s *Store) ListToolUses(ctx context.Context, limit, offset int) ([]*domain.ToolUse, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM tool_uses`
	var total int
	if err := s.conn(ctx).QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tool uses: %w", err)
	}

	query := `
		SELECT id, message_id, tool_name, arguments, result, status, error, created_at
		FROM tool_uses
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tool uses: %w", err)
	}
	defer rows.Close()

	uses, err := scanToolUses(rows)
	if err != nil {
		return nil, 0, err
	}
	return uses, total, nil
}

func scanToolUses(rows pgx.Rows) ([]*domain.ToolUse, error) {
	var uses []*domain.ToolUse
	for rows.Next() {
		tu := &domain.ToolUse{}
		if err := rows.Scan(
			&tu.ID, &tu.MessageID, &tu.ToolName, &tu.Arguments,
			&tu.Result, &tu.Status, &tu.Error, &tu.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tool use: %w", err)
		}
		uses = append(uses, tu)
	}
	return uses, rows.Err()
}
