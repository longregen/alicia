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

type Adapter struct {
	manager     *Manager
	toolService ports.ToolService
	mcpRepo     ports.MCPServerRepository
	idGen       ports.IDGenerator
	serverTools map[string][]string
}

func NewAdapter(ctx context.Context, toolService ports.ToolService, mcpRepo ports.MCPServerRepository, idGen ports.IDGenerator) *Adapter {
	adapter := &Adapter{
		manager:     NewManager(ctx),
		toolService: toolService,
		mcpRepo:     mcpRepo,
		idGen:       idGen,
		serverTools: make(map[string][]string),
	}

	adapter.manager.SetConnectionCallback(adapter.onConnectionChange)

	return adapter
}

// onConnectionChange unregisters tool executors on disconnect (so they won't be offered
// to the LLM) and re-registers them on reconnect.
func (a *Adapter) onConnectionChange(serverName string, connected bool) {
	if connected {
		log.Printf("MCP server %s connected, re-registering tools", serverName)
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
		if toolNames, exists := a.serverTools[serverName]; exists {
			for _, toolName := range toolNames {
				a.toolService.UnregisterExecutor(toolName)
			}
		}
	}
}

func (a *Adapter) InitializeServers(ctx context.Context, configs []config.MCPServerConfig) error {
	a.autoPopulateBuiltinServers(ctx)

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

	for _, cfg := range configs {
		if err := a.addServerToManager(ctx, cfg); err != nil {
			log.Printf("Warning: Failed to initialize MCP server %s: %v", cfg.Name, err)
			continue
		}
	}
	return nil
}

type builtinMCPServer struct {
	Name       string
	Command    string
	EnvVars    []string
	RequireEnv string
}

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

func (a *Adapter) autoPopulateBuiltinServers(ctx context.Context) {
	for _, builtin := range builtinServers {
		if err := a.autoPopulateServer(ctx, builtin); err != nil {
			log.Printf("Warning: Failed to auto-populate %s MCP server: %v", builtin.Name, err)
		}
	}
}

func (a *Adapter) autoPopulateServer(ctx context.Context, builtin builtinMCPServer) error {
	if builtin.RequireEnv != "" && os.Getenv(builtin.RequireEnv) == "" {
		return nil
	}

	wasDeleted, err := a.mcpRepo.WasDeleted(ctx, builtin.Name)
	if err != nil {
		return fmt.Errorf("failed to check if %s server was deleted: %w", builtin.Name, err)
	}
	if wasDeleted {
		log.Printf("%s MCP server was previously deleted, not auto-populating", builtin.Name)
		return nil
	}

	existing, err := a.mcpRepo.GetByName(ctx, builtin.Name)
	if err == nil && existing != nil {
		return nil
	}

	var env []string
	for _, envVar := range builtin.EnvVars {
		if value := os.Getenv(envVar); value != "" {
			env = append(env, envVar+"="+value)
		}
	}

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

func (a *Adapter) AddServer(ctx context.Context, cfg config.MCPServerConfig) error {
	if err := a.addServerToManager(ctx, cfg); err != nil {
		return err
	}

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
		log.Printf("Warning: Failed to persist MCP server %s to database: %v", cfg.Name, err)
	}

	return nil
}

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

func (a *Adapter) RemoveServer(ctx context.Context, name string) error {
	if toolNames, exists := a.serverTools[name]; exists {
		for _, toolName := range toolNames {
			tool, err := a.toolService.GetByName(ctx, toolName)
			if err == nil {
				_, _ = a.toolService.Disable(ctx, tool.ID)
			}
		}
		delete(a.serverTools, name)
	}

	server, err := a.mcpRepo.GetByName(ctx, name)
	if err == nil && server != nil {
		if err := a.mcpRepo.Delete(ctx, server.ID); err != nil {
			log.Printf("Warning: Failed to delete MCP server %s from database: %v", name, err)
		}
	}

	return a.manager.RemoveServer(name)
}

func (a *Adapter) discoverAndRegisterTools(ctx context.Context, serverName string, client *Client) error {
	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	log.Printf("Discovered %d tools from MCP server %s", len(tools), serverName)

	var registeredTools []string

	for _, mcpTool := range tools {
		toolName := fmt.Sprintf("mcp_%s_%s", serverName, mcpTool.Name)

		description := mcpTool.Description
		if description == "" {
			description = fmt.Sprintf("Tool from MCP server %s", serverName)
		}
		description = fmt.Sprintf("[MCP:%s] %s", serverName, description)

		_, err := a.toolService.RegisterTool(ctx, toolName, description, mcpTool.InputSchema)
		if err != nil {
			log.Printf("Warning: Failed to register tool %s: %v", toolName, err)
			continue
		}

		executor := a.createExecutor(serverName, mcpTool.Name)
		if err := a.toolService.RegisterExecutor(toolName, executor); err != nil {
			log.Printf("Warning: Failed to register executor for tool %s: %v", toolName, err)
			continue
		}

		registeredTools = append(registeredTools, toolName)
		log.Printf("Registered MCP tool: %s", toolName)
	}

	a.serverTools[serverName] = registeredTools

	return nil
}

func (a *Adapter) createExecutor(serverName, toolName string) func(ctx context.Context, arguments map[string]any) (any, error) {
	return func(ctx context.Context, arguments map[string]any) (any, error) {
		client, err := a.manager.GetClient(serverName)
		if err != nil {
			return nil, fmt.Errorf("MCP server %s not available: %w", serverName, err)
		}

		result, err := client.CallTool(ctx, toolName, arguments)
		if err != nil {
			return nil, fmt.Errorf("MCP tool call failed: %w", err)
		}

		if result.IsError {
			return nil, fmt.Errorf("MCP tool returned error: %s", formatContent(result.Content))
		}

		return formatToolResult(result), nil
	}
}

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

func formatToolResult(result *ToolsCallResult) any {
	if len(result.Content) == 1 && result.Content[0].Type == "text" {
		return result.Content[0].Text
	}

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

func (a *Adapter) RefreshTools(ctx context.Context, serverName string) error {
	client, err := a.manager.GetClient(serverName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if toolNames, exists := a.serverTools[serverName]; exists {
		for _, toolName := range toolNames {
			tool, err := a.toolService.GetByName(ctx, toolName)
			if err == nil {
				_ = a.toolService.Delete(ctx, tool.ID)
			}
		}
		delete(a.serverTools, serverName)
	}

	return a.discoverAndRegisterTools(ctx, serverName, client)
}

func (a *Adapter) ListServers() []string {
	return a.manager.ListServers()
}

func (a *Adapter) GetServerStatus() map[string]bool {
	servers := a.manager.ListServers()
	status := make(map[string]bool)
	for _, name := range servers {
		connected, _ := a.manager.GetServerStatus(name)
		status[name] = connected
	}
	return status
}

func (a *Adapter) GetServerTools(serverName string) []string {
	return a.serverTools[serverName]
}

func (a *Adapter) GetClient(serverName string) (*Client, error) {
	return a.manager.GetClient(serverName)
}

func (a *Adapter) Close() error {
	return a.manager.Close()
}

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
