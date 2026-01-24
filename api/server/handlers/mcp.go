package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/services"
)

// MCPHandler handles MCP server endpoints.
type MCPHandler struct {
	mcpSvc *services.MCPService
}

// NewMCPHandler creates a new MCP handler.
func NewMCPHandler(svc *services.MCPService) *MCPHandler {
	return &MCPHandler{mcpSvc: svc}
}

// List handles GET /mcp/servers
func (h *MCPHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.mcpSvc.ListServers(r.Context())
	if err != nil {
		respondError(w, "failed to list servers", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"servers": servers,
	}, http.StatusOK)
}

// Create handles POST /mcp/servers
func (h *MCPHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string   `json:"name"`
		TransportType string   `json:"transport_type"`
		Command       string   `json:"command"`
		Args          []string `json:"args"`
		URL           string   `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.TransportType == "" {
		req.TransportType = "stdio"
	}

	server, err := h.mcpSvc.CreateServer(r.Context(), req.Name, req.TransportType, req.Command, req.Args, req.URL)
	if err != nil {
		respondError(w, "failed to create server", http.StatusInternalServerError)
		return
	}

	respondJSON(w, server, http.StatusCreated)
}

// Get handles GET /mcp/servers/{name}
func (h *MCPHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	server, err := h.mcpSvc.GetServerByName(r.Context(), name)
	if err != nil {
		respondError(w, "server not found", http.StatusNotFound)
		return
	}

	respondJSON(w, server, http.StatusOK)
}

// Update handles PUT /mcp/servers/{name}
func (h *MCPHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	server, err := h.mcpSvc.GetServerByName(r.Context(), name)
	if err != nil {
		respondError(w, "server not found", http.StatusNotFound)
		return
	}

	var req struct {
		TransportType *string  `json:"transport_type"`
		Command       *string  `json:"command"`
		Args          []string `json:"args"`
		URL           *string  `json:"url"`
		Enabled       *bool    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TransportType != nil {
		server.TransportType = *req.TransportType
	}
	if req.Command != nil {
		server.Command = *req.Command
	}
	if req.Args != nil {
		server.Args = req.Args
	}
	if req.URL != nil {
		server.URL = *req.URL
	}
	if req.Enabled != nil {
		server.Enabled = *req.Enabled
	}

	if err := h.mcpSvc.UpdateServer(r.Context(), server); err != nil {
		respondError(w, "failed to update server", http.StatusInternalServerError)
		return
	}

	respondJSON(w, server, http.StatusOK)
}

// Delete handles DELETE /mcp/servers/{name}
func (h *MCPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.mcpSvc.DeleteServerByName(r.Context(), name); err != nil {
		respondError(w, "failed to delete server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
