package handlers

import (
	"net/http"
	"strings"

	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/config"
)

type MCPServerRequest struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	Env       []string `json:"env,omitempty"`
	URL       string   `json:"url,omitempty"`
	APIKey    string   `json:"api_key,omitempty"`
}

type MCPServerResponse struct {
	Name      string        `json:"name"`
	Transport string        `json:"transport"`
	Command   string        `json:"command,omitempty"`
	Status    string        `json:"status"`
	Tools     []MCPToolInfo `json:"tools"`
}

type MCPToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type MCPServersListResponse struct {
	Servers []MCPServerResponse `json:"servers"`
	Total   int                 `json:"total"`
}

type MCPToolsListResponse struct {
	Tools map[string][]MCPToolInfo `json:"tools"`
	Total int                      `json:"total"`
}

type MCPHandler struct {
	mcpAdapter *mcp.Adapter
}

func NewMCPHandler(mcpAdapter *mcp.Adapter) *MCPHandler {
	return &MCPHandler{
		mcpAdapter: mcpAdapter,
	}
}

func (h *MCPHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	serverNames := h.mcpAdapter.ListServers()
	serverStatus := h.mcpAdapter.GetServerStatus()
	servers := make([]MCPServerResponse, 0, len(serverNames))
	for _, name := range serverNames {
		status := "disconnected"
		if connected, exists := serverStatus[name]; exists && connected {
			status = "connected"
		}

		toolNames := h.mcpAdapter.GetServerTools(name)
		tools := make([]MCPToolInfo, 0, len(toolNames))

		for _, toolName := range toolNames {
			tools = append(tools, MCPToolInfo{
				Name: toolName,
			})
		}

		transport := "unknown"
		command := ""

		client, err := h.mcpAdapter.GetClient(name)
		if err == nil && client != nil {
			serverInfo := client.GetServerInfo()
			if serverInfo != nil {
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

func (h *MCPHandler) AddServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, ok := decodeJSON[MCPServerRequest](r, w)
	if !ok {
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		respondError(w, "validation_error", "Server name is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Transport) == "" {
		respondError(w, "validation_error", "Transport is required", http.StatusBadRequest)
		return
	}

	if req.Transport != "stdio" && req.Transport != "sse" && req.Transport != "http" {
		respondError(w, "validation_error", "Transport must be 'stdio', 'sse', or 'http'", http.StatusBadRequest)
		return
	}

	if req.Transport == "stdio" && strings.TrimSpace(req.Command) == "" {
		respondError(w, "validation_error", "Command is required for stdio transport", http.StatusBadRequest)
		return
	}

	if (req.Transport == "sse" || req.Transport == "http") && strings.TrimSpace(req.URL) == "" {
		respondError(w, "validation_error", "URL is required for HTTP/SSE transport", http.StatusBadRequest)
		return
	}

	serverConfig := config.MCPServerConfig{
		Name:           req.Name,
		Transport:      req.Transport,
		Command:        req.Command,
		Args:           req.Args,
		Env:            req.Env,
		URL:            req.URL,
		APIKey:         req.APIKey,
		AutoReconnect:  true,
		ReconnectDelay: 5,
	}

	if err := h.mcpAdapter.AddServer(ctx, serverConfig); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			respondError(w, "conflict", "Server with this name already exists", http.StatusConflict)
			return
		}
		respondError(w, "internal_error", "Failed to add MCP server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client, err := h.mcpAdapter.GetClient(req.Name)
	status := "connected"
	var tools []MCPToolInfo

	if err != nil {
		status = "error"
	} else if client != nil {
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

func (h *MCPHandler) RemoveServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name, ok := validateURLParam(r, w, "name", "Server name")
	if !ok {
		return
	}

	if err := h.mcpAdapter.RemoveServer(ctx, name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, "not_found", "MCP server not found", http.StatusNotFound)
			return
		}
		respondError(w, "internal_error", "Failed to remove MCP server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MCPHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	serverNames := h.mcpAdapter.ListServers()
	toolsMap := make(map[string][]MCPToolInfo)
	totalTools := 0

	for _, serverName := range serverNames {
		_, err := h.mcpAdapter.GetClient(serverName)
		if err != nil {
			toolsMap[serverName] = []MCPToolInfo{}
			continue
		}

		toolNames := h.mcpAdapter.GetServerTools(serverName)
		tools := make([]MCPToolInfo, 0, len(toolNames))

		client, clientErr := h.mcpAdapter.GetClient(serverName)
		if clientErr == nil && client != nil {
			mcpTools, err := client.ListTools(ctx)
			if err == nil {
				toolDescMap := make(map[string]string)
				for _, tool := range mcpTools {
					toolDescMap[tool.Name] = tool.Description
				}

				for _, toolName := range toolNames {
					originalName := strings.TrimPrefix(toolName, "mcp_"+serverName+"_")
					description := toolDescMap[originalName]

					tools = append(tools, MCPToolInfo{
						Name:        toolName,
						Description: description,
					})
				}
			} else {
				for _, toolName := range toolNames {
					tools = append(tools, MCPToolInfo{
						Name: toolName,
					})
				}
			}
		} else {
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

func (h *MCPHandler) GetClient(serverName string) (*mcp.Client, error) {
	return h.mcpAdapter.GetClient(serverName)
}

func (h *MCPHandler) GetServerNames() []string {
	return h.mcpAdapter.ListServers()
}
