package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Adapter integrates MCP servers with Alicia's tool service
type Adapter struct {
	manager     *Manager
	toolService ports.ToolService
	mcpRepo     ports.MCPServerRepository
	idGen       ports.IDGenerator
	serverTools map[string][]string // Maps server name to tool names
}

// NewAdapter creates a new MCP adapter
func NewAdapter(ctx context.Context, toolService ports.ToolService, mcpRepo ports.MCPServerRepository, idGen ports.IDGenerator) *Adapter {
	adapter := &Adapter{
		manager:     NewManager(ctx),
		toolService: toolService,
		mcpRepo:     mcpRepo,
		idGen:       idGen,
		serverTools: make(map[string][]string),
	}

	// Set up connection callback to handle executor registration/unregistration
	adapter.manager.SetConnectionCallback(adapter.onConnectionChange)

	return adapter
}

// onConnectionChange handles MCP server connection state changes.
// When a server disconnects, it unregisters its tool executors so they won't be
// offered to the LLM during response generation.
// When a server reconnects, it re-discovers and re-registers its tools.
func (a *Adapter) onConnectionChange(serverName string, connected bool) {
	if connected {
		log.Printf("MCP server %s connected, re-registering tools", serverName)
		// On reconnection, re-discover and register tools
		client, err := a.manager.GetClient(serverName)
		if err != nil {
			log.Printf("Warning: Failed to get client for reconnected server %s: %v", serverName, err)
			return
		}
		if err := a.discoverAndRegisterTools(context.Background(), serverName, client); err != nil {
			log.Printf("Warning: Failed to re-register tools for server %s: %v", serverName, err)
		}
	} else {
		log.Printf("MCP server %s disconnected, unregistering tool executors", serverName)
		// On disconnection, unregister executors for all tools from this server
		if toolNames, exists := a.serverTools[serverName]; exists {
			for _, toolName := range toolNames {
				a.toolService.UnregisterExecutor(toolName)
			}
		}
	}
}

// InitializeServers initializes all configured MCP servers and loads from database
func (a *Adapter) InitializeServers(ctx context.Context, configs []config.MCPServerConfig) error {
	// Auto-populate built-in MCP servers
	a.autoPopulateBuiltinServers(ctx)

	// First, load servers from database
	dbServers, err := a.mcpRepo.List(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load MCP servers from database: %v", err)
	} else {
		for _, server := range dbServers {
			cfg := config.MCPServerConfig{
				Name:           server.Name,
				Transport:      server.TransportType,
				Command:        server.Command,
				Args:           server.Args,
				Env:            server.Env,
				URL:            server.URL,
				APIKey:         server.APIKey,
				AutoReconnect:  server.AutoReconnect,
				ReconnectDelay: server.ReconnectDelay,
			}
			if err := a.addServerToManager(ctx, cfg); err != nil {
				log.Printf("Warning: Failed to initialize MCP server %s from database: %v", cfg.Name, err)
			}
		}
	}

	// Then add servers from config (these won't be persisted unless explicitly added via API)
	for _, cfg := range configs {
		if err := a.addServerToManager(ctx, cfg); err != nil {
			log.Printf("Warning: Failed to initialize MCP server %s: %v", cfg.Name, err)
			continue
		}
	}
	return nil
}

// builtinMCPServer defines a built-in MCP server that can be auto-populated
type builtinMCPServer struct {
	Name       string
	Command    string
	EnvVars    []string // Environment variable names to check/pass through
	RequireEnv string   // If set, only populate if this env var is set
}

// builtinServers defines the built-in MCP servers that are auto-populated
var builtinServers = []builtinMCPServer{
	{
		Name:       "garden",
		Command:    "mcp-garden",
		RequireEnv: "GARDEN_DATABASE_URL",
		EnvVars: []string{
			"GARDEN_DATABASE_URL",
			"DATABASE_DOC_PATH",
			"MCP_MAX_CHARACTER_RESPONSE_SIZE",
			"LLM_URL",
			"LLM_API_KEY",
			"LLM_MODEL",
			"LLM_DEFAULT_MAX_TOKENS",
		},
	},
	{
		Name:    "web",
		Command: "mcp-web",
		EnvVars: []string{},
	},
}

// autoPopulateBuiltinServers creates built-in MCP servers if they don't exist
// and haven't been previously deleted
func (a *Adapter) autoPopulateBuiltinServers(ctx context.Context) {
	for _, builtin := range builtinServers {
		if err := a.autoPopulateServer(ctx, builtin); err != nil {
			log.Printf("Warning: Failed to auto-populate %s MCP server: %v", builtin.Name, err)
		}
	}
}

