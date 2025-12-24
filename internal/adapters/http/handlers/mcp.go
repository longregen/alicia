package handlers

import (
	"net/http"
	"strings"

	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/config"
)

// MCPServerRequest represents a request to add an MCP server
type MCPServerRequest struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	Env       []string `json:"env,omitempty"`
	URL       string   `json:"url,omitempty"`
	APIKey    string   `json:"api_key,omitempty"`
}

// MCPServerResponse represents a response with MCP server information
type MCPServerResponse struct {
	Name      string        `json:"name"`
	Transport string        `json:"transport"`
	Command   string        `json:"command,omitempty"`
	Status    string        `json:"status"` // "connected", "disconnected", "error"
	Tools     []MCPToolInfo `json:"tools"`  // list of tool names and descriptions
}

// MCPToolInfo represents basic information about an MCP tool
type MCPToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// MCPServersListResponse represents a list of MCP servers
type MCPServersListResponse struct {
	Servers []MCPServerResponse `json:"servers"`
	Total   int                 `json:"total"`
}

// MCPToolsListResponse represents all tools from all MCP servers
type MCPToolsListResponse struct {
	Tools map[string][]MCPToolInfo `json:"tools"` // server name -> tools
	Total int                      `json:"total"`
}

// MCPHandler handles MCP server configuration endpoints
type MCPHandler struct {
	mcpAdapter *mcp.Adapter
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(mcpAdapter *mcp.Adapter) *MCPHandler {
	return &MCPHandler{
		mcpAdapter: mcpAdapter,
	}
}

// ListServers handles GET /api/v1/mcp/servers
func (h *MCPHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	serverNames := h.mcpAdapter.ListServers()

	// Get status for each server
	serverStatus := h.mcpAdapter.GetServerStatus()

	// Build response
	servers := make([]MCPServerResponse, 0, len(serverNames))
	for _, name := range serverNames {
		status := "disconnected"
		if connected, exists := serverStatus[name]; exists && connected {
			status = "connected"
		}

		// Get tools for this server
		toolNames := h.mcpAdapter.GetServerTools(name)
		tools := make([]MCPToolInfo, 0, len(toolNames))

		// For each tool, we'll just use the name (description would require
		// fetching from tool service, which we'll skip for now for simplicity)
		for _, toolName := range toolNames {
			tools = append(tools, MCPToolInfo{
				Name: toolName,
			})
		}

		// Get server config info (we'll extract from the client if available)
		transport := "unknown"
		command := ""

		// Try to get client info
		client, err := h.mcpAdapter.GetClient(name)
		if err == nil && client != nil {
			serverInfo := client.GetServerInfo()
			if serverInfo != nil {
				// We don't have direct access to transport/command from the client,
				// so we'll just note it's available
				transport = "configured"
			}
		}

		servers = append(servers, MCPServerResponse{
			Name:      name,
			Transport: transport,
			Command:   command,
			Status:    status,
			Tools:     tools,
		})
	}

	response := MCPServersListResponse{
		Servers: servers,
		Total:   len(servers),
	}

	respondJSON(w, response, http.StatusOK)
}

// AddServer handles POST /api/v1/mcp/servers
func (h *MCPHandler) AddServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode request
	req, ok := decodeJSON[MCPServerRequest](r, w)
	if !ok {
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, "validation_error", "Server name is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Transport) == "" {
		respondError(w, "validation_error", "Transport is required", http.StatusBadRequest)
		return
	}

	// Validate transport type
	if req.Transport != "stdio" && req.Transport != "sse" && req.Transport != "http" {
		respondError(w, "validation_error", "Transport must be 'stdio', 'sse', or 'http'", http.StatusBadRequest)
		return
	}

	// Validate transport-specific requirements
	if req.Transport == "stdio" && strings.TrimSpace(req.Command) == "" {
		respondError(w, "validation_error", "Command is required for stdio transport", http.StatusBadRequest)
		return
	}

	if (req.Transport == "sse" || req.Transport == "http") && strings.TrimSpace(req.URL) == "" {
		respondError(w, "validation_error", "URL is required for HTTP/SSE transport", http.StatusBadRequest)
		return
	}

	// Convert request to config
	serverConfig := config.MCPServerConfig{
		Name:           req.Name,
		Transport:      req.Transport,
		Command:        req.Command,
		Args:           req.Args,
		Env:            req.Env,
		URL:            req.URL,
		APIKey:         req.APIKey,
		AutoReconnect:  true, // Enable auto-reconnect by default
		ReconnectDelay: 5,    // 5 seconds default
	}

	// Add server via adapter
	if err := h.mcpAdapter.AddServer(ctx, serverConfig); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			respondError(w, "conflict", "Server with this name already exists", http.StatusConflict)
			return
		}
		respondError(w, "internal_error", "Failed to add MCP server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the newly added server's info
	client, err := h.mcpAdapter.GetClient(req.Name)
	status := "connected"
	var tools []MCPToolInfo

	if err != nil {
		status = "error"
	} else if client != nil {
		// Get tools for this server
		toolNames := h.mcpAdapter.GetServerTools(req.Name)
		tools = make([]MCPToolInfo, 0, len(toolNames))
		for _, toolName := range toolNames {
			tools = append(tools, MCPToolInfo{
				Name: toolName,
			})
		}
	}

	response := MCPServerResponse{
		Name:      req.Name,
		Transport: req.Transport,
		Command:   req.Command,
		Status:    status,
		Tools:     tools,
	}

	respondJSON(w, response, http.StatusCreated)
}

