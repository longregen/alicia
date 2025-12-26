package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
)

// Version information (set via ldflags)
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// Shared global variables
var (
	cfg       *config.Config
	llmClient *llm.Client
)

// toolExecutorAdapter adapts ToolService to the ToolExecutor interface
type toolExecutorAdapter struct {
	toolService ports.ToolService
}

// Execute implements usecases.ToolExecutor interface
func (a *toolExecutorAdapter) Execute(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
	return a.toolService.ExecuteTool(ctx, tool.Name, arguments)
}

// initDB initializes a database connection pool for CLI commands
func initDB(ctx context.Context) (*pgxpool.Pool, error) {
	if cfg.Database.PostgresURL == "" {
		return nil, fmt.Errorf("PostgreSQL connection required. Set ALICIA_POSTGRES_URL")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Force UTC timezone to prevent timezone-related issues with TIMESTAMP columns
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return pool, nil
}

// maskSecret masks a secret string for display
func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 8 {
		return "(set)"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// boolStatus returns a status string for a boolean
func boolStatus(b bool) string {
	if b {
		return "configured"
	}
	return "not configured"
}

// closeMCPAdapter closes the MCP adapter connections
func closeMCPAdapter(adapter *mcp.Adapter) {
	if adapter != nil {
		log.Println("Closing MCP connections...")
		if err := adapter.Close(); err != nil {
			log.Printf("Warning: Failed to close MCP adapter: %v", err)
		}
	}
}

// initMCPAdapter initializes and configures the MCP adapter if servers are configured
func initMCPAdapter(ctx context.Context, toolService ports.ToolService, mcpRepo ports.MCPServerRepository, idGen ports.IDGenerator) *mcp.Adapter {
	log.Println("Initializing MCP adapter...")
	adapter := mcp.NewAdapter(ctx, toolService, mcpRepo, idGen)

	if err := adapter.InitializeServers(ctx, cfg.MCP.Servers); err != nil {
		log.Printf("Warning: Failed to initialize some MCP servers: %v", err)
	}

	serverStatus := adapter.GetServerStatus()
	connectedCount := 0
	for serverName, connected := range serverStatus {
		if connected {
			connectedCount++
			toolCount := len(adapter.GetServerTools(serverName))
			log.Printf("MCP server '%s' connected (%d tools)", serverName, toolCount)
		} else {
			log.Printf("MCP server '%s' disconnected", serverName)
		}
	}
	log.Printf("MCP initialization complete: %d/%d servers connected", connectedCount, len(cfg.MCP.Servers))

	return adapter
}
