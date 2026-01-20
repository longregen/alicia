package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// TestMessagesHandler_Send_ProcessUserMessageFailure verifies that process user message
// failures are returned as HTTP errors (since agent-based processing requires this to succeed).
func TestMessagesHandler_Send_ProcessUserMessageFailure(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()

	// Mock use case that returns an error
	processUseCase := newMockProcessUserMessageUseCase()
	processUseCase.err = errors.New("database error")

	// Create WebSocket broadcaster to capture events
	wsBroadcaster := NewWebSocketBroadcaster()

	handler := NewMessagesHandler(
		conversationRepo,
		messageRepo,
		nil, // toolUseRepo
		nil, // memoryUsageRepo
		nil, // sendMessageUseCase (deprecated)
		processUseCase,
		nil, // editAssistantMessageUseCase
		nil, // editUserMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		wsBroadcaster,
		nil, // idGen
	)

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

	// The request should fail since processUserMessageUseCase failed
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}

	t.Log("Handler correctly returned error for processUserMessage failure")
}

// TestMessagesHandler_Send_WithProcessUserMessageSuccess verifies that when process user message
// SUCCEEDS, the request is accepted and forwarded to the agent.
func TestMessagesHandler_Send_WithProcessUserMessageSuccess(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()

	// Mock use case that succeeds
	processUseCase := newMockProcessUserMessageUseCase()
	userMsg := models.NewUserMessage("am_user", "ac_test123", 1, "Hi")
	processUseCase.output = &ports.ProcessUserMessageOutput{
		Message: userMsg,
	}

	wsBroadcaster := NewWebSocketBroadcaster()

	handler := NewMessagesHandler(
		conversationRepo,
		messageRepo,
		nil, // toolUseRepo
		nil, // memoryUsageRepo
		nil, // sendMessageUseCase (deprecated)
		processUseCase,
		nil, // editAssistantMessageUseCase
		nil, // editUserMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		wsBroadcaster,
		nil, // idGen
	)

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	body := `{"contents": "Hi"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	// Request should be accepted - response generation will be handled by agent
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	t.Log("Handler processed successful send message case - forwarded to agent")
}

// TestMessagesHandler_Send_WithoutProcessUserMessageUseCase verifies that when
// processUserMessageUseCase is nil, the handler returns service unavailable.
func TestMessagesHandler_Send_WithoutProcessUserMessageUseCase(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()

	wsBroadcaster := NewWebSocketBroadcaster()

	handler := NewMessagesHandler(
		conversationRepo,
		messageRepo,
		nil, // toolUseRepo
		nil, // memoryUsageRepo
		nil, // sendMessageUseCase (deprecated)
		nil, // processUserMessageUseCase - nil to test error handling
		nil, // editAssistantMessageUseCase
		nil, // editUserMessageUseCase
		nil, // regenerateResponseUseCase
		nil, // continueResponseUseCase
		wsBroadcaster,
		nil, // idGen
	)

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	body := `{"contents": "Hi"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	// Request should fail with 503 Service Unavailable
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	t.Log("Handler correctly returned 503 when processUserMessageUseCase is nil")
}
