package handlers

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

// WebSocketBroadcaster manages WebSocket connections per conversation
type WebSocketBroadcaster struct {
	// connections maps conversation ID to a set of WebSocket connections
	connections map[string]map[*websocket.Conn]struct{}
	mu          sync.RWMutex
}

// NewWebSocketBroadcaster creates a new WebSocket broadcaster
func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return &WebSocketBroadcaster{
		connections: make(map[string]map[*websocket.Conn]struct{}),
	}
}

// Subscribe adds a WebSocket connection to a conversation's subscriber list
func (b *WebSocketBroadcaster) Subscribe(conversationID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connections[conversationID] == nil {
		b.connections[conversationID] = make(map[*websocket.Conn]struct{})
	}

	b.connections[conversationID][conn] = struct{}{}
	log.Printf("WebSocket subscribed to conversation %s (total: %d)", conversationID, len(b.connections[conversationID]))
}

// Unsubscribe removes a WebSocket connection from a conversation's subscriber list
func (b *WebSocketBroadcaster) Unsubscribe(conversationID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if conns, ok := b.connections[conversationID]; ok {
		delete(conns, conn)
		log.Printf("WebSocket unsubscribed from conversation %s (remaining: %d)", conversationID, len(conns))

		// Clean up empty conversation maps
		if len(conns) == 0 {
			delete(b.connections, conversationID)
		}
	}
}

// BroadcastBinary broadcasts binary MessagePack data to all subscribers of a conversation
func (b *WebSocketBroadcaster) BroadcastBinary(conversationID string, data []byte) {
	// Copy connections under single lock to avoid holding during I/O
	b.mu.RLock()
	conns, ok := b.connections[conversationID]
	if !ok || len(conns) == 0 {
		b.mu.RUnlock()
		return
	}
	targets := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		targets = append(targets, conn)
	}
	b.mu.RUnlock()

	// Broadcast to all connections
	for _, conn := range targets {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("Failed to broadcast to WebSocket connection: %v", err)
			// Remove failed connection
			b.Unsubscribe(conversationID, conn)
		}
	}
}

// BroadcastMessage broadcasts a message to all subscribers of a conversation
func (b *WebSocketBroadcaster) BroadcastMessage(conversationID string, msg *dto.MessageResponse) {
	// Encode message to MessagePack
	data, err := msgpack.Marshal(msg)
	if err != nil {
		log.Printf("Failed to encode message for WebSocket broadcast: %v", err)
		return
	}

	b.BroadcastBinary(conversationID, data)
}

// BroadcastError broadcasts an error message to all subscribers of a conversation
func (b *WebSocketBroadcaster) BroadcastError(conversationID string, code string, message string) {
	errorData := map[string]interface{}{
		"type": "error",
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	}

	// Encode error to MessagePack
	data, err := msgpack.Marshal(errorData)
	if err != nil {
		log.Printf("Failed to encode error for WebSocket broadcast: %v", err)
		return
	}

	b.BroadcastBinary(conversationID, data)
}

// GetSubscriberCount returns the number of active subscribers for a conversation
func (b *WebSocketBroadcaster) GetSubscriberCount(conversationID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if conns, ok := b.connections[conversationID]; ok {
		return len(conns)
	}
	return 0
}

// BroadcastConversationUpdate broadcasts a conversation metadata update to all subscribers
func (b *WebSocketBroadcaster) BroadcastConversationUpdate(conversation *models.Conversation) {
	if conversation == nil {
		return
	}

	update := protocol.ConversationUpdate{
		ConversationID: conversation.ID,
		Title:          conversation.Title,
		Status:         string(conversation.Status),
		UpdatedAt:      conversation.UpdatedAt.Format(time.RFC3339),
	}

	// Send flat structure (consistent with BroadcastMessage)
	// Frontend wrapInEnvelope detects type by field presence
	data, err := msgpack.Marshal(update)
	if err != nil {
		log.Printf("Failed to encode conversation update for WebSocket broadcast: %v", err)
		return
	}

	b.BroadcastBinary(conversation.ID, data)
}
