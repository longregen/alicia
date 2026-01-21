package handlers

import (
	"net/http"
	"strings"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type NoteHandler struct {
	noteRepo    ports.NoteRepository
	idGenerator ports.IDGenerator
}

func NewNoteHandler(noteRepo ports.NoteRepository, idGenerator ports.IDGenerator) *NoteHandler {
	return &NoteHandler{
		noteRepo:    noteRepo,
		idGenerator: idGenerator,
	}
}

type CreateNoteRequest struct {
	Content  string `json:"content"`
	Category string `json:"category"`
}

type UpdateNoteRequest struct {
	Content string `json:"content"`
}

type NoteResponse struct {
	ID         string `json:"id"`
	MessageID  string `json:"message_id,omitempty"`
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"`
	Content    string `json:"content"`
	Category   string `json:"category"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type NoteListResponse struct {
	Notes []NoteResponse `json:"notes"`
	Total int            `json:"total"`
}

func inferTargetType(id string) string {
	if len(id) < 3 {
		return "message"
	}

	parts := strings.SplitN(id, "_", 2)
	if len(parts) < 2 {
		return "message"
	}

	prefix := parts[0]
	switch prefix {
	case "am":
		return "message"
	case "atu":
		return "tool_use"
	case "ar":
		return "reasoning"
	default:
		return "message"
	}
}

func toNoteResponse(note *models.Note, targetType string) *NoteResponse {
	category := note.Category
	if category == "" {
		category = models.NoteCategoryGeneral
	}
	return &NoteResponse{
		ID:         note.ID,
		MessageID:  note.MessageID,
		TargetID:   note.MessageID,
		TargetType: targetType,
		Content:    note.Content,
		Category:   category,
		CreatedAt:  note.CreatedAt.Unix(),
		UpdatedAt:  note.UpdatedAt.Unix(),
	}
}

func (h *NoteHandler) CreateMessageNote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[CreateNoteRequest](r, w)
	if !ok {
		return
	}

	if req.Content == "" {
		respondError(w, "validation_error", "Note content is required", http.StatusBadRequest)
		return
	}

	validCategories := map[string]bool{
		"improvement": true,
		"correction":  true,
		"context":     true,
		"general":     true,
	}

	if req.Category != "" && !validCategories[req.Category] {
		respondError(w, "validation_error", "Invalid note category", http.StatusBadRequest)
		return
	}

	category := req.Category
	if category == "" {
		category = models.NoteCategoryGeneral
	}

	noteID := h.idGenerator.GenerateNoteID()
	note := models.NewNote(noteID, messageID, req.Content, category)

	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		respondError(w, "server_error", "Failed to create note", http.StatusInternalServerError)
		return
	}

	response := toNoteResponse(note, "message")
	respondJSON(w, response, http.StatusCreated)
}

func (h *NoteHandler) GetMessageNotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	notes, err := h.noteRepo.GetByMessage(r.Context(), messageID)
	if err != nil {
		respondError(w, "server_error", "Failed to retrieve notes", http.StatusInternalServerError)
		return
	}

	noteResponses := make([]NoteResponse, len(notes))
	for i, note := range notes {
		noteResponses[i] = *toNoteResponse(note, "message")
	}

	response := &NoteListResponse{
		Notes: noteResponses,
		Total: len(noteResponses),
	}

	respondJSON(w, response, http.StatusOK)
}

func (h *NoteHandler) CreateToolUseNote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	toolUseID, ok := validateURLParam(r, w, "id", "Tool use ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[CreateNoteRequest](r, w)
	if !ok {
		return
	}

	if req.Content == "" {
		respondError(w, "validation_error", "Note content is required", http.StatusBadRequest)
		return
	}

	validCategories := map[string]bool{
		"improvement": true,
		"correction":  true,
		"context":     true,
		"general":     true,
	}

	if req.Category != "" && !validCategories[req.Category] {
		respondError(w, "validation_error", "Invalid note category", http.StatusBadRequest)
		return
	}

	category := req.Category
	if category == "" {
		category = models.NoteCategoryGeneral
	}

	noteID := h.idGenerator.GenerateNoteID()
	note := models.NewNote(noteID, toolUseID, req.Content, category)

	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		respondError(w, "server_error", "Failed to create note", http.StatusInternalServerError)
		return
	}

	response := toNoteResponse(note, "tool_use")
	respondJSON(w, response, http.StatusCreated)
}

func (h *NoteHandler) CreateReasoningNote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	reasoningID, ok := validateURLParam(r, w, "id", "Reasoning ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[CreateNoteRequest](r, w)
	if !ok {
		return
	}

	if req.Content == "" {
		respondError(w, "validation_error", "Note content is required", http.StatusBadRequest)
		return
	}

	validCategories := map[string]bool{
		"improvement": true,
		"correction":  true,
		"context":     true,
		"general":     true,
	}

	if req.Category != "" && !validCategories[req.Category] {
		respondError(w, "validation_error", "Invalid note category", http.StatusBadRequest)
		return
	}

	category := req.Category
	if category == "" {
		category = models.NoteCategoryGeneral
	}

	noteID := h.idGenerator.GenerateNoteID()
	note := models.NewNote(noteID, reasoningID, req.Content, category)

	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		respondError(w, "server_error", "Failed to create note", http.StatusInternalServerError)
		return
	}

	response := toNoteResponse(note, "reasoning")
	respondJSON(w, response, http.StatusCreated)
}

func (h *NoteHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	noteID, ok := validateURLParam(r, w, "id", "Note ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateNoteRequest](r, w)
	if !ok {
		return
	}

	if req.Content == "" {
		respondError(w, "validation_error", "Note content is required", http.StatusBadRequest)
		return
	}

	if err := h.noteRepo.Update(r.Context(), noteID, req.Content); err != nil {
		respondError(w, "server_error", "Failed to update note", http.StatusInternalServerError)
		return
	}

	note, err := h.noteRepo.GetByID(r.Context(), noteID)
	if err != nil {
		respondError(w, "server_error", "Failed to retrieve updated note", http.StatusInternalServerError)
		return
	}

	targetType := inferTargetType(note.MessageID)
	response := toNoteResponse(note, targetType)
	respondJSON(w, response, http.StatusOK)
}

func (h *NoteHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	noteID, ok := validateURLParam(r, w, "id", "Note ID")
	if !ok {
		return
	}

	if err := h.noteRepo.Delete(r.Context(), noteID); err != nil {
		respondError(w, "server_error", "Failed to delete note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
