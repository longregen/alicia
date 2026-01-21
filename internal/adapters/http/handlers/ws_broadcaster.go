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

type WebSocketBroadcaster struct {
	connections map[string]map[*websocket.Conn]struct{}
	mu          sync.RWMutex
	agentConn   *websocket.Conn
	agentMu     sync.RWMutex
}

func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return &WebSocketBroadcaster{
		connections: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (b *WebSocketBroadcaster) Subscribe(conversationID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connections[conversationID] == nil {
		b.connections[conversationID] = make(map[*websocket.Conn]struct{})
	}

	b.connections[conversationID][conn] = struct{}{}
	log.Printf("WebSocket subscribed to conversation %s (total: %d)", conversationID, len(b.connections[conversationID]))
}

func (b *WebSocketBroadcaster) Unsubscribe(conversationID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if conns, ok := b.connections[conversationID]; ok {
		delete(conns, conn)
		log.Printf("WebSocket unsubscribed from conversation %s (remaining: %d)", conversationID, len(conns))

		if len(conns) == 0 {
			delete(b.connections, conversationID)
		}
	}
}

func (b *WebSocketBroadcaster) BroadcastBinary(conversationID string, data []byte) {
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

	for _, conn := range targets {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("Failed to broadcast to WebSocket connection: %v", err)
			b.Unsubscribe(conversationID, conn)
		}
	}
}

func (b *WebSocketBroadcaster) BroadcastMessage(conversationID string, msg *dto.MessageResponse) {
	data, err := msgpack.Marshal(msg)
	if err != nil {
		log.Printf("Failed to encode message for WebSocket broadcast: %v", err)
		return
	}

	b.BroadcastBinary(conversationID, data)
}

func (b *WebSocketBroadcaster) BroadcastError(conversationID string, code string, message string) {
	errorData := map[string]interface{}{
		"type": "error",
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	}

	data, err := msgpack.Marshal(errorData)
	if err != nil {
		log.Printf("Failed to encode error for WebSocket broadcast: %v", err)
		return
	}

	b.BroadcastBinary(conversationID, data)
}

func (b *WebSocketBroadcaster) GetSubscriberCount(conversationID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if conns, ok := b.connections[conversationID]; ok {
		return len(conns)
	}
	return 0
}

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

	data, err := msgpack.Marshal(update)
	if err != nil {
		log.Printf("Failed to encode conversation update for WebSocket broadcast: %v", err)
		return
	}

	b.BroadcastBinary(conversation.ID, data)
}

func (b *WebSocketBroadcaster) SubscribeAgent(conn *websocket.Conn) {
	b.agentMu.Lock()
	defer b.agentMu.Unlock()
	b.agentConn = conn
	log.Printf("Agent WebSocket connection registered")
}

func (b *WebSocketBroadcaster) UnsubscribeAgent(conn *websocket.Conn) {
	b.agentMu.Lock()
	defer b.agentMu.Unlock()
	if b.agentConn == conn {
		b.agentConn = nil
		log.Printf("Agent WebSocket connection unregistered")
	}
}

func (b *WebSocketBroadcaster) IsAgentConnected() bool {
	b.agentMu.RLock()
	defer b.agentMu.RUnlock()
	return b.agentConn != nil
}

func (b *WebSocketBroadcaster) BroadcastResponseGenerationRequest(conversationID string, req *protocol.ResponseGenerationRequest) {
	b.agentMu.RLock()
	agentConn := b.agentConn
	b.agentMu.RUnlock()

	if agentConn == nil {
		log.Printf("No agent connected, cannot send ResponseGenerationRequest for conversation %s", conversationID)
		return
	}

	envelope := protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeResponseGenerationRequest,
		Body:           req,
	}

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		log.Printf("Failed to encode ResponseGenerationRequest: %v", err)
		return
	}

	agentConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := agentConn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("Failed to send ResponseGenerationRequest to agent: %v", err)
		b.UnsubscribeAgent(agentConn)
		return
	}

	log.Printf("Sent ResponseGenerationRequest to agent for conversation %s (type: %s, messageID: %s)",
		conversationID, req.RequestType, req.MessageID)
}

func (b *WebSocketBroadcaster) BroadcastBinaryExcluding(conversationID string, data []byte, exclude *websocket.Conn) {
	b.mu.RLock()
	conns, ok := b.connections[conversationID]
	if !ok || len(conns) == 0 {
		b.mu.RUnlock()
		return
	}
	targets := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		if conn != exclude {
			targets = append(targets, conn)
		}
	}
	b.mu.RUnlock()

	for _, conn := range targets {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("Failed to broadcast to WebSocket connection: %v", err)
			b.Unsubscribe(conversationID, conn)
		}
	}
}
