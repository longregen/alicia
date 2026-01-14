package handlers

import (
	"testing"
)

// TestWebSocketBroadcaster_BroadcastError verifies error broadcasting for WebSocket
func TestWebSocketBroadcaster_BroadcastError(t *testing.T) {
	broadcaster := NewWebSocketBroadcaster()

	// This test just verifies the method exists and doesn't panic
	// We can't test actual WebSocket connections without more complex setup
	broadcaster.BroadcastError("ac_test123", "generation_failed", "Test error")

	// If we got here without panic, the method works
	t.Log("BroadcastError method executed successfully")
}
