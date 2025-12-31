package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// TestMessagesHandler_Send_GenerateResponseFailure verifies that AI response
// generation failures are broadcast to SSE/WebSocket subscribers as error events.
func TestMessagesHandler_Send_GenerateResponseFailure(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	// Mock use case that returns an error
	generateUseCase := newMockGenerateResponseUseCase()
	generateUseCase.err = errors.New("LLM API rate limit exceeded")

	// Create broadcasters to capture events
	sseBroadcaster := NewSSEBroadcaster()
	wsBroadcaster := NewWebSocketBroadcaster()

	handler := NewMessagesHandler(
		conversationRepo,
		messageRepo,
		idGen,
		generateUseCase,
		sseBroadcaster,
		wsBroadcaster,
	)

	// Create a test conversation
	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	// Subscribe to SSE events for this conversation
	sseChan := sseBroadcaster.Subscribe("ac_test123")
	defer sseBroadcaster.Unsubscribe("ac_test123", sseChan)

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

	// The user message should be created successfully
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["contents"] != "Hello, Alicia!" {
		t.Errorf("expected contents 'Hello, Alicia!', got %v", response["contents"])
	}

	// Now we need to wait for the goroutine to complete and check if an error
	// was broadcast to SSE subscribers
	receivedEvents := []string{}
	errorEventReceived := false

	// Wait for events with timeout
	timeout := time.After(2 * time.Second)

	// We expect to receive:
	// 1. The user message broadcast (from line 300)
	// 2. An ERROR event when generation fails (MISSING - this is the bug)
	expectedEventCount := 2

eventLoop:
	for len(receivedEvents) < expectedEventCount {
		select {
		case event := <-sseChan:
			receivedEvents = append(receivedEvents, event)

			// Parse the event to check if it's an error event
			// SSE format is "data: {json}\n\n"
			if strings.HasPrefix(event, "data: ") {
				jsonStr := strings.TrimPrefix(event, "data: ")
				jsonStr = strings.TrimSuffix(jsonStr, "\n\n")

				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &eventData); err == nil {
					eventType, ok := eventData["type"].(string)
					if ok && eventType == "error" {
						errorEventReceived = true

						// Verify error details
						if errorMsg, ok := eventData["error"].(map[string]interface{}); ok {
							if code, ok := errorMsg["code"].(string); ok && code == "generation_failed" {
								t.Logf("Received expected error event: %v", errorMsg)
							}
						}
					}
				}
			}

		case <-timeout:
			t.Logf("Timeout waiting for events. Received %d events, expected %d",
				len(receivedEvents), expectedEventCount)
			break eventLoop
		}
	}

	// Log what we received for debugging
	t.Logf("Received %d SSE events:", len(receivedEvents))
	for i, event := range receivedEvents {
		t.Logf("  Event %d: %s", i+1, event)
	}

	// Verify that error was broadcast to subscribers
	if !errorEventReceived {
		t.Errorf("No error event was broadcast to SSE subscribers")
		t.Errorf("Expected error event with type='error' and code='generation_failed'")
	}
}

// TestMessagesHandler_Send_GenerateResponseNilOutput verifies that nil output
// cases are broadcast as error events to SSE/WebSocket subscribers.
func TestMessagesHandler_Send_GenerateResponseNilOutput(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	// Mock use case that returns nil output (unexpected behavior)
	generateUseCase := newMockGenerateResponseUseCase()
	generateUseCase.output = nil
	generateUseCase.err = nil

	sseBroadcaster := NewSSEBroadcaster()
	wsBroadcaster := NewWebSocketBroadcaster()

	handler := NewMessagesHandler(
		conversationRepo,
		messageRepo,
		idGen,
		generateUseCase,
		sseBroadcaster,
		wsBroadcaster,
	)

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	sseChan := sseBroadcaster.Subscribe("ac_test123")
	defer sseBroadcaster.Unsubscribe("ac_test123", sseChan)

	body := `{"contents": "What's the weather?"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	errorEventReceived := false
	timeout := time.After(2 * time.Second)

eventLoop:
	for {
		select {
		case event := <-sseChan:
			if strings.HasPrefix(event, "data: ") {
				jsonStr := strings.TrimPrefix(event, "data: ")
				jsonStr = strings.TrimSuffix(jsonStr, "\n\n")

				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &eventData); err == nil {
					if eventType, ok := eventData["type"].(string); ok && eventType == "error" {
						errorEventReceived = true
						break eventLoop
					}
				}
			}

		case <-timeout:
			break eventLoop
		}
	}

	if !errorEventReceived {
		t.Errorf("No error event was broadcast for nil output case")
		t.Errorf("Expected error event with type='error' and code='generation_failed'")
	}
}

// TestMessagesHandler_Send_WithWorkingBroadcasters verifies that when generation
// SUCCEEDS, the response IS properly broadcast to subscribers. This confirms that
// the broadcasting infrastructure works - it's only the error case that's broken.
func TestMessagesHandler_Send_WithWorkingBroadcasters(t *testing.T) {
	conversationRepo := newMockConversationRepo()
	messageRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	// Mock use case that succeeds
	generateUseCase := newMockGenerateResponseUseCase()
	assistantMsg := models.NewAssistantMessage("am_response", "ac_test123", 2, "Hello! How can I help?")
	generateUseCase.output = &ports.GenerateResponseOutput{
		Message: assistantMsg,
	}

	sseBroadcaster := NewSSEBroadcaster()

	handler := NewMessagesHandler(
		conversationRepo,
		messageRepo,
		idGen,
		generateUseCase,
		sseBroadcaster,
		nil,
	)

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conversationRepo.conversations["ac_test123"] = conv

	sseChan := sseBroadcaster.Subscribe("ac_test123")
	defer sseBroadcaster.Unsubscribe("ac_test123", sseChan)

	body := `{"contents": "Hi"}`
	req := httptest.NewRequest("POST", "/api/v1/conversations/ac_test123/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "ac_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addUserContext(req, "test-user")

	handler.Send(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	// Should receive 2 message events: user message + assistant response
	receivedMessages := 0
	timeout := time.After(2 * time.Second)

eventLoop:
	for receivedMessages < 2 {
		select {
		case event := <-sseChan:
			if strings.HasPrefix(event, "data: ") {
				jsonStr := strings.TrimPrefix(event, "data: ")
				jsonStr = strings.TrimSuffix(jsonStr, "\n\n")

				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &eventData); err == nil {
					if eventType, ok := eventData["type"].(string); ok && eventType == "message" {
						receivedMessages++
						t.Logf("Received message event: %v", eventData)
					}
				}
			}

		case <-timeout:
			break eventLoop
		}
	}

	if receivedMessages != 2 {
		t.Errorf("Expected 2 message events (user + assistant), got %d", receivedMessages)
		t.Errorf("This proves broadcasting works for SUCCESS but not for ERRORS")
	}
}
