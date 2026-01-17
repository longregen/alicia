package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/domain/models"
)

// mockToolService is a minimal implementation for testing
type mockToolService struct{}

func (m *mockToolService) RegisterTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) EnsureTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) RegisterExecutor(name string, executor func(context.Context, map[string]any) (any, error)) error {
	return nil
}

func (m *mockToolService) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) Update(ctx context.Context, tool *models.Tool) error {
	return nil
}

func (m *mockToolService) Enable(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) Disable(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) ListAll(ctx context.Context) ([]*models.Tool, error) {
	return nil, nil
}

func (m *mockToolService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockToolService) ExecuteTool(ctx context.Context, name string, arguments map[string]any) (any, error) {
	return nil, nil
}

func (m *mockToolService) CreateToolUse(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error) {
	return nil, nil
}

func (m *mockToolService) ExecuteToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	return nil, nil
}

func (m *mockToolService) GetToolUseByID(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	return nil, nil
}

func (m *mockToolService) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	return nil, nil
}

func (m *mockToolService) GetPendingToolUses(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	return nil, nil
}

func (m *mockToolService) CancelToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	return nil, nil
}

// mockMCPServerRepository is a minimal implementation for testing
type mockMCPServerRepository struct{}

func (m *mockMCPServerRepository) Create(ctx context.Context, server *models.MCPServer) error {
	return nil
}

func (m *mockMCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServer, error) {
	return nil, nil
}

func (m *mockMCPServerRepository) GetByName(ctx context.Context, name string) (*models.MCPServer, error) {
	return nil, nil
}

func (m *mockMCPServerRepository) Update(ctx context.Context, server *models.MCPServer) error {
	return nil
}

func (m *mockMCPServerRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMCPServerRepository) List(ctx context.Context) ([]*models.MCPServer, error) {
	return nil, nil
}

func (m *mockMCPServerRepository) WasDeleted(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func TestMCPHandler_ListServers(t *testing.T) {
	ctx := context.Background()
	toolService := &mockToolService{}
	mcpRepo := &mockMCPServerRepository{}
	idGen := &mockIDGenerator{}
	adapter := mcp.NewAdapter(ctx, toolService, mcpRepo, idGen)
	handler := NewMCPHandler(adapter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/servers", nil)
	w := httptest.NewRecorder()

	handler.ListServers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response MCPServersListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("Expected 0 servers, got %d", response.Total)
	}
}

func TestMCPHandler_AddServer_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	toolService := &mockToolService{}
	mcpRepo := &mockMCPServerRepository{}
	idGen := &mockIDGenerator{}
	adapter := mcp.NewAdapter(ctx, toolService, mcpRepo, idGen)
	handler := NewMCPHandler(adapter)

	tests := []struct {
		name           string
		request        MCPServerRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing name",
			request: MCPServerRequest{
				Transport: "stdio",
				Command:   "test",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Server name is required",
		},
		{
			name: "missing transport",
			request: MCPServerRequest{
				Name:    "test",
				Command: "test",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Transport is required",
		},
		{
			name: "invalid transport",
			request: MCPServerRequest{
				Name:      "test",
				Transport: "invalid",
				Command:   "test",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Transport must be 'stdio', 'sse', or 'http'",
		},
		{
			name: "stdio missing command",
			request: MCPServerRequest{
				Name:      "test",
				Transport: "stdio",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Command is required for stdio transport",
		},
		{
			name: "http missing url",
			request: MCPServerRequest{
				Name:      "test",
				Transport: "http",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "URL is required for HTTP/SSE transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/servers", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.AddServer(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check error message contains expected text
			if tt.expectedError != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedError)) {
				t.Errorf("Expected error message to contain %q, got %q", tt.expectedError, w.Body.String())
			}
		})
	}
}

func TestMCPHandler_RemoveServer(t *testing.T) {
	ctx := context.Background()
	toolService := &mockToolService{}
	mcpRepo := &mockMCPServerRepository{}
	idGen := &mockIDGenerator{}
	adapter := mcp.NewAdapter(ctx, toolService, mcpRepo, idGen)
	handler := NewMCPHandler(adapter)

	// Test removing non-existent server
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mcp/servers/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.RemoveServer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent server, got %d", http.StatusNotFound, w.Code)
	}
}

func TestMCPHandler_ListTools(t *testing.T) {
	ctx := context.Background()
	toolService := &mockToolService{}
	mcpRepo := &mockMCPServerRepository{}
	idGen := &mockIDGenerator{}
	adapter := mcp.NewAdapter(ctx, toolService, mcpRepo, idGen)
	handler := NewMCPHandler(adapter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/tools", nil)
	w := httptest.NewRecorder()

	handler.ListTools(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response MCPToolsListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("Expected 0 tools initially, got %d", response.Total)
	}
}
