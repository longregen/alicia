package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
)

// TestMessagesHandler_Send_Success_AcceptsRequest tests that the handler
// accepts requests and processes them asynchronously.
func TestMessagesHandler_Send_Success_AcceptsRequest(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sendMessageUseCase := newMockSendMessageUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, nil, nil, sendMessageUseCase, newMockProcessUserMessageUseCase(), nil, nil, nil, nil, nil, nil)

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

	// Verify the handler returns 202 Accepted
	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rr.Code)
	}

	// Verify the response contains the expected fields
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}

	if response["conversation_id"] != "ac_test123" {
		t.Errorf("Expected conversation_id 'ac_test123', got %v", response["conversation_id"])
	}
}

// TestMessagesHandler_Send_WithExistingTip tests that the handler accepts
// requests for conversations that already have a tip message.
func TestMessagesHandler_Send_WithExistingTip(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	sendMessageUseCase := newMockSendMessageUseCase()
	handler := NewMessagesHandler(conversationRepo, messageRepo, nil, nil, sendMessageUseCase, newMockProcessUserMessageUseCase(), nil, nil, nil, nil, nil, nil)

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

	// Verify the handler returns 202 Accepted
	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rr.Code)
	}

	// Verify the response
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}
}
