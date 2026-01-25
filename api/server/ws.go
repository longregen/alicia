package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/longregen/alicia/api/config"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/protocol"
	"github.com/longregen/alicia/api/server/handlers"
	"github.com/longregen/alicia/api/store"
	"github.com/longregen/alicia/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const WriteTimeout = 10 * time.Second

// SyncResult is the result returned when waiting synchronously for an agent response.
type SyncResult struct {
	AssistantMessage *protocol.AssistantMessage
	ToolUses         []protocol.ToolUseRequest
	Error            string
}

type Hub struct {
	convSubs     map[string]map[*websocket.Conn]struct{}
	convMu       sync.RWMutex
	agentConn    *websocket.Conn
	agentMu      sync.RWMutex
	voiceConn    *websocket.Conn
	voiceMu      sync.RWMutex
	assistantConn          *websocket.Conn
	assistantMu            sync.RWMutex
	assistantTools         []protocol.AssistantTool
	lastAssistantHeartbeat time.Time
	monitorConns map[*websocket.Conn]struct{}
	monitorMu    sync.RWMutex
	// Enables blocking request/response pattern over async WebSocket for the REST API's sync endpoint
	syncWaiters      map[string]chan SyncResult
	syncToolUses     map[string][]protocol.ToolUseRequest // tool uses accumulated by assistant message ID for sync responses
	syncToolUsesKeys map[string][]string                  // waiter key -> list of assistant message IDs accumulated
	syncMu           sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		convSubs:         make(map[string]map[*websocket.Conn]struct{}),
		monitorConns:     make(map[*websocket.Conn]struct{}),
		syncWaiters:      make(map[string]chan SyncResult),
		syncToolUses:     make(map[string][]protocol.ToolUseRequest),
		syncToolUsesKeys: make(map[string][]string),
	}
}

func (h *Hub) Subscribe(convID string, conn *websocket.Conn) {
	h.convMu.Lock()
	defer h.convMu.Unlock()

	if h.convSubs[convID] == nil {
		h.convSubs[convID] = make(map[*websocket.Conn]struct{})
	}
	h.convSubs[convID][conn] = struct{}{}
	slog.Info("ws: subscribed", "conversation_id", convID, "total", len(h.convSubs[convID]))
}

func (h *Hub) Unsubscribe(convID string, conn *websocket.Conn) {
	h.convMu.Lock()
	defer h.convMu.Unlock()

	if subs, ok := h.convSubs[convID]; ok {
		delete(subs, conn)
		if len(subs) == 0 {
			delete(h.convSubs, convID)
		}
		slog.Info("ws: unsubscribed", "conversation_id", convID)
	}
}

func (h *Hub) UnsubscribeAll(conn *websocket.Conn) {
	h.convMu.Lock()
	defer h.convMu.Unlock()

	for convID, subs := range h.convSubs {
		delete(subs, conn)
		if len(subs) == 0 {
			delete(h.convSubs, convID)
		}
	}
}

func (h *Hub) SubscribeAgent(conn *websocket.Conn) {
	h.agentMu.Lock()
	defer h.agentMu.Unlock()
	h.agentConn = conn
	slog.Info("ws: agent connected")
}

func (h *Hub) UnsubscribeAgent(conn *websocket.Conn) {
	h.agentMu.Lock()
	defer h.agentMu.Unlock()
	if h.agentConn == conn {
		h.agentConn = nil
		slog.Info("ws: agent disconnected")
	}
}

func (h *Hub) SubscribeVoice(conn *websocket.Conn) {
	h.voiceMu.Lock()
	defer h.voiceMu.Unlock()
	h.voiceConn = conn
	slog.Info("ws: voice-helper connected")
}

func (h *Hub) UnsubscribeVoice(conn *websocket.Conn) {
	h.voiceMu.Lock()
	defer h.voiceMu.Unlock()
	if h.voiceConn == conn {
		h.voiceConn = nil
		slog.Info("ws: voice-helper disconnected")
	}
}

func (h *Hub) SubscribeMonitor(conn *websocket.Conn) {
	h.monitorMu.Lock()
	defer h.monitorMu.Unlock()
	h.monitorConns[conn] = struct{}{}
	slog.Info("ws: monitor connected", "total", len(h.monitorConns))
}

