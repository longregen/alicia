package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/longregen/alicia/internal/adapters/http/dto"
)

// SSEBroadcaster manages Server-Sent Events connections and broadcasts messages
type SSEBroadcaster struct {
	mu          sync.RWMutex
	connections map[string]map[chan string]struct{} // conversationID -> set of channels
}

// NewSSEBroadcaster creates a new SSE broadcaster instance
func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		connections: make(map[string]map[chan string]struct{}),
	}
}

// Subscribe creates a new SSE connection for a conversation
func (b *SSEBroadcaster) Subscribe(conversationID string) chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan string, 10) // Buffer to prevent blocking

	if _, exists := b.connections[conversationID]; !exists {
		b.connections[conversationID] = make(map[chan string]struct{})
	}

	b.connections[conversationID][ch] = struct{}{}
	log.Printf("SSE: Client subscribed to conversation %s (total: %d)", conversationID, len(b.connections[conversationID]))

	return ch
}

// Unsubscribe removes an SSE connection
func (b *SSEBroadcaster) Unsubscribe(conversationID string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if connections, exists := b.connections[conversationID]; exists {
		delete(connections, ch)
		close(ch)

		// Clean up empty conversation maps
		if len(connections) == 0 {
			delete(b.connections, conversationID)
		}

		log.Printf("SSE: Client unsubscribed from conversation %s (remaining: %d)", conversationID, len(connections))
	}
}

// BroadcastMessageEvent sends a message event to all subscribers of a conversation
func (b *SSEBroadcaster) BroadcastMessageEvent(conversationID string, message *dto.MessageResponse) {
	b.mu.RLock()
	connections, exists := b.connections[conversationID]
	b.mu.RUnlock()

	if !exists || len(connections) == 0 {
		return // No subscribers
	}

	// Create event data
	eventData := map[string]interface{}{
		"type":    "message",
		"message": message,
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("SSE: Failed to marshal message event: %v", err)
		return
	}

	// Format as SSE event
	event := fmt.Sprintf("data: %s\n\n", string(jsonData))

	// Broadcast to all connections (non-blocking)
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range connections {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel buffer full, skip this client
			log.Printf("SSE: Channel buffer full for conversation %s", conversationID)
		}
	}

	log.Printf("SSE: Broadcasted message to %d clients for conversation %s", len(connections), conversationID)
}

// BroadcastSyncEvent sends a sync event to all subscribers
func (b *SSEBroadcaster) BroadcastSyncEvent(conversationID string) {
	b.mu.RLock()
	connections, exists := b.connections[conversationID]
	b.mu.RUnlock()

	if !exists || len(connections) == 0 {
		return
	}

	event := "data: {\"type\":\"sync\"}\n\n"

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range connections {
		select {
		case ch <- event:
		default:
			log.Printf("SSE: Channel buffer full for conversation %s", conversationID)
		}
	}
}

// GetConnectionCount returns the number of active connections for a conversation
func (b *SSEBroadcaster) GetConnectionCount(conversationID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if connections, exists := b.connections[conversationID]; exists {
		return len(connections)
	}
	return 0
}
