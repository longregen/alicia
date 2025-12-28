package handlers

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/vmihailenco/msgpack/v5"
)

// WebSocketSyncHandler handles WebSocket-based message synchronization
type WebSocketSyncHandler struct {
	upgrader         websocket.Upgrader
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	idGen            ports.IDGenerator
	broadcaster      *WebSocketBroadcaster
}

// NewWebSocketSyncHandler creates a new WebSocket sync handler
func NewWebSocketSyncHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	idGen ports.IDGenerator,
	broadcaster *WebSocketBroadcaster,
) *WebSocketSyncHandler {
	return &WebSocketSyncHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper CORS origin checking
				return true
			},
		},
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		idGen:            idGen,
		broadcaster:      broadcaster,
	}
}

// Handle upgrades HTTP connection to WebSocket and manages message sync
func (h *WebSocketSyncHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	// Verify conversation exists, is active, and belongs to the user
	conversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), conversationID, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Conversation not found or access denied", http.StatusNotFound)
		} else {
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	if !requireActiveConversation(conversation, w) {
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	// Subscribe to conversation broadcasts
	h.broadcaster.Subscribe(conversationID, conn)
	defer h.broadcaster.Unsubscribe(conversationID, conn)

	log.Printf("WebSocket connection established for conversation %s (user %s)", conversationID, userID)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Use WaitGroup to coordinate goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Start read pump (reads messages from client)
	go func() {
		defer wg.Done()
		h.readPump(ctx, conn, conversationID)
		cancel() // Cancel context when read pump exits
	}()

	// Start write pump (sends heartbeats and responses)
	go func() {
		defer wg.Done()
		h.writePump(ctx, conn, conversationID)
	}()

	// Wait for both pumps to finish
	wg.Wait()
	log.Printf("WebSocket connection closed for conversation %s", conversationID)
}

// readPump reads messages from the WebSocket connection
func (h *WebSocketSyncHandler) readPump(ctx context.Context, conn *websocket.Conn, conversationID string) {
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read message
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		if messageType != websocket.BinaryMessage {
			log.Printf("Received non-binary message, ignoring")
			continue
		}

		// Decode MessagePack message
		var syncReq dto.SyncRequest
		if err := msgpack.Unmarshal(data, &syncReq); err != nil {
			log.Printf("Failed to decode MessagePack: %v", err)
			h.sendError(conn, "invalid_message", "Failed to decode message")
			continue
		}

		// Process sync request
		response := h.processSyncRequest(ctx, conversationID, &syncReq)

		// Send response
		responseData, err := msgpack.Marshal(response)
		if err != nil {
			log.Printf("Failed to encode response: %v", err)
			continue
		}

		if err := conn.WriteMessage(websocket.BinaryMessage, responseData); err != nil {
			log.Printf("Failed to write response: %v", err)
			return
		}
	}
}

// writePump sends periodic ping messages to keep the connection alive
func (h *WebSocketSyncHandler) writePump(ctx context.Context, conn *websocket.Conn, conversationID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

// processSyncRequest processes a sync request and returns the response
func (h *WebSocketSyncHandler) processSyncRequest(ctx context.Context, conversationID string, req *dto.SyncRequest) *dto.SyncResponse {
	syncedMessages := make([]dto.SyncedMessage, 0, len(req.Messages))

	for _, msgReq := range req.Messages {
		syncedMsg, err := h.processMessage(ctx, conversationID, msgReq)
		if err != nil {
			syncedMessages = append(syncedMessages, dto.ToSyncedMessageWithConflict(
				msgReq.LocalID,
				"Internal error: "+err.Error(),
				nil,
			))
			continue
		}
		syncedMessages = append(syncedMessages, syncedMsg)
	}

	return &dto.SyncResponse{
		SyncedMessages: syncedMessages,
		SyncedAt:       time.Now().Format(time.RFC3339),
	}
}

// processMessage processes a single message sync request
func (h *WebSocketSyncHandler) processMessage(ctx context.Context, conversationID string, msgReq dto.SyncMessageRequest) (dto.SyncedMessage, error) {
	// Validation
	if msgReq.LocalID == "" {
		return dto.ToSyncedMessageWithConflict(
			msgReq.LocalID,
			"Local ID is required",
			nil,
		), nil
	}

	if msgReq.Role == "" {
		return dto.ToSyncedMessageWithConflict(
			msgReq.LocalID,
			"Message role is required",
			nil,
		), nil
	}

	// Check if message with this local ID already exists
	existingMsg, err := h.messageRepo.GetByLocalID(ctx, msgReq.LocalID)
	if err != nil && err != pgx.ErrNoRows {
		return dto.SyncedMessage{}, err
	}

	// If message already exists, check for conflicts
	if existingMsg != nil {
		if existingMsg.Contents == msgReq.Contents {
			return dto.ToSyncedMessage(existingMsg), nil
		}

		// Content differs - conflict detected
		existingMsg.MarkAsConflict()
		if err := h.messageRepo.Update(ctx, existingMsg); err != nil {
			return dto.SyncedMessage{}, err
		}

		return dto.ToSyncedMessageWithConflict(
			msgReq.LocalID,
			"Content mismatch with existing message",
			existingMsg,
		), nil
	}

	// Create new message
	serverID := h.idGen.GenerateMessageID()

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, msgReq.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	updatedAt := createdAt
	if msgReq.UpdatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, msgReq.UpdatedAt); err == nil {
			updatedAt = parsed
		}
	}

	// Create message with sync tracking
	message := &models.Message{
		ID:               serverID,
		ConversationID:   conversationID,
		SequenceNumber:   msgReq.SequenceNumber,
		PreviousID:       msgReq.PreviousID,
		Role:             models.MessageRole(msgReq.Role),
		Contents:         msgReq.Contents,
		LocalID:          msgReq.LocalID,
		ServerID:         serverID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	// Mark as synced
	now := time.Now()
	message.SyncedAt = &now

	// Save to database
	if err := h.messageRepo.Create(ctx, message); err != nil {
		return dto.SyncedMessage{}, err
	}

	// Broadcast to other WebSocket clients
	messageResponse := (&dto.MessageResponse{}).FromModel(message)
	h.broadcaster.BroadcastMessage(conversationID, messageResponse)

	return dto.ToSyncedMessage(message), nil
}

// sendError sends an error message to the WebSocket client
func (h *WebSocketSyncHandler) sendError(conn *websocket.Conn, errorType, message string) {
	errorResp := dto.NewErrorResponse(errorType, message, http.StatusBadRequest)
	data, err := msgpack.Marshal(errorResp)
	if err != nil {
		log.Printf("Failed to encode error response: %v", err)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}
