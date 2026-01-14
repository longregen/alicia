package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
)

// mockConversationRepoWithUpdateTipError is a mock that returns an error on UpdateTip
type mockConversationRepoWithUpdateTipError struct {
	*mockConversationRepo
	updateTipErr error
}

func newMockConversationRepoWithUpdateTipError(updateTipErr error) *mockConversationRepoWithUpdateTipError {
	return &mockConversationRepoWithUpdateTipError{
		mockConversationRepo: newMockConversationRepo(),
		updateTipErr:         updateTipErr,
	}
}

func (m *mockConversationRepoWithUpdateTipError) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	return m.updateTipErr
}

// TestMessagesHandler_Send_UpdateTipFailure verifies that UpdateTip failures
// are properly handled by returning a 500 error to prevent conversation
// branching state corruption.
func TestMessagesHandler_Send_UpdateTipFailure(t *testing.T) {
	// Create a mock conversation repo that will fail on UpdateTip
	updateTipError := errors.New("database error: failed to update tip")
	conversationRepo := newMockConversationRepoWithUpdateTipError(updateTipError)
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil)

	// Create a test conversation with an existing tip
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	existingTipID := "am_existing_tip"
	conv.TipMessageID = &existingTipID
	conversationRepo.conversations["ac_test123"] = conv

	// Create the existing tip message
	existingTip := models.NewUserMessage(existingTipID, "ac_test123", 1, "Existing message")
	messageRepo.messages[existingTipID] = existingTip

	// Send a new message
	body := `{"contents": "New message"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	// Verify the handler returns 500 Internal Server Error
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when UpdateTip fails, got %d", rr.Code)
	}

	// Verify the response contains a properly formatted error
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "internal_error" {
		t.Errorf("Expected error 'internal_error', got %v", response["error"])
	}

	if response["message"] != "Failed to update conversation tip" {
		t.Errorf("Expected message 'Failed to update conversation tip', got %v", response["message"])
	}
}

// TestMessagesHandler_Send_UpdateTipFailure_WithNilTip tests the same scenario
// but with a conversation that has no existing tip (nil TipMessageID).
func TestMessagesHandler_Send_UpdateTipFailure_WithNilTip(t *testing.T) {
	// Create a mock conversation repo that will fail on UpdateTip
	updateTipError := errors.New("database error: failed to update tip")
	conversationRepo := newMockConversationRepoWithUpdateTipError(updateTipError)
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil)

	// Create a test conversation with no existing tip (nil TipMessageID)
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	// conv.TipMessageID is nil by default
	conversationRepo.conversations["ac_test123"] = conv

	// Send the first message
	body := `{"contents": "First message"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	// Verify the handler returns 500 Internal Server Error
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when UpdateTip fails, got %d", rr.Code)
	}

	// Verify the response contains a properly formatted error
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "internal_error" {
		t.Errorf("Expected error 'internal_error', got %v", response["error"])
	}

	if response["message"] != "Failed to update conversation tip" {
		t.Errorf("Expected message 'Failed to update conversation tip', got %v", response["message"])
	}
}

// TestMessagesHandler_Send_UpdateTipSuccess_BaseCase is a control test that
// verifies the happy path where UpdateTip succeeds.
func TestMessagesHandler_Send_UpdateTipSuccess_BaseCase(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()
	generateUseCase := newMockGenerateResponseUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, idGen, generateUseCase, nil)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	// Send a message
	body := `{"contents": "Hello, Alicia!"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Setup chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	// This should succeed
	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] != "am_test123" {
		t.Errorf("expected id 'am_test123', got %v", response["id"])
	}
}