func (h *Hub) UnsubscribeMonitor(conn *websocket.Conn) {
	h.monitorMu.Lock()
	defer h.monitorMu.Unlock()
	delete(h.monitorConns, conn)
	slog.Info("ws: monitor disconnected", "total", len(h.monitorConns))
}

func (h *Hub) SubscribeAssistant(conn *websocket.Conn) {
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()
	h.assistantConn = conn
	h.lastAssistantHeartbeat = time.Now()
	slog.Info("ws: assistant connected")
}

func (h *Hub) updateAssistantHeartbeat() {
	h.assistantMu.Lock()
	h.lastAssistantHeartbeat = time.Now()
	h.assistantMu.Unlock()
}

func (h *Hub) UnsubscribeAssistant(conn *websocket.Conn) {
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()
	if h.assistantConn == conn {
		h.assistantConn = nil
		h.assistantTools = nil
		slog.Info("ws: assistant disconnected")
	}
}

func (h *Hub) SetAssistantTools(tools []protocol.AssistantTool) {
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()
	h.assistantTools = tools
}

func (h *Hub) BroadcastToAssistant(data []byte) {
	h.broadcastToMonitors(data, "server", "assistant")

	h.assistantMu.RLock()
	conn := h.assistantConn
	h.assistantMu.RUnlock()

	if conn == nil {
		slog.Warn("ws: no assistant connected")
		return
	}

	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		slog.Error("ws: assistant send error", "error", err)
	}
}

type monitorFrame struct {
	Src  string `msgpack:"src"`
	Dst  string `msgpack:"dst"`
	Data []byte `msgpack:"data"`
}

func (h *Hub) broadcastToMonitors(data []byte, src, dst string) {
	h.monitorMu.RLock()
	if len(h.monitorConns) == 0 {
		h.monitorMu.RUnlock()
		return
	}
	conns := make([]*websocket.Conn, 0, len(h.monitorConns))
	for conn := range h.monitorConns {
		conns = append(conns, conn)
	}
	h.monitorMu.RUnlock()

	frame, err := msgpack.Marshal(&monitorFrame{Src: src, Dst: dst, Data: data})
	if err != nil {
		return
	}

	for _, conn := range conns {
		conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
		if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			h.UnsubscribeMonitor(conn)
		}
	}
}

func (h *Hub) BroadcastToVoice(data []byte) {
	h.broadcastToMonitors(data, "server", "voice")

	h.voiceMu.RLock()
	conn := h.voiceConn
	h.voiceMu.RUnlock()

	if conn == nil {
		slog.Warn("ws: no voice-helper connected")
		return
	}

	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		slog.Error("ws: voice send error", "error", err)
	}
}

func (h *Hub) BroadcastToConversation(convID string, data []byte) {
	h.broadcastToMonitors(data, "server", "client")

	h.convMu.RLock()
	subs := make([]*websocket.Conn, 0, len(h.convSubs[convID]))
	for conn := range h.convSubs[convID] {
		subs = append(subs, conn)
	}
	h.convMu.RUnlock()

	for _, conn := range subs {
		conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			slog.Warn("ws: broadcast error (client likely disconnected)", "error", err, "conversation_id", convID)
			h.Unsubscribe(convID, conn)
		}
	}
}

func (h *Hub) BroadcastToAgent(data []byte) {
	h.broadcastToMonitors(data, "server", "agent")

	h.agentMu.RLock()
	conn := h.agentConn
	h.agentMu.RUnlock()

	if conn == nil {
		slog.Warn("ws: no agent connected")
		return
	}

	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		slog.Error("ws: agent send error", "error", err)
	}
}

func (h *Hub) SendGenerationRequest(ctx context.Context, convID, userMsgID string, previousID *string, usePareto bool) {
	req := protocol.GenerationRequest{
		ConversationID:  convID,
		MessageID:       userMsgID,
		RequestType:     "send",
		EnableTools:     true,
		EnableReasoning: true,
		EnableStreaming:  true,
		UsePareto:       usePareto,
	}
	if previousID != nil {
		req.PreviousID = *previousID
	}

	env := protocol.NewEnvelope(convID, protocol.TypeGenRequest, req)
	tc := otel.InjectToTraceContext(ctx, convID, otel.UserIDFromContext(ctx))
	env.TraceID = tc.TraceID
	env.SpanID = tc.SpanID
	env.TraceFlags = tc.TraceFlags
	env.SessionID = tc.SessionID
	env.UserID = tc.UserID

	data, err := env.Encode()
	if err != nil {
		slog.Error("ws: encode generation request error", "error", err)
		return
	}

	h.BroadcastToAgent(data)
}