// autoPopulateServer creates a built-in MCP server if conditions are met
func (a *Adapter) autoPopulateServer(ctx context.Context, builtin builtinMCPServer) error {
	// Check if required env var is set (if specified)
	if builtin.RequireEnv != "" && os.Getenv(builtin.RequireEnv) == "" {
		return nil // Required env var not set, skip
	}

	// Check if the server was previously deleted (soft delete)
	wasDeleted, err := a.mcpRepo.WasDeleted(ctx, builtin.Name)
	if err != nil {
		return fmt.Errorf("failed to check if %s server was deleted: %w", builtin.Name, err)
	}
	if wasDeleted {
		log.Printf("%s MCP server was previously deleted, not auto-populating", builtin.Name)
		return nil
	}

	// Check if the server already exists
	existing, err := a.mcpRepo.GetByName(ctx, builtin.Name)
	if err == nil && existing != nil {
		return nil // Already exists
	}

	// Build environment variables
	var env []string
	for _, envVar := range builtin.EnvVars {
		if value := os.Getenv(envVar); value != "" {
			env = append(env, envVar+"="+value)
		}
	}

	// Create the server
	log.Printf("Auto-populating %s MCP server", builtin.Name)

	server := &models.MCPServer{
		ID:             a.idGen.GenerateMCPServerID(),
		Name:           builtin.Name,
		TransportType:  "stdio",
		Command:        builtin.Command,
		Args:           []string{},
		Env:            env,
		AutoReconnect:  true,
		ReconnectDelay: 5,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := a.mcpRepo.Create(ctx, server); err != nil {
		return fmt.Errorf("failed to create %s MCP server: %w", builtin.Name, err)
	}

	log.Printf("Successfully created %s MCP server", builtin.Name)
	return nil
}

// AddServer adds a new MCP server, persists it, and registers its tools
func (a *Adapter) AddServer(ctx context.Context, cfg config.MCPServerConfig) error {
	// First add to manager
	if err := a.addServerToManager(ctx, cfg); err != nil {
		return err
	}

	// Persist to database
	server := &models.MCPServer{
		ID:             a.idGen.GenerateMCPServerID(),
		Name:           cfg.Name,
		TransportType:  cfg.Transport,
		Command:        cfg.Command,
		Args:           cfg.Args,
		Env:            cfg.Env,
		URL:            cfg.URL,
		APIKey:         cfg.APIKey,
		AutoReconnect:  cfg.AutoReconnect,
		ReconnectDelay: cfg.ReconnectDelay,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := a.mcpRepo.Create(ctx, server); err != nil {
		// If persistence fails, still keep the server running but log error
		log.Printf("Warning: Failed to persist MCP server %s to database: %v", cfg.Name, err)
	}

	return nil
}

// addServerToManager adds server to manager without persistence (used for initialization)
func (a *Adapter) addServerToManager(ctx context.Context, cfg config.MCPServerConfig) error {
	serverConfig := &ServerConfig{
		Name:           cfg.Name,
		Transport:      cfg.Transport,
		Command:        cfg.Command,
		Args:           cfg.Args,
		Env:            cfg.Env,
		URL:            cfg.URL,
		APIKey:         cfg.APIKey,
		AutoReconnect:  cfg.AutoReconnect,
		ReconnectDelay: time.Duration(cfg.ReconnectDelay) * time.Second,
	}

	if err := a.manager.AddServer(serverConfig); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	client, err := a.manager.GetClient(cfg.Name)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if err := a.discoverAndRegisterTools(ctx, cfg.Name, client); err != nil {
		log.Printf("Warning: Failed to discover tools for server %s: %v", cfg.Name, err)
	}

	return nil
}

// RemoveServer removes an MCP server and its tools
func (a *Adapter) RemoveServer(ctx context.Context, name string) error {
	// Disable/delete tools registered by this server
	if toolNames, exists := a.serverTools[name]; exists {
		for _, toolName := range toolNames {
			tool, err := a.toolService.GetByName(ctx, toolName)
			if err == nil {
				_, _ = a.toolService.Disable(ctx, tool.ID)
			}
		}
		delete(a.serverTools, name)
	}

	// Remove from database
	server, err := a.mcpRepo.GetByName(ctx, name)
	if err == nil && server != nil {
		if err := a.mcpRepo.Delete(ctx, server.ID); err != nil {
			log.Printf("Warning: Failed to delete MCP server %s from database: %v", name, err)
		}
	}

	return a.manager.RemoveServer(name)
}

// discoverAndRegisterTools discovers tools from an MCP server and registers them
func (a *Adapter) discoverAndRegisterTools(ctx context.Context, serverName string, client *Client) error {
	// List tools from the MCP server
	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	log.Printf("Discovered %d tools from MCP server %s", len(tools), serverName)

	// Track registered tools
	var registeredTools []string

	// Register each tool
	for _, mcpTool := range tools {
		// Create a unique tool name by prefixing with server name
		toolName := fmt.Sprintf("mcp_%s_%s", serverName, mcpTool.Name)

		// Create description
		description := mcpTool.Description
		if description == "" {
			description = fmt.Sprintf("Tool from MCP server %s", serverName)
		}
		description = fmt.Sprintf("[MCP:%s] %s", serverName, description)

		// Register the tool
		_, err := a.toolService.RegisterTool(ctx, toolName, description, mcpTool.InputSchema)
		if err != nil {
			// Tool might already exist, skip
			log.Printf("Warning: Failed to register tool %s: %v", toolName, err)
			continue
		}

		// Register the executor
		executor := a.createExecutor(serverName, mcpTool.Name)
		if err := a.toolService.RegisterExecutor(toolName, executor); err != nil {
			log.Printf("Warning: Failed to register executor for tool %s: %v", toolName, err)
			continue
		}

		registeredTools = append(registeredTools, toolName)
		log.Printf("Registered MCP tool: %s", toolName)
	}

	// Track which tools belong to this server
	a.serverTools[serverName] = registeredTools

	return nil
}

// createExecutor creates a tool executor that proxies to the MCP server
func (a *Adapter) createExecutor(serverName, toolName string) func(ctx context.Context, arguments map[string]any) (any, error) {
	return func(ctx context.Context, arguments map[string]any) (any, error) {
		// Get the client for this server
		client, err := a.manager.GetClient(serverName)
		if err != nil {
			return nil, fmt.Errorf("MCP server %s not available: %w", serverName, err)
		}

		// Call the tool on the MCP server
		result, err := client.CallTool(ctx, toolName, arguments)
		if err != nil {
			return nil, fmt.Errorf("MCP tool call failed: %w", err)
		}

		// Check for errors in the result
		if result.IsError {
			return nil, fmt.Errorf("MCP tool returned error: %s", formatContent(result.Content))
		}

		// Format the result
		return formatToolResult(result), nil
	}
}

// formatContent formats MCP content items into a string
func formatContent(content []ContentItem) string {
	var parts []string
	for _, item := range content {
		switch item.Type {
		case "text":
			parts = append(parts, item.Text)
		case "image":
			parts = append(parts, fmt.Sprintf("[Image: %s]", item.MimeType))
		case "resource":
			parts = append(parts, fmt.Sprintf("[Resource: %s]", item.Text))
		default:
			parts = append(parts, fmt.Sprintf("[%s]", item.Type))
		}
	}
	return strings.Join(parts, "\n")
}

// formatToolResult formats an MCP tool result for Alicia
func formatToolResult(result *ToolsCallResult) any {
	// If there's only one text content item, return it directly
	if len(result.Content) == 1 && result.Content[0].Type == "text" {
		return result.Content[0].Text
	}

	// Otherwise, return structured data
	contentItems := make([]map[string]any, 0, len(result.Content))
	for _, item := range result.Content {
		contentItem := map[string]any{
			"type": item.Type,
		}
		if item.Text != "" {
			contentItem["text"] = item.Text
		}
		if item.Data != "" {
			contentItem["data"] = item.Data
		}
		if item.MimeType != "" {
			contentItem["mimeType"] = item.MimeType
		}
		contentItems = append(contentItems, contentItem)
	}

	return map[string]any{
		"content": contentItems,
	}
}

// RefreshTools rediscovers and re-registers tools from a specific server
func (a *Adapter) RefreshTools(ctx context.Context, serverName string) error {
	client, err := a.manager.GetClient(serverName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	// Remove old tools
	if toolNames, exists := a.serverTools[serverName]; exists {
		for _, toolName := range toolNames {
			tool, err := a.toolService.GetByName(ctx, toolName)
			if err == nil {
				_ = a.toolService.Delete(ctx, tool.ID)
			}
		}
		delete(a.serverTools, serverName)
	}

	// Re-discover and register tools
	return a.discoverAndRegisterTools(ctx, serverName, client)
}

// ListServers returns a list of all server names
func (a *Adapter) ListServers() []string {
	return a.manager.ListServers()
}

// GetServerStatus returns the connection status of all servers
func (a *Adapter) GetServerStatus() map[string]bool {
	servers := a.manager.ListServers()
	status := make(map[string]bool)
	for _, name := range servers {
		connected, _ := a.manager.GetServerStatus(name)
		status[name] = connected
	}
	return status
}

// GetServerTools returns the tools registered for a specific server
func (a *Adapter) GetServerTools(serverName string) []string {
	return a.serverTools[serverName]
}

// GetClient returns the client for a specific server
func (a *Adapter) GetClient(serverName string) (*Client, error) {
	return a.manager.GetClient(serverName)
}

// Close closes all MCP server connections
func (a *Adapter) Close() error {
	return a.manager.Close()
}

// ExportToolsAsJSON exports all MCP tools as JSON for debugging
func (a *Adapter) ExportToolsAsJSON(ctx context.Context) (string, error) {
	result := make(map[string]any)

	for serverName, toolNames := range a.serverTools {
		tools := make([]map[string]any, 0, len(toolNames))
		for _, toolName := range toolNames {
			tool, err := a.toolService.GetByName(ctx, toolName)
			if err != nil {
				continue
			}
			tools = append(tools, map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"schema":      tool.Schema,
				"enabled":     tool.Enabled,
			})
		}
		result[serverName] = tools
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
