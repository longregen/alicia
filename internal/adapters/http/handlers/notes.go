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

// CreateNoteRequest represents a note creation request
type CreateNoteRequest struct {
	Content  string `json:"content"`
	Category string `json:"category"` // "improvement", "correction", "context", "general"
}

// UpdateNoteRequest represents a note update request
type UpdateNoteRequest struct {
	Content string `json:"content"`
}

// NoteResponse represents a note
type NoteResponse struct {
	ID         string `json:"id"`
	MessageID  string `json:"message_id,omitempty"`
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"` // "message", "tool_use", "reasoning"
	Content    string `json:"content"`
	Category   string `json:"category"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// NoteListResponse represents a list of notes
type NoteListResponse struct {
	Notes []NoteResponse `json:"notes"`
	Total int            `json:"total"`
}

// inferTargetType infers the target type from the ID prefix
func inferTargetType(id string) string {
	if len(id) < 3 {
		return "message"
	}

	// Extract prefix (everything before first underscore)
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

// toNoteResponse converts a domain Note to a NoteResponse
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

// --- Message Notes ---

// CreateMessageNote handles POST /api/v1/messages/{id}/notes
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

	// Validate required fields
	if req.Content == "" {
		respondError(w, "validation_error", "Note content is required", http.StatusBadRequest)
		return
	}

	// Validate category
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

	// Default category
	category := req.Category
	if category == "" {
		category = models.NoteCategoryGeneral
	}

	// Create note using repository
	noteID := h.idGenerator.GenerateNoteID()
	note := models.NewNote(noteID, messageID, req.Content, category)

	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		respondError(w, "server_error", "Failed to create note", http.StatusInternalServerError)
		return
	}

	response := toNoteResponse(note, "message")
	respondJSON(w, response, http.StatusCreated)
}

// GetMessageNotes handles GET /api/v1/messages/{id}/notes
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

	// Retrieve notes from repository
	notes, err := h.noteRepo.GetByMessage(r.Context(), messageID)
	if err != nil {
		respondError(w, "server_error", "Failed to retrieve notes", http.StatusInternalServerError)
		return
	}

	// Convert to response format
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

// --- Tool Use Notes ---

// CreateToolUseNote handles POST /api/v1/tool-uses/{id}/notes
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

	// Validate category
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

	// Create note using repository (use toolUseID as messageID for now)
	noteID := h.idGenerator.GenerateNoteID()
	note := models.NewNote(noteID, toolUseID, req.Content, category)

	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		respondError(w, "server_error", "Failed to create note", http.StatusInternalServerError)
		return
	}

	response := toNoteResponse(note, "tool_use")
	respondJSON(w, response, http.StatusCreated)
}

// --- Reasoning Notes ---

// CreateReasoningNote handles POST /api/v1/reasoning/{id}/notes
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

	// Validate category
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

	// Create note using repository (use reasoningID as messageID for now)
	noteID := h.idGenerator.GenerateNoteID()
	note := models.NewNote(noteID, reasoningID, req.Content, category)

	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		respondError(w, "server_error", "Failed to create note", http.StatusInternalServerError)
		return
	}

	response := toNoteResponse(note, "reasoning")
	respondJSON(w, response, http.StatusCreated)
}

// --- General Note Operations ---

// UpdateNote handles PUT /api/v1/notes/{id}
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

	// Update note using repository
	if err := h.noteRepo.Update(r.Context(), noteID, req.Content); err != nil {
		respondError(w, "server_error", "Failed to update note", http.StatusInternalServerError)
		return
	}

	// Retrieve updated note to return full details
	note, err := h.noteRepo.GetByID(r.Context(), noteID)
	if err != nil {
		respondError(w, "server_error", "Failed to retrieve updated note", http.StatusInternalServerError)
		return
	}

	// Infer target type from the note's message_id (which contains the target ID)
	targetType := inferTargetType(note.MessageID)
	response := toNoteResponse(note, targetType)
	respondJSON(w, response, http.StatusOK)
}

// DeleteNote handles DELETE /api/v1/notes/{id}
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

	// Delete note using repository
	if err := h.noteRepo.Delete(r.Context(), noteID); err != nil {
		respondError(w, "server_error", "Failed to delete note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
