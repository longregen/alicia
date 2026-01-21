package handlers

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/vmihailenco/msgpack/v5"

	_ "github.com/longregen/alicia/internal/adapters/http/encoding"
)

type MultiplexedWSHandler struct {
	upgrader         websocket.Upgrader
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	idGen            ports.IDGenerator
	broadcaster      *WebSocketBroadcaster
}

type connectionState struct {
	conn          *websocket.Conn
	subscriptions map[string]struct{}
	mu            sync.RWMutex
	stanzaID      int32
	isAgent       bool
}

func (cs *connectionState) nextStanzaID() int32 {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.stanzaID--
	return cs.stanzaID
}

func (cs *connectionState) subscribe(conversationID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.subscriptions[conversationID] = struct{}{}
}

func (cs *connectionState) unsubscribe(conversationID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.subscriptions, conversationID)
}

func (cs *connectionState) isSubscribed(conversationID string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	_, ok := cs.subscriptions[conversationID]
	return ok
}

func (cs *connectionState) getSubscriptions() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	subs := make([]string, 0, len(cs.subscriptions))
	for id := range cs.subscriptions {
		subs = append(subs, id)
	}
	return subs
}

func NewMultiplexedWSHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	idGen ports.IDGenerator,
	broadcaster *WebSocketBroadcaster,
	allowedOrigins []string,
) *MultiplexedWSHandler {
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginsMap[origin] = true
	}

	return &MultiplexedWSHandler{
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

func (h *MultiplexedWSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	state := &connectionState{
		conn:          conn,
		subscriptions: make(map[string]struct{}),
		stanzaID:      0,
	}

	log.Printf("Multiplexed WebSocket connection established")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	defer func() {
		for _, convID := range state.getSubscriptions() {
			h.broadcaster.Unsubscribe(convID, conn)
		}
		if state.isAgent {
			h.broadcaster.UnsubscribeAgent(conn)
		}
		log.Printf("Multiplexed WebSocket connection closed, cleaned up %d subscriptions", len(state.subscriptions))
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		h.readPump(ctx, state)
		cancel()
	}()

	go func() {
		defer wg.Done()
		h.writePump(ctx, state)
	}()

	wg.Wait()
}

func (h *MultiplexedWSHandler) readPump(ctx context.Context, state *connectionState) {
	defer state.conn.Close()

	state.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	state.conn.SetPongHandler(func(string) error {
		state.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		messageType, data, err := state.conn.ReadMessage()
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

		var envelope protocol.Envelope
		if err := msgpack.Unmarshal(data, &envelope); err != nil {
			log.Printf("Failed to decode envelope: %v", err)
			h.sendError(state, "", "invalid_message", "Failed to decode message")
			continue
		}

		switch envelope.Type {
		case protocol.TypeSubscribe:
			h.handleSubscribe(ctx, state, &envelope)
		case protocol.TypeUnsubscribe:
			h.handleUnsubscribe(state, &envelope)
		case protocol.TypeSyncRequest:
			if state.isSubscribed(envelope.ConversationID) {
				h.handleSyncRequest(ctx, state, &envelope)
			} else {
				h.sendError(state, envelope.ConversationID, "not_subscribed", "Not subscribed to conversation")
			}
		default:
			if state.isAgent && envelope.ConversationID != "" {
				h.broadcaster.BroadcastBinaryExcluding(envelope.ConversationID, data, state.conn)
			} else if envelope.ConversationID != "" && state.isSubscribed(envelope.ConversationID) {
				broadcastData, _ := msgpack.Marshal(&envelope)
				h.broadcaster.BroadcastBinary(envelope.ConversationID, broadcastData)
			}
		}
	}
}

func (h *MultiplexedWSHandler) handleSubscribe(ctx context.Context, state *connectionState, envelope *protocol.Envelope) {
	bodyBytes, err := msgpack.Marshal(envelope.Body)
	if err != nil {
		h.sendSubscribeAck(state, "", false, "Invalid subscribe request body")
		return
	}

	var req dto.SubscribeRequest
	if err := msgpack.Unmarshal(bodyBytes, &req); err != nil {
		h.sendSubscribeAck(state, "", false, "Invalid subscribe request")
		return
	}

	if req.AgentMode {
		state.isAgent = true
		h.broadcaster.SubscribeAgent(state.conn)
		log.Printf("Agent connected and subscribed for response generation")
		h.sendSubscribeAck(state, "", true, "")
		return
	}

	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = envelope.ConversationID
	}

	if conversationID == "" {
		h.sendSubscribeAck(state, "", false, "Conversation ID required")
		return
	}

	h.broadcaster.Subscribe(conversationID, state.conn)
	state.subscribe(conversationID)

	log.Printf("Client subscribed to conversation %s", conversationID)
	h.sendSubscribeAck(state, conversationID, true, "")
}

func (h *MultiplexedWSHandler) handleUnsubscribe(state *connectionState, envelope *protocol.Envelope) {
	bodyBytes, err := msgpack.Marshal(envelope.Body)
	if err != nil {
		h.sendUnsubscribeAck(state, "", false)
		return
	}

	var req dto.UnsubscribeRequest
	if err := msgpack.Unmarshal(bodyBytes, &req); err != nil {
		h.sendUnsubscribeAck(state, "", false)
		return
	}

	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = envelope.ConversationID
	}

	if conversationID == "" {
		h.sendUnsubscribeAck(state, "", false)
		return
	}

	h.broadcaster.Unsubscribe(conversationID, state.conn)
	state.unsubscribe(conversationID)

	log.Printf("Client unsubscribed from conversation %s", conversationID)
	h.sendUnsubscribeAck(state, conversationID, true)
}

