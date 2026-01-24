package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/services"
)

type ConversationHandler struct {
	convSvc *services.ConversationService
}

func NewConversationHandler(convSvc *services.ConversationService) *ConversationHandler {
	return &ConversationHandler{convSvc: convSvc}
}

func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	conv, err := h.convSvc.Create(r.Context(), userID, req.Title)
	if err != nil {
		respondError(w, "failed to create conversation", http.StatusInternalServerError)
		return
	}

	respondJSON(w, conv, http.StatusCreated)
}

func (h *ConversationHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")

	conv, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}

	respondJSON(w, conv, http.StatusOK)
}

func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)
	activeOnly := r.URL.Query().Get("active") == "true"

	var convs []*domain.Conversation
	var total int
	var err error

	if activeOnly {
		convs, total, err = h.convSvc.ListActive(r.Context(), userID, limit, offset)
	} else {
		convs, total, err = h.convSvc.List(r.Context(), userID, limit, offset)
	}

	if err != nil {
		respondError(w, "failed to list conversations", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"conversations": convs,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	}, http.StatusOK)
}

func (h *ConversationHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")

	conv, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}

	var req struct {
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title != nil {
		conv.Title = *req.Title
	}
	if req.Status != nil {
		if *req.Status != domain.ConversationStatusActive && *req.Status != domain.ConversationStatusArchived {
			respondError(w, "status must be 'active' or 'archived'", http.StatusBadRequest)
			return
		}
		conv.Status = *req.Status
	}

	if err := h.convSvc.Update(r.Context(), conv); err != nil {
		respondError(w, "failed to update conversation", http.StatusInternalServerError)
		return
	}

	respondJSON(w, conv, http.StatusOK)
}

func (h *ConversationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")

	_, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}

	if err := h.convSvc.Delete(r.Context(), convID); err != nil {
		respondError(w, "failed to delete conversation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