func (h *Hub) BroadcastEnvelope(convID string, msgType protocol.MessageType, body any) {
	env := protocol.NewEnvelope(convID, msgType, body)
	data, err := env.Encode()
	if err != nil {
		slog.Error("ws: encode envelope error", "error", err)
		return
	}
	h.BroadcastToConversation(convID, data)
}

func (h *Hub) BroadcastPreferencesUpdate(prefs *domain.UserPreferences) {
	update := protocol.PreferencesUpdate{
		UserID:                   prefs.UserID,
		Theme:                    prefs.Theme,
		AudioOutputEnabled:       prefs.AudioOutputEnabled,
		VoiceSpeed:               prefs.VoiceSpeed,
		MemoryMinImportance:      prefs.MemoryMinImportance,
		MemoryMinHistorical:      prefs.MemoryMinHistorical,
		MemoryMinPersonal:        prefs.MemoryMinPersonal,
		MemoryMinFactual:         prefs.MemoryMinFactual,
		MemoryRetrievalCount:     prefs.MemoryRetrievalCount,
		MaxTokens:                prefs.MaxTokens,
		Temperature:              prefs.Temperature,
		ParetoTargetScore:        prefs.ParetoTargetScore,
		ParetoMaxGenerations:     prefs.ParetoMaxGenerations,
		ParetoBranchesPerGen:     prefs.ParetoBranchesPerGen,
		ParetoArchiveSize:        prefs.ParetoArchiveSize,
		ParetoEnableCrossover:    prefs.ParetoEnableCrossover,
		NotesSimilarityThreshold: prefs.NotesSimilarityThreshold,
		NotesMaxCount:            prefs.NotesMaxCount,
		ConfirmDeleteMemory:      prefs.ConfirmDeleteMemory,
		ShowRelevanceScores:      prefs.ShowRelevanceScores,
	}

	env := protocol.NewEnvelope("", protocol.TypePreferencesUpdate, update)
	data, err := env.Encode()
	if err != nil {
		slog.Error("ws: encode preferences update error", "error", err)
		return
	}

	h.BroadcastToAgent(data)
	h.BroadcastToVoice(data)
	slog.Info("ws: broadcasted preferences update", "user_id", prefs.UserID)
}

