package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/pkg/protocol"
)

// Mock MCP Adapter
type mockMCPAdapter struct {
	serverStatus map[string]bool
}

func (m *mockMCPAdapter) GetServerStatus() map[string]bool {
	return m.serverStatus
}

// Tests for ServerInfoHandler.GetServerInfo

func TestServerInfoHandler_GetServerInfo_Success(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "gpt-4",
		},
	}

	handler := NewServerInfoHandler(cfg, newMockConversationRepo(), newMockMessageRepo(), (*mcp.Adapter)(nil))

	req := httptest.NewRequest("GET", "/api/v1/server/info", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.GetServerInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response protocol.ServerInfo
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Connection.Status != "connected" {
		t.Errorf("expected connection status 'connected', got %v", response.Connection.Status)
	}

	if response.Model.Name != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %v", response.Model.Name)
	}

	if len(response.MCPServers) != 0 {
		t.Errorf("expected 0 MCP servers (nil adapter), got %d", len(response.MCPServers))
	}
}

func TestServerInfoHandler_GetServerInfo_NoAuth(t *testing.T) {
	cfg := &config.Config{}
	handler := NewServerInfoHandler(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/v1/server/info", nil)

	rr := httptest.NewRecorder()
	handler.GetServerInfo(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

// Tests for ServerInfoHandler.GetSessionStats

func TestServerInfoHandler_GetSessionStats_Success(t *testing.T) {
	cfg := &config.Config{}
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()

	// Create test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	convRepo.conversations["ac_test123"] = conv

	// Create test messages
	msg1 := models.NewMessage("am_1", "ac_test123", 0, models.MessageRoleUser, "Hello")
	msg2 := models.NewMessage("am_2", "ac_test123", 1, models.MessageRoleAssistant, "Hi")
	msg2.ToolUses = []*models.ToolUse{
		{
			ID:        "at_1",
			ToolName:  "test_tool",
			Arguments: map[string]any{"param": "value"},
		},
	}

	msgRepo.byConversation["ac_test123"] = []*models.Message{msg1, msg2}

	handler := NewServerInfoHandler(cfg, convRepo, msgRepo, nil)

	req := httptest.NewRequest("GET", "/api/v1/conversations/ac_test123/stats", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetSessionStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response SessionStatsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.MessageCount != 2 {
		t.Errorf("expected 2 messages, got %d", response.MessageCount)
	}

	if response.ToolCallCount != 1 {
		t.Errorf("expected 1 tool call, got %d", response.ToolCallCount)
	}

	if response.ConversationID != "ac_test123" {
		t.Errorf("expected conversation_id 'ac_test123', got %v", response.ConversationID)
	}
}

func TestServerInfoHandler_GetSessionStats_ConversationNotFound(t *testing.T) {
	cfg := &config.Config{}
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()

	handler := NewServerInfoHandler(cfg, convRepo, msgRepo, nil)

	req := httptest.NewRequest("GET", "/api/v1/conversations/nonexistent/stats", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetSessionStats(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestServerInfoHandler_GetSessionStats_MessageRepoError(t *testing.T) {
	cfg := &config.Config{}
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	msgRepo.getErr = errors.New("database error")

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	convRepo.conversations["ac_test123"] = conv

	handler := NewServerInfoHandler(cfg, convRepo, msgRepo, nil)

	req := httptest.NewRequest("GET", "/api/v1/conversations/ac_test123/stats", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetSessionStats(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for ServerInfoHandler.GetGlobalStats

func TestServerInfoHandler_GetGlobalStats_Success(t *testing.T) {
	cfg := &config.Config{}
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()

	// Create test conversations
	conv1 := models.NewConversation("ac_1", "test-user", "Conversation 1")
	conv2 := models.NewConversation("ac_2", "test-user", "Conversation 2")
	convRepo.conversations["ac_1"] = conv1
	convRepo.conversations["ac_2"] = conv2

	// Create test messages
	msg1 := models.NewMessage("am_1", "ac_1", 0, models.MessageRoleUser, "user message")
	msg2 := models.NewMessage("am_2", "ac_2", 1, models.MessageRoleAssistant, "assistant message")
	msg2.ToolUses = []*models.ToolUse{
		{
			ID:       "at_1",
			ToolName: "test_tool",
		},
	}

	msgRepo.byConversation["ac_1"] = []*models.Message{msg1}
	msgRepo.byConversation["ac_2"] = []*models.Message{msg2}

	handler := NewServerInfoHandler(cfg, convRepo, msgRepo, nil)

	req := httptest.NewRequest("GET", "/api/v1/server/stats", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.GetGlobalStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response protocol.SessionStats
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.MessageCount != 2 {
		t.Errorf("expected 2 messages, got %d", response.MessageCount)
	}

	if response.ToolCallCount != 1 {
		t.Errorf("expected 1 tool call, got %d", response.ToolCallCount)
	}

	if response.SessionDuration <= 0 {
		t.Error("expected session duration > 0")
	}
}

func TestServerInfoHandler_GetGlobalStats_ConversationRepoError(t *testing.T) {
	cfg := &config.Config{}
	convRepo := newMockConversationRepo()
	convRepo.listErr = errors.New("database error")
	msgRepo := newMockMessageRepo()

	handler := NewServerInfoHandler(cfg, convRepo, msgRepo, nil)

	req := httptest.NewRequest("GET", "/api/v1/server/stats", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.GetGlobalStats(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestServerInfoHandler_GetGlobalStats_NoAuth(t *testing.T) {
	cfg := &config.Config{}
	handler := NewServerInfoHandler(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/v1/server/stats", nil)

	rr := httptest.NewRecorder()
	handler.GetGlobalStats(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}
