package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type MCPServerRepository struct {
	BaseRepository
}

func NewMCPServerRepository(pool *pgxpool.Pool) *MCPServerRepository {
	return &MCPServerRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *MCPServerRepository) Create(ctx context.Context, server *models.MCPServer) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var env []byte
	var err error
	if len(server.Env) > 0 {
		env, err = json.Marshal(server.Env)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO alicia_mcp_servers (
			id, name, transport_type, command, args, env, url, api_key,
			auto_reconnect, reconnect_delay, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		server.ID,
		server.Name,
		server.TransportType,
		nullString(server.Command),
		server.Args,
		env,
		nullString(server.URL),
		nullString(server.APIKey),
		server.AutoReconnect,
		server.ReconnectDelay,
		server.CreatedAt,
		server.UpdatedAt,
	)

	return err
}

func (r *MCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServer, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, transport_type, command, args, env, url, api_key,
		       auto_reconnect, reconnect_delay, created_at, updated_at
		FROM alicia_mcp_servers
		WHERE id = $1`

	return r.scanMCPServer(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *MCPServerRepository) GetByName(ctx context.Context, name string) (*models.MCPServer, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, transport_type, command, args, env, url, api_key,
		       auto_reconnect, reconnect_delay, created_at, updated_at
		FROM alicia_mcp_servers
		WHERE name = $1`

	return r.scanMCPServer(r.conn(ctx).QueryRow(ctx, query, name))
}

func (r *MCPServerRepository) Update(ctx context.Context, server *models.MCPServer) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var env []byte
	var err error
	if len(server.Env) > 0 {
		env, err = json.Marshal(server.Env)
		if err != nil {
			return err
		}
	}

	query := `
		UPDATE alicia_mcp_servers
		SET name = $2,
			transport_type = $3,
			command = $4,
			args = $5,
			env = $6,
			url = $7,
			api_key = $8,
			auto_reconnect = $9,
			reconnect_delay = $10,
			updated_at = $11
		WHERE id = $1`

	_, err = r.conn(ctx).Exec(ctx, query,
		server.ID,
		server.Name,
		server.TransportType,
		nullString(server.Command),
		server.Args,
		env,
		nullString(server.URL),
		nullString(server.APIKey),
		server.AutoReconnect,
		server.ReconnectDelay,
		server.UpdatedAt,
	)

	return err
}

func (r *MCPServerRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `DELETE FROM alicia_mcp_servers WHERE id = $1`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *MCPServerRepository) List(ctx context.Context) ([]*models.MCPServer, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, name, transport_type, command, args, env, url, api_key,
		       auto_reconnect, reconnect_delay, created_at, updated_at
		FROM alicia_mcp_servers
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMCPServers(rows)
}

func (r *MCPServerRepository) scanMCPServer(row pgx.Row) (*models.MCPServer, error) {
	var s models.MCPServer
	var env []byte
	var command, url, apiKey sql.NullString

	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.TransportType,
		&command,
		&s.Args,
		&env,
		&url,
		&apiKey,
		&s.AutoReconnect,
		&s.ReconnectDelay,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if checkNoRows(err) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	s.Command = getString(command)
	s.URL = getString(url)
	s.APIKey = getString(apiKey)

	s.Env, err = unmarshalJSONSlice[string](env)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *MCPServerRepository) scanMCPServers(rows pgx.Rows) ([]*models.MCPServer, error) {
	var servers []*models.MCPServer

	for rows.Next() {
		var s models.MCPServer
		var env []byte
		var command, url, apiKey sql.NullString

		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.TransportType,
			&command,
			&s.Args,
			&env,
			&url,
			&apiKey,
			&s.AutoReconnect,
			&s.ReconnectDelay,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		s.Command = getString(command)
		s.URL = getString(url)
		s.APIKey = getString(apiKey)

		s.Env, err = unmarshalJSONSlice[string](env)
		if err != nil {
			return nil, err
		}

		servers = append(servers, &s)
	}

	return servers, rows.Err()
}