func (h *Hub) SendGenerationRequestSync(ctx context.Context, convID, userMsgID string, previousID *string, usePareto bool) (*SyncResult, error) {
	// Register a waiter for this conversation
	key := convID + ":" + userMsgID
	ch := make(chan SyncResult, 1)

	h.syncMu.Lock()
	h.syncWaiters[key] = ch
	h.syncMu.Unlock()

	defer func() {
		h.syncMu.Lock()
		delete(h.syncWaiters, key)
		for _, msgID := range h.syncToolUsesKeys[key] {
			delete(h.syncToolUses, msgID)
		}
		delete(h.syncToolUsesKeys, key)
		h.syncMu.Unlock()
	}()

	// Send the generation request
	req := protocol.GenerationRequest{
		ConversationID:  convID,
		MessageID:       userMsgID,
		RequestType:     "send",
		EnableTools:     true,
		EnableReasoning: true,
		EnableStreaming:  false,
		UsePareto:       usePareto,
	}
	if previousID != nil {
		req.PreviousID = *previousID
	}

	env := protocol.NewEnvelope(convID, protocol.TypeGenRequest, req)
	tc := otel.InjectToTraceContext(ctx, convID, otel.UserIDFromContext(ctx))
	env.TraceID = tc.TraceID
	env.SpanID = tc.SpanID
	env.TraceFlags = tc.TraceFlags
	env.SessionID = tc.SessionID
	env.UserID = tc.UserID

	data, err := env.Encode()
	if err != nil {
		return nil, fmt.Errorf("encode generation request: %w", err)
	}

	h.BroadcastToAgent(data)

	// Wait for result or context cancellation
	select {
	case result := <-ch:
		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// NotifySyncWaiter checks if there's a sync waiter for the given conversation and notifies it.
// Called when the agent sends an AssistantMessage. Returns true if a waiter was notified.
func (h *Hub) NotifySyncWaiter(convID string, msg *protocol.AssistantMessage) bool {
	if msg.PreviousID == "" {
		return false
	}

	key := convID + ":" + msg.PreviousID
	h.syncMu.Lock()
	defer h.syncMu.Unlock()

	ch, ok := h.syncWaiters[key]
	if !ok {
		return false
	}

	// Collect accumulated tool uses for this assistant message
	toolUses := h.syncToolUses[msg.ID]
	delete(h.syncToolUses, msg.ID)

	select {
	case ch <- SyncResult{AssistantMessage: msg, ToolUses: toolUses}:
		return true
	default:
		return false
	}
}

// NotifySyncWaiterToolUse records a tool use for sync waiters.
// Tool uses are accumulated by assistant message ID until the AssistantMessage arrives,
// then included in the SyncResult.
func (h *Hub) NotifySyncWaiterToolUse(convID string, tu *protocol.ToolUseRequest) {
	if tu.MessageID == "" {
		return
	}
	h.syncMu.Lock()
	defer h.syncMu.Unlock()
	h.syncToolUses[tu.MessageID] = append(h.syncToolUses[tu.MessageID], *tu)

	// Track this message ID for cleanup if the waiter times out
	for key := range h.syncWaiters {
		if len(key) > len(convID) && key[:len(convID)+1] == convID+":" {
			h.syncToolUsesKeys[key] = append(h.syncToolUsesKeys[key], tu.MessageID)
			break
		}
	}
}

func (h *Hub) WaitForGeneration(ctx context.Context, convID, userMsgID string, previousID *string, usePareto bool) (*handlers.SyncGenerationResult, error) {
	result, err := h.SendGenerationRequestSync(ctx, convID, userMsgID, previousID, usePareto)
	if err != nil {
		return nil, err
	}
	if result.AssistantMessage == nil {
		return nil, fmt.Errorf("no assistant message received")
	}

	// Convert protocol tool uses to handler tool use info
	genResult := &handlers.SyncGenerationResult{
		MessageID: result.AssistantMessage.ID,
		Content:   result.AssistantMessage.Content,
	}
	for _, tu := range result.ToolUses {
		genResult.ToolUses = append(genResult.ToolUses, handlers.ToolUseInfo{
			ID:        tu.ID,
			ToolName:  tu.ToolName,
			Arguments: tu.Arguments,
			Status:    domain.ToolUseStatusPending,
		})
	}
	return genResult, nil
}

type WSHandler struct {
	hub      *Hub
	cfg      *config.Config
	store    *store.Store
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *Hub, cfg *config.Config, s *store.Store) *WSHandler {
	h := &WSHandler{hub: hub, cfg: cfg, store: s}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

func (h *WSHandler) checkOrigin(r *http.Request) bool {
	allowedOrigins := h.cfg.Server.AllowedOrigins
	for _, o := range allowedOrigins {
		if o == "*" {
			return true
		}
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return h.cfg.Server.AllowEmptyOrigin
	}
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws: upgrade error", "error", err)
		return
	}
	defer conn.Close()

	var isAgent bool
	var isVoice bool
	var isMonitor bool
	var isAssistant bool

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("ws: read error", "error", err)
			}
			break
		}

		// Forward all inbound messages to monitors
		if !isMonitor {
			src := "client"
			if isAgent {
				src = "agent"
			} else if isVoice {
				src = "voice"
			} else if isAssistant {
				src = "assistant"
			}
			h.hub.broadcastToMonitors(data, src, "server")
		}

		env, err := protocol.DecodeEnvelope(data)
		if err != nil {
			slog.Error("ws: decode error", "error", err)
			continue
		}

		// Detached from connection context: message processing must complete
		// even if the client disconnects.
		func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer ctxCancel()
			if env.HasTraceContext() {
				ctx = otel.ExtractFromTraceContext(ctx, otel.TraceContext{
					TraceID:    env.TraceID,
					SpanID:     env.SpanID,
					TraceFlags: env.TraceFlags,
					SessionID:  env.SessionID,
					UserID:     env.UserID,
				})
			}

			switch env.Type {
			case protocol.TypeSubscribe:
				sub, err := protocol.DecodeBody[protocol.Subscribe](env)
				if err != nil {
					slog.Error("ws: decode subscribe error", "error", err)
					return
				}

				if sub.AgentMode {
					if !h.verifyAgentAuth(r) {
						slog.Warn("ws: agent auth failed")
						h.sendSubscribeAck(conn, "", true, false, "authentication required")
						return
					}
					isAgent = true
					h.hub.SubscribeAgent(conn)
					h.sendSubscribeAck(conn, "", true, true, "")
				} else if sub.VoiceMode {
					if !h.verifyAgentAuth(r) {
						slog.Warn("ws: voice auth failed")
						h.sendSubscribeAck(conn, "", false, false, "authentication required")
						return
					}
					isVoice = true
					h.hub.SubscribeVoice(conn)
					h.sendSubscribeAck(conn, "", false, true, "")
				} else if sub.MonitorMode {
					isMonitor = true
					h.hub.SubscribeMonitor(conn)
					h.sendSubscribeAck(conn, "", false, true, "")
				} else if sub.AssistantMode {
					if !h.verifyAgentAuth(r) {
						slog.Warn("ws: assistant auth failed")
						h.sendSubscribeAck(conn, "", false, false, "authentication required")
						return
					}
					isAssistant = true
					h.hub.SubscribeAssistant(conn)
					h.sendSubscribeAck(conn, "", false, true, "")
				} else if sub.ConversationID != "" {
					h.hub.Subscribe(sub.ConversationID, conn)
					h.sendSubscribeAck(conn, sub.ConversationID, false, true, "")
				}

			case protocol.TypeUnsubscribe:
				unsub, err := protocol.DecodeBody[protocol.Unsubscribe](env)
				if err != nil {
					return
				}
				h.hub.Unsubscribe(unsub.ConversationID, conn)

			case protocol.TypeVoiceJoinRequest:
				if env.ConversationID != "" {
					slog.Info("ws: voice join request", "conversation_id", env.ConversationID)
					h.hub.BroadcastToVoice(data)
				}

			case protocol.TypeVoiceJoinAck:
				if isVoice && env.ConversationID != "" {
					slog.Info("ws: voice join ack", "conversation_id", env.ConversationID)
					h.hub.BroadcastToConversation(env.ConversationID, data)
				}

			case protocol.TypeVoiceLeaveRequest:
				if env.ConversationID != "" {
					slog.Info("ws: voice leave request", "conversation_id", env.ConversationID)
					h.hub.BroadcastToVoice(data)
				}

			case protocol.TypeVoiceLeaveAck:
				if isVoice && env.ConversationID != "" {
					slog.Info("ws: voice leave ack", "conversation_id", env.ConversationID)
					h.hub.BroadcastToConversation(env.ConversationID, data)
				}

			case protocol.TypeVoiceSpeaking, protocol.TypeVoiceStatus:
				if isVoice && env.ConversationID != "" {
					h.hub.BroadcastToConversation(env.ConversationID, data)
				}

			case protocol.TypeUserMessage:
				if env.ConversationID != "" {
					// Handle user messages from any subscribed client (voice, assistant, or regular)
					h.handleClientUserMessage(ctx, env)
				}

			case protocol.TypeGenRequest:
				if !isAgent && !isVoice && env.ConversationID != "" {
					h.hub.BroadcastToAgent(data)
				}

			case protocol.TypeAssistantHeartbeat:
				if isAssistant {
					h.hub.updateAssistantHeartbeat()
					h.hub.broadcastToMonitors(data, "assistant", "monitor")
				}

			case protocol.TypeAssistantToolsRegister:
				if isAssistant {
					h.hub.updateAssistantHeartbeat()
					reg, err := protocol.DecodeBody[protocol.AssistantToolsRegister](env)
					if err != nil {
						slog.Error("ws: decode assistant tools register error", "error", err)
						return
					}
					h.hub.SetAssistantTools(reg.Tools)
					ack := protocol.AssistantToolsAck{Success: true, ToolCount: len(reg.Tools)}
					ackEnv := protocol.NewEnvelope("", protocol.TypeAssistantToolsAck, ack)
					ackData, err := ackEnv.Encode()
					if err != nil {
						slog.Error("ws: encode assistant tools ack error", "error", err)
						return
					}
					conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
					conn.WriteMessage(websocket.BinaryMessage, ackData)
					slog.Info("ws: assistant registered tools", "count", len(reg.Tools))
				}

			case protocol.TypeToolUseResult:
				if isAssistant {
					h.hub.updateAssistantHeartbeat()
					// Route tool results from assistant to monitors (for mcp-assistant bridge)
					h.hub.broadcastToMonitors(data, "assistant", "monitor")
				} else if isAgent && env.ConversationID != "" {
					h.persistAgentMessage(ctx, env)
					h.hub.BroadcastToConversation(env.ConversationID, data)
				}

			case protocol.TypeToolUseRequest:
				if isAgent && env.ConversationID != "" {
					h.persistAgentMessage(ctx, env)
					h.hub.BroadcastToConversation(env.ConversationID, data)
					// Route client-execution tools to the assistant device
					req, err := protocol.DecodeBody[protocol.ToolUseRequest](env)
					if err == nil && req.Execution == "client" {
						h.hub.BroadcastToAssistant(data)
					}
				} else if isMonitor {
					// mcp-assistant bridge sends tool requests via monitor mode
					req, err := protocol.DecodeBody[protocol.ToolUseRequest](env)
					if err == nil && req.Execution == "client" {
						h.hub.BroadcastToAssistant(data)
					}
				}

			default:
				if isAgent && env.ConversationID != "" {
					slog.Info("ws: agent->user", "type", env.Type, "conversation_id", env.ConversationID)
					h.persistAgentMessage(ctx, env)
					h.hub.BroadcastToConversation(env.ConversationID, data)
				}
			}
		}()
	}

	if isAgent {
		h.hub.UnsubscribeAgent(conn)
	} else if isVoice {
		h.hub.UnsubscribeVoice(conn)
	} else if isAssistant {
		h.hub.UnsubscribeAssistant(conn)
	} else if isMonitor {
		h.hub.UnsubscribeMonitor(conn)
	} else {
		h.hub.UnsubscribeAll(conn)
	}
}

