package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/vmihailenco/msgpack/v5"
)

func TestNewWebSocketSyncHandler(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	if handler == nil {
		t.Fatal("expected handler to be created")
	}
	if handler.conversationRepo == nil {
		t.Error("expected conversationRepo to be set")
	}
	if handler.messageRepo == nil {
		t.Error("expected messageRepo to be set")
	}
	if handler.idGen == nil {
		t.Error("expected idGen to be set")
	}
	if handler.broadcaster == nil {
		t.Error("expected broadcaster to be set")
	}
}

func TestWebSocketSyncHandler_Handle_NoUserID(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	req := httptest.NewRequest("GET", "/api/v1/conversations/conv_123/ws", nil)
	rr := httptest.NewRecorder()

	// No user context added
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "conv_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Handle(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestWebSocketSyncHandler_Handle_ConversationNotFound(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	req := httptest.NewRequest("GET", "/api/v1/conversations/nonexistent/ws", nil)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Handle(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestWebSocketSyncHandler_Handle_InactiveConversation(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	// Create an archived conversation
	conv := models.NewConversation("conv_123", "test-user", "Test Conversation")
	conv.Archive()
	conversationRepo.conversations["conv_123"] = conv

	req := httptest.NewRequest("GET", "/api/v1/conversations/conv_123/ws", nil)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "conv_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Handle(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestWebSocketSyncHandler_ProcessMessage_Success(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	// Create test conversation
	conv := models.NewConversation("conv_123", "test-user", "Test Conversation")
	conversationRepo.conversations["conv_123"] = conv

	ctx := context.Background()
	msgReq := dto.SyncMessageRequest{
		LocalID:        "local_123",
		SequenceNumber: 1,
		Role:           string(models.MessageRoleUser),
		Contents:       "Hello, world!",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if syncedMsg.Status != "synced" {
		t.Errorf("expected status 'synced', got '%s'", syncedMsg.Status)
	}
	if syncedMsg.LocalID != "local_123" {
		t.Errorf("expected local ID 'local_123', got '%s'", syncedMsg.LocalID)
	}
	if syncedMsg.ServerID != "am_test123" {
		t.Errorf("expected server ID 'am_test123', got '%s'", syncedMsg.ServerID)
	}
}

func TestWebSocketSyncHandler_ProcessMessage_MissingLocalID(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	ctx := context.Background()
	msgReq := dto.SyncMessageRequest{
		LocalID:        "", // Missing
		SequenceNumber: 1,
		Role:           string(models.MessageRoleUser),
		Contents:       "Hello, world!",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if syncedMsg.Status != "conflict" {
		t.Errorf("expected status 'conflict', got '%s'", syncedMsg.Status)
	}
	if syncedMsg.Conflict == nil {
		t.Fatal("expected conflict details")
	}
	if syncedMsg.Conflict.Reason != "Local ID is required" {
		t.Errorf("unexpected conflict reason: %s", syncedMsg.Conflict.Reason)
	}
}

func TestWebSocketSyncHandler_ProcessMessage_MissingRole(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	ctx := context.Background()
	msgReq := dto.SyncMessageRequest{
		LocalID:        "local_123",
		SequenceNumber: 1,
		Role:           "", // Missing
		Contents:       "Hello, world!",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if syncedMsg.Status != "conflict" {
		t.Errorf("expected status 'conflict', got '%s'", syncedMsg.Status)
	}
	if syncedMsg.Conflict == nil {
		t.Fatal("expected conflict details")
	}
	if syncedMsg.Conflict.Reason != "Message role is required" {
		t.Errorf("unexpected conflict reason: %s", syncedMsg.Conflict.Reason)
	}
}

func TestWebSocketSyncHandler_ProcessMessage_DuplicateWithSameContent(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	// Create existing message
	existingMsg := models.NewLocalMessage("local_123", "conv_123", 1, models.MessageRoleUser, "Hello, world!")
	existingMsg.MarkAsSynced("am_existing")
	messageRepo.messages[existingMsg.ID] = existingMsg

	ctx := context.Background()
	msgReq := dto.SyncMessageRequest{
		LocalID:        "local_123",
		SequenceNumber: 1,
		Role:           string(models.MessageRoleUser),
		Contents:       "Hello, world!", // Same content
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if syncedMsg.Status != "synced" {
		t.Errorf("expected status 'synced', got '%s'", syncedMsg.Status)
	}
	if syncedMsg.LocalID != "local_123" {
		t.Errorf("expected local ID 'local_123', got '%s'", syncedMsg.LocalID)
	}
}

func TestWebSocketSyncHandler_ProcessMessage_DuplicateWithDifferentContent(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	// Create existing message
	existingMsg := models.NewLocalMessage("local_123", "conv_123", 1, models.MessageRoleUser, "Original content")
	existingMsg.MarkAsSynced("am_existing")
	messageRepo.messages[existingMsg.ID] = existingMsg

	ctx := context.Background()
	msgReq := dto.SyncMessageRequest{
		LocalID:        "local_123",
		SequenceNumber: 1,
		Role:           string(models.MessageRoleUser),
		Contents:       "Different content", // Different content
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if syncedMsg.Status != "conflict" {
		t.Errorf("expected status 'conflict', got '%s'", syncedMsg.Status)
	}
	if syncedMsg.Conflict == nil {
		t.Fatal("expected conflict details")
	}
	if syncedMsg.Conflict.Reason != "Content mismatch with existing message" {
		t.Errorf("unexpected conflict reason: %s", syncedMsg.Conflict.Reason)
	}
	if syncedMsg.Conflict.ServerMessage == nil {
		t.Error("expected server message in conflict")
	}
}

func TestWebSocketSyncHandler_ProcessSyncRequest(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	ctx := context.Background()
	syncReq := &dto.SyncRequest{
		Messages: []dto.SyncMessageRequest{
			{
				LocalID:        "local_1",
				SequenceNumber: 1,
				Role:           string(models.MessageRoleUser),
				Contents:       "Message 1",
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
			{
				LocalID:        "local_2",
				SequenceNumber: 2,
				Role:           string(models.MessageRoleAssistant),
				Contents:       "Message 2",
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
		},
	}

	response := handler.processSyncRequest(ctx, "conv_123", syncReq)

	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.SyncedMessages) != 2 {
		t.Errorf("expected 2 synced messages, got %d", len(response.SyncedMessages))
	}
	if response.SyncedAt == "" {
		t.Error("expected SyncedAt to be set")
	}
}

func TestWebSocketSyncHandler_ProcessSyncRequest_WithErrors(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	ctx := context.Background()
	syncReq := &dto.SyncRequest{
		Messages: []dto.SyncMessageRequest{
			{
				LocalID:        "", // Invalid
				SequenceNumber: 1,
				Role:           string(models.MessageRoleUser),
				Contents:       "Message 1",
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
			{
				LocalID:        "local_2",
				SequenceNumber: 2,
				Role:           "", // Invalid
				Contents:       "Message 2",
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
		},
	}

	response := handler.processSyncRequest(ctx, "conv_123", syncReq)

	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.SyncedMessages) != 2 {
		t.Errorf("expected 2 synced messages, got %d", len(response.SyncedMessages))
	}

	// Both should have conflicts
	for i, msg := range response.SyncedMessages {
		if msg.Status != "conflict" {
			t.Errorf("message %d: expected status 'conflict', got '%s'", i, msg.Status)
		}
	}
}

func TestWebSocketSyncHandler_MessagePackEncoding(t *testing.T) {
	// Test that sync request/response can be properly encoded/decoded with MessagePack
	syncReq := dto.SyncRequest{
		Messages: []dto.SyncMessageRequest{
			{
				LocalID:        "local_123",
				SequenceNumber: 1,
				Role:           string(models.MessageRoleUser),
				Contents:       "Test message",
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
		},
	}

	// Encode
	data, err := msgpack.Marshal(syncReq)
	if err != nil {
		t.Fatalf("failed to encode sync request: %v", err)
	}

	// Decode
	var decoded dto.SyncRequest
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to decode sync request: %v", err)
	}

	if len(decoded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(decoded.Messages))
	}
	if decoded.Messages[0].LocalID != "local_123" {
		t.Errorf("expected local ID 'local_123', got '%s'", decoded.Messages[0].LocalID)
	}
}

func TestWebSocketSyncHandler_TimestampParsing(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	ctx := context.Background()

	// Test with valid timestamps
	now := time.Now()
	msgReq := dto.SyncMessageRequest{
		LocalID:        "local_123",
		SequenceNumber: 1,
		Role:           string(models.MessageRoleUser),
		Contents:       "Hello",
		CreatedAt:      now.Format(time.RFC3339),
		UpdatedAt:      now.Add(1 * time.Second).Format(time.RFC3339),
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if syncedMsg.Status != "synced" {
		t.Errorf("expected status 'synced', got '%s'", syncedMsg.Status)
	}
}

func TestWebSocketSyncHandler_TimestampParsing_InvalidFormat(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	broadcaster := NewWebSocketBroadcaster()

	handler := NewWebSocketSyncHandler(conversationRepo, messageRepo, idGen, broadcaster)

	ctx := context.Background()

	// Test with invalid timestamp (should use current time as fallback)
	msgReq := dto.SyncMessageRequest{
		LocalID:        "local_123",
		SequenceNumber: 1,
		Role:           string(models.MessageRoleUser),
		Contents:       "Hello",
		CreatedAt:      "invalid-timestamp",
		UpdatedAt:      "also-invalid",
	}

	syncedMsg, err := handler.processMessage(ctx, "conv_123", msgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still succeed (with current time as fallback)
	if syncedMsg.Status != "synced" {
		t.Errorf("expected status 'synced', got '%s'", syncedMsg.Status)
	}
}

// Mock helper to extend mockMessageRepo for GetByLocalID
func init() {
	// Ensure GetByLocalID is properly implemented in the mock
	// (Already implemented in test_helpers_test.go)
}
