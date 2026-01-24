package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/services"
)

type Broadcaster interface {
	SendGenerationRequest(ctx context.Context, convID, userMsgID string, previousID *string, usePareto bool)
}

type SyncGenerationResult struct {
	MessageID string
	Content   string
	ToolUses  []ToolUseInfo
}

type ToolUseInfo struct {
	ID       string         `json:"id"`
	ToolName string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Status   string         `json:"status"`
}

type SyncBroadcaster interface {
	Broadcaster
	WaitForGeneration(ctx context.Context, convID, userMsgID string, previousID *string, usePareto bool) (*SyncGenerationResult, error)
}

type MessageHandler struct {
	msgSvc  *services.MessageService
	convSvc *services.ConversationService
	hub     Broadcaster
}

func NewMessageHandler(msgSvc *services.MessageService, convSvc *services.ConversationService, hub Broadcaster) *MessageHandler {
	return &MessageHandler{msgSvc: msgSvc, convSvc: convSvc, hub: hub}
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")

	// Verify ownership
	conv, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}

	// If ?all=true, return all messages; otherwise return chain from tip (or all if no tip)
	var msgs any
	all := r.URL.Query().Get("all") == "true"

	if !all && conv.TipMessageID != nil {
		chain, err := h.msgSvc.GetMessageChain(r.Context(), *conv.TipMessageID)
		if err != nil {
			respondError(w, "failed to get messages", http.StatusInternalServerError)
			return
		}
		msgs = chain
	} else {
		limit := parseIntQuery(r, "limit", 1000)
		list, err := h.msgSvc.ListMessages(r.Context(), convID, limit)
		if err != nil {
			respondError(w, "failed to list messages", http.StatusInternalServerError)
			return
		}
		msgs = list
	}

	respondJSON(w, map[string]any{
		"messages": msgs,
	}, http.StatusOK)
}

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")
	syncMode := r.URL.Query().Get("sync") == "true"
	debugf("[MessageHandler.Create] convID=%s userID=%s sync=%v", convID, userID, syncMode)

	// Verify ownership
	conv, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		slog.Error("failed to get conversation for user", "error", err, "conversation_id", convID, "user_id", userID)
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}
	debugf("[MessageHandler.Create] conversation found: tipMessageID=%v", conv.TipMessageID)

	var req struct {
		Content    string  `json:"content"`
		PreviousID *string `json:"previous_id"`
		UsePareto  bool    `json:"use_pareto"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode message create request", "error", err)
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	debugf("[MessageHandler.Create] content=%q previousID=%v usePareto=%v", req.Content, req.PreviousID, req.UsePareto)

	if req.Content == "" {
		respondError(w, "content is required", http.StatusBadRequest)
		return
	}

	// Use tip as previous if not specified
	previousID := req.PreviousID
	if previousID == nil {
		previousID = conv.TipMessageID
	}
	debugf("[MessageHandler.Create] using previousID=%v", previousID)

	// Create the user message
	msg, err := h.msgSvc.CreateUserMessage(r.Context(), convID, req.Content, previousID)
	if err != nil {
		slog.Error("failed to create user message", "error", err, "conversation_id", convID)
		respondError(w, "failed to create message", http.StatusInternalServerError)
		return
	}
	debugf("[MessageHandler.Create] created message: id=%s", msg.ID)

	if syncMode {
		syncHub, ok := h.hub.(SyncBroadcaster)
		if !ok {
			respondError(w, "sync mode not supported", http.StatusNotImplemented)
			return
		}

		result, err := syncHub.WaitForGeneration(r.Context(), convID, msg.ID, previousID, req.UsePareto)
		if err != nil {
			slog.Error("sync generation failed", "error", err, "conversation_id", convID)
			respondError(w, "generation failed: "+err.Error(), http.StatusGatewayTimeout)
			return
		}

		assistantMsg := map[string]any{
			"id":              result.MessageID,
			"conversation_id": convID,
			"previous_id":     msg.ID,
			"role":            domain.RoleAssistant,
			"content":         result.Content,
			"status":          domain.MessageStatusCompleted,
			"tool_uses":       result.ToolUses,
		}

		respondJSON(w, map[string]any{
			"user_message":      msg,
			"assistant_message": assistantMsg,
		}, http.StatusOK)
		return
	}

	// Async mode: broadcast generation request to agent
	h.hub.SendGenerationRequest(r.Context(), convID, msg.ID, previousID, req.UsePareto)

	respondJSON(w, msg, http.StatusAccepted)
}

func (h *MessageHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	msgID := chi.URLParam(r, "id")

	msg, err := h.msgSvc.GetMessage(r.Context(), msgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, "message not found", http.StatusNotFound)
		} else {
			respondError(w, "failed to get message", http.StatusInternalServerError)
		}
		return
	}

	// Verify user owns the conversation
	_, err = h.convSvc.GetByUser(r.Context(), msg.ConversationID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, "message not found", http.StatusNotFound)
		} else {
			respondError(w, "failed to verify ownership", http.StatusInternalServerError)
		}
		return
	}

	respondJSON(w, msg, http.StatusOK)
}

func (h *MessageHandler) GetSiblings(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	msgID := chi.URLParam(r, "id")

	// First get the message to find its conversation
	msg, err := h.msgSvc.GetMessage(r.Context(), msgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, "message not found", http.StatusNotFound)
		} else {
			respondError(w, "failed to get message", http.StatusInternalServerError)
		}
		return
	}

	// Verify user owns the conversation
	_, err = h.convSvc.GetByUser(r.Context(), msg.ConversationID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, "message not found", http.StatusNotFound)
		} else {
			respondError(w, "failed to verify ownership", http.StatusInternalServerError)
		}
		return
	}

	siblings, err := h.msgSvc.GetMessageSiblings(r.Context(), msgID)
	if err != nil {
		respondError(w, "failed to get siblings", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"siblings": siblings,
	}, http.StatusOK)
}
