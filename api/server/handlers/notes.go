package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/services"
	"github.com/longregen/alicia/shared/id"
)

type NoteHandler struct {
	noteSvc *services.NoteService
}

func NewNoteHandler(noteSvc *services.NoteService) *NoteHandler {
	return &NoteHandler{noteSvc: noteSvc}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Use client-provided ID or generate one
	noteID := req.ID
	if noteID == "" {
		noteID = id.NewNote()
	}

	note, err := h.noteSvc.Create(r.Context(), noteID, userID, req.Title, req.Content)
	if err != nil {
		respondError(w, "failed to create note", http.StatusInternalServerError)
		return
	}

	respondJSON(w, note, http.StatusCreated)
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	notes, err := h.noteSvc.List(r.Context(), userID)
	if err != nil {
		respondError(w, "failed to list notes", http.StatusInternalServerError)
		return
	}

	if notes == nil {
		notes = []*domain.Note{}
	}

	respondJSON(w, map[string]any{"notes": notes}, http.StatusOK)
}

func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	noteID := chi.URLParam(r, "id")

	note, err := h.noteSvc.Get(r.Context(), noteID)
	if err != nil {
		respondError(w, "failed to get note", http.StatusInternalServerError)
		return
	}
	if note == nil || note.UserID != userID {
		respondError(w, "note not found", http.StatusNotFound)
		return
	}

	respondJSON(w, note, http.StatusOK)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	noteID := chi.URLParam(r, "id")

	note, err := h.noteSvc.Get(r.Context(), noteID)
	if err != nil {
		respondError(w, "failed to get note", http.StatusInternalServerError)
		return
	}
	if note == nil || note.UserID != userID {
		respondError(w, "note not found", http.StatusNotFound)
		return
	}

	var req struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title != nil {
		note.Title = *req.Title
	}
	if req.Content != nil {
		note.Content = *req.Content
	}

	if err := h.noteSvc.Update(r.Context(), note); err != nil {
		respondError(w, "failed to update note", http.StatusInternalServerError)
		return
	}

	respondJSON(w, note, http.StatusOK)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	noteID := chi.URLParam(r, "id")

	note, err := h.noteSvc.Get(r.Context(), noteID)
	if err != nil {
		respondError(w, "failed to get note", http.StatusInternalServerError)
		return
	}
	if note == nil || note.UserID != userID {
		respondError(w, "note not found", http.StatusNotFound)
		return
	}

	if err := h.noteSvc.Delete(r.Context(), noteID); err != nil {
		respondError(w, "failed to delete note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