func (h *WSHandler) verifyAgentAuth(r *http.Request) bool {
	secret := h.cfg.Server.AgentSecret
	if secret == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if auth == "Bearer "+secret {
		return true
	}
	if r.URL.Query().Get("agent_secret") == secret {
		return true
	}
	return false
}

func (h *WSHandler) sendSubscribeAck(conn *websocket.Conn, convID string, agentMode, success bool, errMsg string) {
	ack := protocol.SubscribeAck{
		ConversationID: convID,
		AgentMode:      agentMode,
		Success:        success,
		Error:          errMsg,
	}
	env := protocol.NewEnvelope(convID, protocol.TypeSubscribeAck, ack)
	data, err := env.Encode()
	if err != nil {
		slog.Error("ws: encode subscribe ack error", "error", err)
		return
	}
	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		slog.Error("ws: send subscribe ack error", "error", err)
	}
}

func (h *WSHandler) persistAgentMessage(ctx context.Context, env *protocol.Envelope) {
	if h.store == nil {
		return
	}

	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("alicia-api")
	ctx, span := tracer.Start(ctx, "ws.persistAgentMessage",
		trace.WithAttributes(
			attribute.String("conversation.id", env.ConversationID),
			attribute.Int("message.type", int(env.Type)),
		),
	)
	defer span.End()

	switch env.Type {
	case protocol.TypeAssistantMsg:
		msg, err := protocol.DecodeBody[protocol.AssistantMessage](env)
		if err != nil {
			slog.Error("ws: decode assistant message error", "error", err)
			return
		}
		if len(msg.Content) == 0 {
			slog.Warn("ws: assistant message has empty content", "message_id", msg.ID)
		}

		var previousID *string
		if msg.PreviousID != "" {
			previousID = &msg.PreviousID
		}

		dbMsg := &domain.Message{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			PreviousID:     previousID,
			Role:           domain.RoleAssistant,
			Content:        msg.Content,
			Reasoning:      msg.Reasoning,
			Status:         domain.MessageStatusCompleted,
			CreatedAt:      time.Now().UTC(),
		}

		if err := h.store.CreateMessage(ctx, dbMsg); err != nil {
			slog.Error("ws: save assistant message error", "error", err, "message_id", msg.ID)
			return
		}

		if err := h.store.UpdateConversationTip(ctx, msg.ConversationID, msg.ID); err != nil {
			slog.Error("ws: update conversation tip error", "error", err, "conversation_id", msg.ConversationID)
		}

		// Notify any sync waiters for this conversation
		h.hub.NotifySyncWaiter(msg.ConversationID, msg)

	case protocol.TypeToolUseResult:
		result, err := protocol.DecodeBody[protocol.ToolUseResult](env)
		if err != nil {
			slog.Error("ws: decode tool use result error", "error", err)
			return
		}

		tu, err := h.store.GetToolUse(ctx, result.RequestID)
		if err != nil {
			slog.Error("ws: get tool use error", "error", err, "request_id", result.RequestID)
			return
		}

		tu.Result = result.Result
		if result.Success {
			tu.Status = domain.ToolUseStatusSuccess
		} else {
			tu.Status = domain.ToolUseStatusError
			tu.Error = result.Error
		}

		if err := h.store.UpdateToolUse(ctx, tu); err != nil {
			slog.Error("ws: update tool use error", "error", err, "tool_use_id", tu.ID)
		}

	case protocol.TypeToolUseRequest:
		req, err := protocol.DecodeBody[protocol.ToolUseRequest](env)
		if err != nil {
			slog.Error("ws: decode tool use request error", "error", err)
			return
		}

		tu := &domain.ToolUse{
			ID:        req.ID,
			MessageID: req.MessageID,
			ToolName:  req.ToolName,
			Arguments: req.Arguments,
			Status:    domain.ToolUseStatusPending,
			CreatedAt: time.Now().UTC(),
		}

		if err := h.store.CreateToolUse(ctx, tu); err != nil {
			slog.Error("ws: save tool use error", "error", err, "tool_use_id", req.ID)
		}

		// Notify sync waiters about this tool use so it can be included in the response
		h.hub.NotifySyncWaiterToolUse(env.ConversationID, req)
	}
}

