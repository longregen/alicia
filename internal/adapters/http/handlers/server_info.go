package handlers

import (
	"net/http"
	"time"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

type ServerInfoHandler struct {
	config           *config.Config
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	mcpAdapter       *mcp.Adapter
	sessionStartTime time.Time
}

func NewServerInfoHandler(
	cfg *config.Config,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	mcpAdapter *mcp.Adapter,
) *ServerInfoHandler {
	return &ServerInfoHandler{
		config:           cfg,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		mcpAdapter:       mcpAdapter,
		sessionStartTime: time.Now(),
	}
}

func (h *ServerInfoHandler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	connection := protocol.ConnectionInfo{
		Status:  "connected",
		Latency: 0,
	}

	model := protocol.ModelInfo{
		Name:     h.config.LLM.Model,
		Provider: "openai",
	}

	var mcpServers []protocol.MCPServerInfo
	if h.mcpAdapter != nil {
		serverStatus := h.mcpAdapter.GetServerStatus()
		for name, connected := range serverStatus {
			status := "disconnected"
			if connected {
				status = "connected"
			}
			mcpServers = append(mcpServers, protocol.MCPServerInfo{
				Name:   name,
				Status: status,
			})
		}
	}

	if mcpServers == nil {
		mcpServers = []protocol.MCPServerInfo{}
	}

	response := protocol.ServerInfo{
		Connection: connection,
		Model:      model,
		MCPServers: mcpServers,
	}

	respondJSON(w, response, http.StatusOK)
}

type SessionStatsResponse struct {
	protocol.SessionStats
	ConversationID string `json:"conversationId,omitempty"`
}

func (h *ServerInfoHandler) GetSessionStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	conv, err := h.conversationRepo.GetByIDAndUserID(r.Context(), conversationID, userID)
	if err != nil {
		respondError(w, "not_found", "Conversation not found", http.StatusNotFound)
		return
	}

	messages, err := h.messageRepo.GetByConversation(r.Context(), conversationID)
	if err != nil {
		respondError(w, "internal_error", "Failed to get messages", http.StatusInternalServerError)
		return
	}

	messageCount := len(messages)
	toolCallCount := 0
	for _, msg := range messages {
		toolCallCount += len(msg.ToolUses)
	}

	sessionDuration := int(time.Since(conv.CreatedAt).Seconds())

	response := SessionStatsResponse{
		SessionStats: protocol.SessionStats{
			MessageCount:    messageCount,
			ToolCallCount:   toolCallCount,
			MemoriesUsed:    0,
			SessionDuration: sessionDuration,
		},
		ConversationID: conversationID,
	}

	respondJSON(w, response, http.StatusOK)
}

func (h *ServerInfoHandler) GetGlobalStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	serverUptime := int(time.Since(h.sessionStartTime).Seconds())

	conversations, err := h.conversationRepo.ListByUserID(r.Context(), userID, 1000, 0)
	if err != nil {
		respondError(w, "internal_error", "Failed to get conversations", http.StatusInternalServerError)
		return
	}

	totalMessages := 0
	totalToolCalls := 0
	for _, conv := range conversations {
		messages, err := h.messageRepo.GetByConversation(r.Context(), conv.ID)
		if err != nil {
			continue
		}
		totalMessages += len(messages)
		for _, msg := range messages {
			totalToolCalls += len(msg.ToolUses)
		}
	}

	response := protocol.SessionStats{
		MessageCount:    totalMessages,
		ToolCallCount:   totalToolCalls,
		MemoriesUsed:    0,
		SessionDuration: serverUptime,
	}

	respondJSON(w, response, http.StatusOK)
}