func (h *MultiplexedWSHandler) handleSyncRequest(ctx context.Context, state *connectionState, envelope *protocol.Envelope) {
	bodyBytes, err := msgpack.Marshal(envelope.Body)
	if err != nil {
		h.sendError(state, envelope.ConversationID, "invalid_message", "Invalid sync request body")
		return
	}

	var syncReq dto.SyncRequest
	if err := msgpack.Unmarshal(bodyBytes, &syncReq); err != nil {
		h.sendError(state, envelope.ConversationID, "invalid_message", "Invalid sync request")
		return
	}

	syncedMessages := make([]dto.SyncedMessage, 0, len(syncReq.Messages))
	for _, msgReq := range syncReq.Messages {
		syncedMsg := h.processMessage(ctx, envelope.ConversationID, msgReq)
		syncedMessages = append(syncedMessages, syncedMsg)
	}

	response := &dto.SyncResponse{
		SyncedMessages: syncedMessages,
		SyncedAt:       time.Now().Format(time.RFC3339),
	}

	responseEnvelope := protocol.NewEnvelope(
		state.nextStanzaID(),
		envelope.ConversationID,
		protocol.TypeSyncResponse,
		response,
	)

	responseData, err := msgpack.Marshal(responseEnvelope)
	if err != nil {
		log.Printf("Failed to encode sync response: %v", err)
		return
	}

	state.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := state.conn.WriteMessage(websocket.BinaryMessage, responseData); err != nil {
		log.Printf("Failed to write sync response: %v", err)
	}
}

func (h *MultiplexedWSHandler) processMessage(ctx context.Context, conversationID string, msgReq dto.SyncMessageRequest) dto.SyncedMessage {
	if msgReq.LocalID == "" {
		return dto.ToSyncedMessageWithConflict(msgReq.LocalID, "Local ID is required", nil)
	}

	if msgReq.Role == "" {
		return dto.ToSyncedMessageWithConflict(msgReq.LocalID, "Message role is required", nil)
	}

	return dto.SyncedMessage{
		LocalID:  msgReq.LocalID,
		ServerID: h.idGen.GenerateMessageID(),
		Status:   "synced",
	}
}

func (h *MultiplexedWSHandler) writePump(ctx context.Context, state *connectionState) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := state.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

func (h *MultiplexedWSHandler) sendSubscribeAck(state *connectionState, conversationID string, success bool, errorMsg string) {
	ack := dto.SubscribeAck{
		ConversationID: conversationID,
		Success:        success,
		Error:          errorMsg,
	}

	envelope := protocol.NewEnvelope(
		state.nextStanzaID(),
		conversationID,
		protocol.TypeSubscribeAck,
		ack,
	)

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		log.Printf("Failed to encode subscribe ack: %v", err)
		return
	}

	state.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := state.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("Failed to send subscribe ack: %v", err)
	}
}

func (h *MultiplexedWSHandler) sendUnsubscribeAck(state *connectionState, conversationID string, success bool) {
	ack := dto.UnsubscribeAck{
		ConversationID: conversationID,
		Success:        success,
	}

	envelope := protocol.NewEnvelope(
		state.nextStanzaID(),
		conversationID,
		protocol.TypeUnsubscribeAck,
		ack,
	)

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		log.Printf("Failed to encode unsubscribe ack: %v", err)
		return
	}

	state.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := state.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("Failed to send unsubscribe ack: %v", err)
	}
}

func (h *MultiplexedWSHandler) sendError(state *connectionState, conversationID string, errorType, message string) {
	errorResp := dto.NewErrorResponse(errorType, message, http.StatusBadRequest)

	envelope := protocol.NewEnvelope(
		state.nextStanzaID(),
		conversationID,
		protocol.TypeErrorMessage,
		errorResp,
	)

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		log.Printf("Failed to encode error response: %v", err)
		return
	}

	state.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := state.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}
