package services

import (
	"context"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
)

// MCPService manages MCP server configurations.
type MCPService struct {
	store *store.Store
}

// NewMCPService creates a new MCP service.
func NewMCPService(s *store.Store) *MCPService {
	return &MCPService{store: s}
}

// CreateServer creates a new MCP server configuration.
func (svc *MCPService) CreateServer(ctx context.Context, name, transportType, command string, args []string, url string) (*domain.MCPServer, error) {
	server := &domain.MCPServer{
		ID:            store.NewMCPServerID(),
		Name:          name,
		TransportType: transportType,
		Command:       command,
		Args:          args,
		URL:           url,
		Enabled:       true,
		CreatedAt:     time.Now().UTC(),
	}

	if err := svc.store.CreateMCPServer(ctx, server); err != nil {
		return nil, err
	}
	return server, nil
}

// GetServer retrieves an MCP server by ID.
func (svc *MCPService) GetServer(ctx context.Context, id string) (*domain.MCPServer, error) {
	return svc.store.GetMCPServer(ctx, id)
}

// GetServerByName retrieves an MCP server by name.
func (svc *MCPService) GetServerByName(ctx context.Context, name string) (*domain.MCPServer, error) {
	return svc.store.GetMCPServerByName(ctx, name)
}

// UpdateServer updates an MCP server configuration.
func (svc *MCPService) UpdateServer(ctx context.Context, server *domain.MCPServer) error {
	return svc.store.UpdateMCPServer(ctx, server)
}

// DeleteServer soft-deletes an MCP server.
func (svc *MCPService) DeleteServer(ctx context.Context, id string) error {
	return svc.store.DeleteMCPServer(ctx, id)
}

// DeleteServerByName soft-deletes an MCP server by name.
func (svc *MCPService) DeleteServerByName(ctx context.Context, name string) error {
	return svc.store.DeleteMCPServerByName(ctx, name)
}

// ListServers returns all active MCP servers.
func (svc *MCPService) ListServers(ctx context.Context) ([]*domain.MCPServer, error) {
	return svc.store.ListMCPServers(ctx)
}

// ListEnabledServers returns all enabled MCP servers.
func (svc *MCPService) ListEnabledServers(ctx context.Context) ([]*domain.MCPServer, error) {
	return svc.store.ListEnabledMCPServers(ctx)
}

// EnableServer enables an MCP server.
func (svc *MCPService) EnableServer(ctx context.Context, id string) error {
	server, err := svc.store.GetMCPServer(ctx, id)
	if err != nil {
		return err
	}
	server.Enabled = true
	return svc.store.UpdateMCPServer(ctx, server)
}

// DisableServer disables an MCP server.
func (svc *MCPService) DisableServer(ctx context.Context, id string) error {
	server, err := svc.store.GetMCPServer(ctx, id)
	if err != nil {
		return err
	}
	server.Enabled = false
	return svc.store.UpdateMCPServer(ctx, server)
}
