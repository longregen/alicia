package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateMCPServer inserts a new MCP server.
func (s *Store) CreateMCPServer(ctx context.Context, server *domain.MCPServer) error {
	query := `
		INSERT INTO mcp_servers (id, name, transport_type, command, args, url, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := s.conn(ctx).Exec(ctx, query,
		server.ID, server.Name, server.TransportType,
		server.Command, server.Args, server.URL,
		server.Enabled, server.CreatedAt)
	if err != nil {
		return fmt.Errorf("create mcp server: %w", err)
	}
	return nil
}

// GetMCPServer retrieves an MCP server by ID.
func (s *Store) GetMCPServer(ctx context.Context, id string) (*domain.MCPServer, error) {
	query := `
		SELECT id, name, transport_type, command, args, url, enabled, created_at
		FROM mcp_servers
		WHERE id = $1 AND deleted_at IS NULL`

	server := &domain.MCPServer{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&server.ID, &server.Name, &server.TransportType,
		&server.Command, &server.Args, &server.URL,
		&server.Enabled, &server.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get mcp server: %w", err)
	}
	return server, nil
}

// GetMCPServerByName retrieves an MCP server by name.
func (s *Store) GetMCPServerByName(ctx context.Context, name string) (*domain.MCPServer, error) {
	query := `
		SELECT id, name, transport_type, command, args, url, enabled, created_at
		FROM mcp_servers
		WHERE name = $1 AND deleted_at IS NULL`

	server := &domain.MCPServer{}
	err := s.conn(ctx).QueryRow(ctx, query, name).Scan(
		&server.ID, &server.Name, &server.TransportType,
		&server.Command, &server.Args, &server.URL,
		&server.Enabled, &server.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get mcp server by name: %w", err)
	}
	return server, nil
}

// UpdateMCPServer updates an MCP server.
func (s *Store) UpdateMCPServer(ctx context.Context, server *domain.MCPServer) error {
	query := `
		UPDATE mcp_servers
		SET transport_type = $2, command = $3, args = $4, url = $5, enabled = $6
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := s.conn(ctx).Exec(ctx, query,
		server.ID, server.TransportType,
		server.Command, server.Args, server.URL, server.Enabled)
	if err != nil {
		return fmt.Errorf("update mcp server: %w", err)
	}
	return nil
}

// DeleteMCPServer soft-deletes an MCP server.
func (s *Store) DeleteMCPServer(ctx context.Context, id string) error {
	query := `UPDATE mcp_servers SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	result, err := s.conn(ctx).Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("delete mcp server: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteMCPServerByName soft-deletes an MCP server by name.
func (s *Store) DeleteMCPServerByName(ctx context.Context, name string) error {
	query := `UPDATE mcp_servers SET deleted_at = $2 WHERE name = $1 AND deleted_at IS NULL`
	result, err := s.conn(ctx).Exec(ctx, query, name, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("delete mcp server by name: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListMCPServers returns all active MCP servers.
func (s *Store) ListMCPServers(ctx context.Context) ([]*domain.MCPServer, error) {
	query := `
		SELECT id, name, transport_type, command, args, url, enabled, created_at
		FROM mcp_servers
		WHERE deleted_at IS NULL
		ORDER BY name`

	rows, err := s.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list mcp servers: %w", err)
	}
	defer rows.Close()

	var servers []*domain.MCPServer
	for rows.Next() {
		server := &domain.MCPServer{}
		if err := rows.Scan(
			&server.ID, &server.Name, &server.TransportType,
			&server.Command, &server.Args, &server.URL,
			&server.Enabled, &server.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mcp server: %w", err)
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}

// ListEnabledMCPServers returns all enabled MCP servers.
func (s *Store) ListEnabledMCPServers(ctx context.Context) ([]*domain.MCPServer, error) {
	query := `
		SELECT id, name, transport_type, command, args, url, enabled, created_at
		FROM mcp_servers
		WHERE deleted_at IS NULL AND enabled = true
		ORDER BY name`

	rows, err := s.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list enabled mcp servers: %w", err)
	}
	defer rows.Close()

	var servers []*domain.MCPServer
	for rows.Next() {
		server := &domain.MCPServer{}
		if err := rows.Scan(
			&server.ID, &server.Name, &server.TransportType,
			&server.Command, &server.Args, &server.URL,
			&server.Enabled, &server.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mcp server: %w", err)
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}