func (h *WSHandler) handleClientUserMessage(ctx context.Context, env *protocol.Envelope) {
	msg, err := protocol.DecodeBody[protocol.UserMessage](env)
	if err != nil {
		slog.Error("ws: decode user message error", "error", err)
		return
	}

	convID := env.ConversationID
	userID := env.UserID

	slog.Info("ws: client user message", "conversation_id", convID, "user_id", userID, "chars", len(msg.Content))

	if h.store == nil {
		slog.Error("ws: store not available for user message")
		return
	}

	if msg.Content == "" {
		slog.Warn("ws: user message has empty content", "conversation_id", convID)
		return
	}

	conv, err := h.store.GetConversation(ctx, convID)
	if err != nil {
		slog.Error("ws: get conversation for user message error", "error", err, "conversation_id", convID)
		return
	}

	previousID := conv.TipMessageID

	userMsg := &domain.Message{
		ID:             store.NewMessageID(),
		ConversationID: convID,
		PreviousID:     previousID,
		Role:           domain.RoleUser,
		Content:        msg.Content,
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}

	err = h.store.WithTx(ctx, func(ctx context.Context) error {
		if err := h.store.CreateMessage(ctx, userMsg); err != nil {
			return err
		}
		return h.store.UpdateConversationTip(ctx, convID, userMsg.ID)
	})
	if err != nil {
		slog.Error("ws: create user message error", "error", err, "conversation_id", convID)
		return
	}

	slog.Info("ws: user message created", "message_id", userMsg.ID, "conversation_id", convID)

	prevID := ""
	if previousID != nil {
		prevID = *previousID
	}
	h.hub.BroadcastEnvelope(convID, protocol.TypeUserMessage, &protocol.UserMessage{
		ID:             userMsg.ID,
		ConversationID: convID,
		Content:        msg.Content,
		PreviousID:     prevID,
	})

	h.hub.SendGenerationRequest(ctx, convID, userMsg.ID, previousID, false)
}
