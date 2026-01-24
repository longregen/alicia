package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/livekit"
	"github.com/longregen/alicia/api/services"
)

// LiveKitHandler handles LiveKit token endpoints.
type LiveKitHandler struct {
	convSvc *services.ConversationService
	lkSvc   *livekit.Service
}

// NewLiveKitHandler creates a new LiveKit handler.
func NewLiveKitHandler(convSvc *services.ConversationService, lkSvc *livekit.Service) *LiveKitHandler {
	return &LiveKitHandler{convSvc: convSvc, lkSvc: lkSvc}
}

// GetToken handles POST /conversations/{id}/token
func (h *LiveKitHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	if h.lkSvc == nil {
		respondError(w, "LiveKit not configured", http.StatusServiceUnavailable)
		return
	}

	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")

	// Verify ownership
	_, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}

	var req struct {
		ParticipantName string `json:"participant_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use user ID as participant name if not provided
		req.ParticipantName = userID
	}
	if req.ParticipantName == "" {
		req.ParticipantName = userID
	}

	// Room name is the conversation ID
	roomName := convID

	token, expiresAt, err := h.lkSvc.GenerateToken(roomName, userID, req.ParticipantName)
	if err != nil {
		respondError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"token":      token,
		"expires_at": expiresAt,
		"room_name":  roomName,
	}, http.StatusOK)
}

// CreateRoom handles POST /conversations/{id}/room
func (h *LiveKitHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	if h.lkSvc == nil {
		respondError(w, "LiveKit not configured", http.StatusServiceUnavailable)
		return
	}

	userID := UserIDFromContext(r.Context())
	convID := chi.URLParam(r, "id")

	// Verify ownership
	_, err := h.convSvc.GetByUser(r.Context(), convID, userID)
	if err != nil {
		respondError(w, "conversation not found", http.StatusNotFound)
		return
	}

	room, err := h.lkSvc.CreateRoom(r.Context(), convID)
	if err != nil {
		respondError(w, "failed to create room", http.StatusInternalServerError)
		return
	}

	respondJSON(w, room, http.StatusCreated)
}
