package handlers

import (
	"testing"
)

// TestSSEBroadcaster_BroadcastErrorEvent verifies error broadcasting functionality
func TestSSEBroadcaster_BroadcastErrorEvent(t *testing.T) {
	broadcaster := NewSSEBroadcaster()
	conversationID := "ac_test123"

	// Subscribe to the conversation
	ch := broadcaster.Subscribe(conversationID)
	defer broadcaster.Unsubscribe(conversationID, ch)

	// Broadcast an error
	go broadcaster.BroadcastErrorEvent(conversationID, "generation_failed", "LLM API rate limit exceeded")

	// Receive the event
	event := <-ch

	// Verify the event format
	if event == "" {
		t.Fatal("Expected event, got empty string")
	}

	// Basic verification that it's an error event
	if len(event) < 10 {
		t.Errorf("Event too short: %s", event)
	}

	t.Logf("Received error event: %s", event)
}

// TestWebSocketBroadcaster_BroadcastError verifies error broadcasting for WebSocket
func TestWebSocketBroadcaster_BroadcastError(t *testing.T) {
	broadcaster := NewWebSocketBroadcaster()

	// This test just verifies the method exists and doesn't panic
	// We can't test actual WebSocket connections without more complex setup
	broadcaster.BroadcastError("ac_test123", "generation_failed", "Test error")

	// If we got here without panic, the method works
	t.Log("BroadcastError method executed successfully")
}
