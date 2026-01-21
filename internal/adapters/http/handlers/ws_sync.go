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

type WebSocketSyncHandler struct {
	upgrader         websocket.Upgrader
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	idGen            ports.IDGenerator
	broadcaster      *WebSocketBroadcaster
}

func NewWebSocketSyncHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	idGen ports.IDGenerator,
	broadcaster *WebSocketBroadcaster,
	allowedOrigins []string,
) *WebSocketSyncHandler {
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginsMap[origin] = true
	}

	return &WebSocketSyncHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return allowedOriginsMap[origin]
			},
		},
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		idGen:            idGen,
		broadcaster:      broadcaster,
	}
}

func (h *WebSocketSyncHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

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

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	h.broadcaster.Subscribe(conversationID, conn)
	defer h.broadcaster.Unsubscribe(conversationID, conn)

	log.Printf("WebSocket connection established for conversation %s (user %s)", conversationID, userID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		h.readPump(ctx, conn, conversationID)
		cancel()
	}()

	go func() {
		defer wg.Done()
		h.writePump(ctx, conn, conversationID)
	}()

	wg.Wait()
	log.Printf("WebSocket connection closed for conversation %s", conversationID)
}

func (h *WebSocketSyncHandler) readPump(ctx context.Context, conn *websocket.Conn, conversationID string) {
	defer conn.Close()

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

		var syncReq dto.SyncRequest
		if err := msgpack.Unmarshal(data, &syncReq); err != nil {
			log.Printf("Failed to decode MessagePack: %v", err)
			h.sendError(conn, "invalid_message", "Failed to decode message")
			continue
		}

		response := h.processSyncRequest(ctx, conversationID, &syncReq)

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

func (h *WebSocketSyncHandler) processMessage(ctx context.Context, conversationID string, msgReq dto.SyncMessageRequest) (dto.SyncedMessage, error) {
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

	existingMsg, err := h.messageRepo.GetByLocalID(ctx, msgReq.LocalID)
	if err != nil && err != pgx.ErrNoRows {
		return dto.SyncedMessage{}, err
	}

	if existingMsg != nil {
		if existingMsg.Contents == msgReq.Contents {
			return dto.ToSyncedMessage(existingMsg), nil
		}

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

	serverID := h.idGen.GenerateMessageID()

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

	now := time.Now()
	message.SyncedAt = &now

	if err := h.messageRepo.Create(ctx, message); err != nil {
		return dto.SyncedMessage{}, err
	}

	messageResponse := (&dto.MessageResponse{}).FromModel(message)
	h.broadcaster.BroadcastMessage(conversationID, messageResponse)

	return dto.ToSyncedMessage(message), nil
}

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
