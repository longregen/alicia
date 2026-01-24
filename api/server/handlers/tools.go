package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/services"
)

// ToolHandler handles tool use review endpoints.
// Note: Tools are global resources shared across all users.
// Tool uses are associated with messages and inherit authorization from conversation ownership.
type ToolHandler struct {
	toolSvc *services.ToolService
	msgSvc  *services.MessageService
	convSvc *services.ConversationService
}

// NewToolHandler creates a new tool handler.
func NewToolHandler(toolSvc *services.ToolService, msgSvc *services.MessageService, convSvc *services.ConversationService) *ToolHandler {
	return &ToolHandler{toolSvc: toolSvc, msgSvc: msgSvc, convSvc: convSvc}
}

// ListTools handles GET /tools
func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.toolSvc.ListTools(r.Context())
	if err != nil {
		respondError(w, "failed to list tools", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"tools": tools,
	}, http.StatusOK)
}

// ListToolUses handles GET /tool-uses
func (h *ToolHandler) ListToolUses(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)

	uses, total, err := h.toolSvc.ListToolUses(r.Context(), limit, offset)
	if err != nil {
		respondError(w, "failed to list tool uses", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"tool_uses": uses,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	}, http.StatusOK)
}

// GetToolUse handles GET /tool-uses/{id}
func (h *ToolHandler) GetToolUse(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	tu, err := h.toolSvc.GetToolUse(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, "tool use not found", http.StatusNotFound)
		} else {
			respondError(w, "failed to get tool use", http.StatusInternalServerError)
		}
		return
	}

	// Verify user owns the conversation via the message
	msg, err := h.msgSvc.GetMessage(r.Context(), tu.MessageID)
	if err != nil {
		respondError(w, "tool use not found", http.StatusNotFound)
		return
	}

	_, err = h.convSvc.GetByUser(r.Context(), msg.ConversationID, userID)
	if err != nil {
		respondError(w, "tool use not found", http.StatusNotFound)
		return
	}

	respondJSON(w, tu, http.StatusOK)
}

// GetToolUsesByMessage handles GET /messages/{id}/tool-uses
func (h *ToolHandler) GetToolUsesByMessage(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	msgID := chi.URLParam(r, "id")

	// Get the message to find its conversation
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

	uses, err := h.toolSvc.GetToolUsesByMessage(r.Context(), msgID)
	if err != nil {
		respondError(w, "failed to get tool uses", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"tool_uses": uses,
	}, http.StatusOK)
}
