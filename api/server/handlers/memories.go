package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/services"
)

type MemoryHandler struct {
	memorySvc *services.MemoryService
	msgSvc    *services.MessageService
	convSvc   *services.ConversationService
}

func NewMemoryHandler(svc *services.MemoryService, msgSvc *services.MessageService, convSvc *services.ConversationService) *MemoryHandler {
	return &MemoryHandler{memorySvc: svc, msgSvc: msgSvc, convSvc: convSvc}
}

func (h *MemoryHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)
	slog.Debug("listing memories", "limit", limit, "offset", offset)

	memories, total, err := h.memorySvc.ListMemories(r.Context(), limit, offset)
	if err != nil {
		slog.Error("failed to list memories", "error", err)
		respondError(w, "failed to list memories", http.StatusInternalServerError)
		return
	}
	slog.Debug("found memories", "count", len(memories), "total", total)

	respondJSON(w, map[string]any{
		"memories": memories,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}, http.StatusOK)
}

func (h *MemoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	memory, err := h.memorySvc.GetMemory(r.Context(), id)
	if err != nil {
		respondError(w, "memory not found", http.StatusNotFound)
		return
	}

	respondJSON(w, memory, http.StatusOK)
}

func (h *MemoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode memory create request", "error", err)
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		respondError(w, "content is required", http.StatusBadRequest)
		return
	}

	memory, err := h.memorySvc.CreateMemory(r.Context(), req.Content, nil)
	if err != nil {
		slog.Error("failed to create memory", "error", err)
		respondError(w, "failed to create memory", http.StatusInternalServerError)
		return
	}
	slog.Debug("created memory", "id", memory.ID)

	respondJSON(w, memory, http.StatusCreated)
}

func (h *MemoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	memory, err := h.memorySvc.GetMemory(r.Context(), id)
	if err != nil {
		respondError(w, "memory not found", http.StatusNotFound)
		return
	}

	var req struct {
		Content    *string  `json:"content"`
		Importance *float32 `json:"importance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content != nil {
		memory.Content = *req.Content
	}
	if req.Importance != nil {
		memory.Importance = *req.Importance
	}

	if err := h.memorySvc.UpdateMemory(r.Context(), memory); err != nil {
		respondError(w, "failed to update memory", http.StatusInternalServerError)
		return
	}

	respondJSON(w, memory, http.StatusOK)
}

func (h *MemoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var reason *string
	if r.Body != nil && r.ContentLength > 0 {
		var req struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Reason != "" {
			reason = &req.Reason
		}
	}

	if err := h.memorySvc.DeleteMemory(r.Context(), id, reason); err != nil {
		respondError(w, "failed to delete memory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MemoryHandler) Search(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		respondError(w, "query is required", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	memories, err := h.memorySvc.SearchMemories(r.Context(), req.Query, limit)
	if err != nil {
		respondError(w, "search failed", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"memories": memories,
	}, http.StatusOK)
}

func (h *MemoryHandler) GetByTags(w http.ResponseWriter, r *http.Request) {
	tags := r.URL.Query()["tag"]
	if len(tags) == 0 {
		respondError(w, "at least one tag is required", http.StatusBadRequest)
		return
	}

	limit := parseIntQuery(r, "limit", 50)

	memories, err := h.memorySvc.GetMemoriesByTags(r.Context(), tags, limit)
	if err != nil {
		respondError(w, "failed to get memories", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"memories": memories,
	}, http.StatusOK)
}

func (h *MemoryHandler) Pin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.memorySvc.PinMemory(r.Context(), id, req.Pinned); err != nil {
		respondError(w, "failed to pin memory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MemoryHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.memorySvc.ArchiveMemory(r.Context(), id); err != nil {
		respondError(w, "failed to archive memory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MemoryHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.memorySvc.AddTag(r.Context(), id, req.Tag); err != nil {
		respondError(w, "failed to add tag", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MemoryHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tag := chi.URLParam(r, "tag")

	if err := h.memorySvc.RemoveTag(r.Context(), id, tag); err != nil {
		respondError(w, "failed to remove tag", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EnrichedMemoryUse includes the memory content with the memory use record.
type EnrichedMemoryUse struct {
	ID         string  `json:"id"`
	MemoryID   string  `json:"memory_id"`
	MessageID  string  `json:"message_id"`
	Content    string  `json:"content"`
	Similarity float32 `json:"similarity"`
	CreatedAt  string  `json:"created_at"`
}

func (h *MemoryHandler) GetMemoryUsesByMessage(w http.ResponseWriter, r *http.Request) {
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

	uses, err := h.memorySvc.GetUsesByMessage(r.Context(), msgID)
	if err != nil {
		respondError(w, "failed to get memory uses", http.StatusInternalServerError)
		return
	}

	// Enrich memory uses with content
	enriched := make([]EnrichedMemoryUse, 0, len(uses))
	for _, use := range uses {
		mem, err := h.memorySvc.GetMemory(r.Context(), use.MemoryID)
		if err != nil {
			// Skip memories that can't be found (may have been deleted)
			continue
		}
		enriched = append(enriched, EnrichedMemoryUse{
			ID:         use.ID,
			MemoryID:   use.MemoryID,
			MessageID:  use.MessageID,
			Content:    mem.Content,
			Similarity: use.Similarity,
			CreatedAt:  use.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		})
	}

	respondJSON(w, map[string]any{
		"memory_uses": enriched,
	}, http.StatusOK)
}
