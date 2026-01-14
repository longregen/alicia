package handlers

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/ports"
)

type TokenHandler struct {
	conversationRepo ports.ConversationRepository
	liveKitService   ports.LiveKitService
}

func NewTokenHandler(
	conversationRepo ports.ConversationRepository,
	liveKitService ports.LiveKitService,
) *TokenHandler {
	return &TokenHandler{
		conversationRepo: conversationRepo,
		liveKitService:   liveKitService,
	}
}

func (h *TokenHandler) Generate(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()
	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[dto.GenerateTokenRequest](r, w)
	if !ok {
		return
	}

	if req.ParticipantID == "" {
		respondError(w, "validation_error", "Participant ID is required", http.StatusBadRequest)
		return
	}

	conversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), conversationID, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Conversation not found or access denied", http.StatusNotFound)
		} else {
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	if !requireActiveConversation(conversation, w) {
		return
	}

	roomName := conversation.LiveKitRoomName
	if roomName == "" {
		roomName = fmt.Sprintf("conv_%s", conversationID)

		// Check if room already exists to avoid race condition
		// This handles the case where multiple requests try to create the same room
		existingRoom, err := h.liveKitService.GetRoom(r.Context(), roomName)
		if err != nil {
			// Room doesn't exist, try to create it
			room, createErr := h.liveKitService.CreateRoom(r.Context(), roomName)
			if createErr != nil {
				// Room creation failed - it might have been created by another request
				// Try to get it one more time before failing
				existingRoom, getErr := h.liveKitService.GetRoom(r.Context(), roomName)
				if getErr != nil {
					// Both create and get failed - return error with details
					respondError(w, "livekit_error", fmt.Sprintf("Failed to create or get LiveKit room: create=%v, get=%v", createErr, getErr), http.StatusInternalServerError)
					return
				}
				// Successfully retrieved room that was created by another request
				roomName = existingRoom.Name
			} else {
				// Room created successfully
				roomName = room.Name
			}
		} else {
			// Room already exists, use it
			roomName = existingRoom.Name
		}

		// Update conversation with room name
		conversation.SetLiveKitRoom(roomName)
		if err := h.conversationRepo.Update(r.Context(), conversation); err != nil {
			respondError(w, "internal_error", "Failed to update conversation", http.StatusInternalServerError)
			return
		}
	}

	participantName := req.ParticipantName
	if participantName == "" {
		participantName = req.ParticipantID
	}

	token, err := h.liveKitService.GenerateToken(r.Context(), roomName, req.ParticipantID, participantName)
	if err != nil {
		respondError(w, "livekit_error", "Failed to generate LiveKit token", http.StatusInternalServerError)
		return
	}

	response := &dto.GenerateTokenResponse{
		Token:         token.Token,
		ExpiresAt:     token.ExpiresAt,
		RoomName:      roomName,
		ParticipantID: req.ParticipantID,
	}

	respondJSON(w, response, http.StatusOK)
}
