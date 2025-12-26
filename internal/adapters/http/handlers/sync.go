package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type SyncHandler struct {
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	idGen            ports.IDGenerator
	broadcaster      *SSEBroadcaster
}

func NewSyncHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	idGen ports.IDGenerator,
	broadcaster *SSEBroadcaster,
) *SyncHandler {
	return &SyncHandler{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		idGen:            idGen,
		broadcaster:      broadcaster,
	}
}

// SyncMessages handles POST /api/v1/conversations/{id}/sync
func (h *SyncHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Limit request body size to prevent memory exhaustion (10MB for batch message sync)
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit
	defer r.Body.Close()
	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	// Verify conversation exists, is active, and belongs to the user
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

	// Parse sync request
	req, ok := decodeJSON[dto.SyncRequest](r, w)
	if !ok {
		return
	}

	// Process each message
	syncedMessages := make([]dto.SyncedMessage, 0, len(req.Messages))

	for _, msgReq := range req.Messages {
		syncedMsg, err := h.processMessage(r, conversationID, msgReq)
		if err != nil {
			// Log error but continue processing other messages
			// In production, you might want more sophisticated error handling
			syncedMessages = append(syncedMessages, dto.ToSyncedMessageWithConflict(
				msgReq.LocalID,
				"Internal error: "+err.Error(),
				nil,
			))
			continue
		}
		syncedMessages = append(syncedMessages, syncedMsg)
	}

	// Return sync response
	response := &dto.SyncResponse{
		SyncedMessages: syncedMessages,
		SyncedAt:       time.Now().UTC(),
	}

	respondJSON(w, response, http.StatusOK)
}

// processMessage processes a single message sync request
func (h *SyncHandler) processMessage(r *http.Request, conversationID string, msgReq dto.SyncMessageRequest) (dto.SyncedMessage, error) {
	ctx := r.Context()

	// Validation
	if msgReq.LocalID == "" {
		return dto.ToSyncedMessageWithConflict(
			msgReq.LocalID,
			"Local ID is required",
			nil,
		), nil
	}

	if msgReq.Role == "" {
		return dto.ToSyncedMessageWithConflict(
			msgReq.LocalID,
			"Message role is required",
			nil,
		), nil
	}

	// Check if message with this local ID already exists
	existingMsg, err := h.messageRepo.GetByLocalID(ctx, msgReq.LocalID)
	if err != nil && err != pgx.ErrNoRows {
		return dto.SyncedMessage{}, err
	}

	// If message already exists, check for conflicts
	if existingMsg != nil {
		// Message was already synced
		if existingMsg.Contents == msgReq.Contents {
			// No conflict - same content
			return dto.ToSyncedMessage(existingMsg), nil
		}

		// Content differs - conflict detected
		existingMsg.MarkAsConflict()
		if err := h.messageRepo.Update(ctx, existingMsg); err != nil {
			return dto.SyncedMessage{}, err
		}

		return dto.ToSyncedMessageWithConflict(
			msgReq.LocalID,
			"Content mismatch with existing message",
			existingMsg,
		), nil
	}

	// Create new message
	serverID := h.idGen.GenerateMessageID()

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, msgReq.CreatedAt)
	if err != nil {
		createdAt = time.Now().UTC()
	}

	updatedAt := createdAt
	if msgReq.UpdatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, msgReq.UpdatedAt); err == nil {
			updatedAt = parsed
		}
	}

	// Create message with sync tracking
	message := &models.Message{
		ID:               serverID,
		ConversationID:   conversationID,
		SequenceNumber:   msgReq.SequenceNumber,
		PreviousID:       msgReq.PreviousID,
		Role:             models.MessageRole(msgReq.Role),
		Contents:         msgReq.Contents,
		LocalID:          msgReq.LocalID,
		ServerID:         serverID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted, // Synced messages are always completed
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	// Mark as synced
	now := time.Now().UTC()
	message.SyncedAt = &now

	// Save to database
	if err := h.messageRepo.Create(ctx, message); err != nil {
		return dto.SyncedMessage{}, err
	}

	// Broadcast message to SSE subscribers (notify other devices)
	if h.broadcaster != nil {
		messageResponse := (&dto.MessageResponse{}).FromModel(message)
		h.broadcaster.BroadcastMessageEvent(conversationID, messageResponse)
	}

	return dto.ToSyncedMessage(message), nil
}

// GetSyncStatus handles GET /api/v1/conversations/{id}/sync/status
func (h *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	// Verify conversation exists and belongs to the user
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

	// Get all messages to calculate status
	messages, err := h.messageRepo.GetByConversation(r.Context(), conversationID)
	if err != nil {
		respondError(w, "internal_error", "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}

	// Count messages by sync status
	var pendingCount, syncedCount, conflictCount int
	var lastSyncedAt *time.Time

	for _, msg := range messages {
		switch msg.SyncStatus {
		case models.SyncStatusPending:
			pendingCount++
		case models.SyncStatusSynced:
			syncedCount++
			if msg.SyncedAt != nil && (lastSyncedAt == nil || msg.SyncedAt.After(*lastSyncedAt)) {
				lastSyncedAt = msg.SyncedAt
			}
		case models.SyncStatusConflict:
			conflictCount++
		}
	}

	response := &dto.SyncStatusResponse{
		ConversationID: conversationID,
		PendingCount:   pendingCount,
		SyncedCount:    syncedCount,
		ConflictCount:  conflictCount,
		LastSyncedAt:   lastSyncedAt,
	}

	respondJSON(w, response, http.StatusOK)
}