// RemoveServer handles DELETE /api/v1/mcp/servers/{name}
func (h *MCPHandler) RemoveServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get server name from URL parameter
	name, ok := validateURLParam(r, w, "name", "Server name")
	if !ok {
		return
	}

	// Remove server via adapter
	if err := h.mcpAdapter.RemoveServer(ctx, name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, "not_found", "MCP server not found", http.StatusNotFound)
			return
		}
		respondError(w, "internal_error", "Failed to remove MCP server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}

// ListTools handles GET /api/v1/mcp/tools
func (h *MCPHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all server names
	serverNames := h.mcpAdapter.ListServers()

	// Build tools map
	toolsMap := make(map[string][]MCPToolInfo)
	totalTools := 0

	for _, serverName := range serverNames {
		// Get client to check if it's connected
		_, err := h.mcpAdapter.GetClient(serverName)
		if err != nil {
			// Server not connected, skip
			toolsMap[serverName] = []MCPToolInfo{}
			continue
		}

		// Get tools from this server
		toolNames := h.mcpAdapter.GetServerTools(serverName)
		tools := make([]MCPToolInfo, 0, len(toolNames))

		// Try to get actual tool info from the MCP server
		client, clientErr := h.mcpAdapter.GetClient(serverName)
		if clientErr == nil && client != nil {
			// List tools from MCP server to get descriptions
			mcpTools, err := client.ListTools(ctx)
			if err == nil {
				// Create a map of tool names to descriptions
				toolDescMap := make(map[string]string)
				for _, tool := range mcpTools {
					toolDescMap[tool.Name] = tool.Description
				}

				// Build tool info with descriptions
				for _, toolName := range toolNames {
					// Extract the original tool name (remove "mcp_<servername>_" prefix)
					originalName := strings.TrimPrefix(toolName, "mcp_"+serverName+"_")
					description := toolDescMap[originalName]

					tools = append(tools, MCPToolInfo{
						Name:        toolName,
						Description: description,
					})
				}
			} else {
				// Fallback: just use names
				for _, toolName := range toolNames {
					tools = append(tools, MCPToolInfo{
						Name: toolName,
					})
				}
			}
		} else {
			// Fallback: just use names
			for _, toolName := range toolNames {
				tools = append(tools, MCPToolInfo{
					Name: toolName,
				})
			}
		}

		toolsMap[serverName] = tools
		totalTools += len(tools)
	}

	response := MCPToolsListResponse{
		Tools: toolsMap,
		Total: totalTools,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetClient is a helper method to access the MCP client for a server
func (h *MCPHandler) GetClient(serverName string) (*mcp.Client, error) {
	return h.mcpAdapter.GetClient(serverName)
}

// GetServerNames is a helper to get server names from the adapter
func (h *MCPHandler) GetServerNames() []string {
	return h.mcpAdapter.ListServers()
}
