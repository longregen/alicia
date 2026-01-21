package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/encoding"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type SyncHandler struct {
	conversationRepo    ports.ConversationRepository
	messageRepo         ports.MessageRepository
	syncMessagesUseCase ports.SyncMessagesUseCase
}

func NewSyncHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	syncMessagesUseCase ports.SyncMessagesUseCase,
) *SyncHandler {
	return &SyncHandler{
		conversationRepo:    conversationRepo,
		messageRepo:         messageRepo,
		syncMessagesUseCase: syncMessagesUseCase,
	}
}

func (h *SyncHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
	defer r.Body.Close()
	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
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

	contentType := r.Header.Get("Content-Type")
	var req *dto.SyncRequest

	if contentType == encoding.ContentTypeMsgpack {
		req, ok = decodeMsgpack[dto.SyncRequest](r, w)
	} else {
		req, ok = decodeJSON[dto.SyncRequest](r, w)
	}

	if !ok {
		return
	}

	output, err := h.syncMessagesUseCase.Execute(r.Context(), &ports.SyncMessagesInput{
		ConversationID: conversationID,
		Messages:       convertToSyncItems(req.Messages),
	})
	if err != nil {
		respondError(w, "internal_error", "Failed to sync messages", http.StatusInternalServerError)
		return
	}

	syncedMessages := make([]dto.SyncedMessage, 0, len(output.Results))
	for _, result := range output.Results {
		if result.Status == "conflict" {
			syncedMessages = append(syncedMessages, dto.ToSyncedMessageWithConflict(
				result.LocalID,
				"Content mismatch with existing message",
				result.Message,
			))
		} else {
			syncedMessages = append(syncedMessages, dto.ToSyncedMessage(result.Message))
		}
	}

	response := &dto.SyncResponse{
		SyncedMessages: syncedMessages,
		SyncedAt:       output.SyncedAt.Format(time.RFC3339),
	}

	acceptType := encoding.NegotiateContentType(r)
	if acceptType == encoding.ContentTypeMsgpack {
		respondMsgpack(w, response, http.StatusOK)
	} else {
		respondJSON(w, response, http.StatusOK)
	}
}

func convertToSyncItems(messages []dto.SyncMessageRequest) []ports.SyncMessageItem {
	items := make([]ports.SyncMessageItem, len(messages))
	for i, msg := range messages {
		createdAt, err := time.Parse(time.RFC3339, msg.CreatedAt)
		if err != nil {
			createdAt = time.Now().UTC()
		}

		updatedAt := createdAt
		if msg.UpdatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, msg.UpdatedAt); err == nil {
				updatedAt = parsed
			}
		}

		items[i] = ports.SyncMessageItem{
			LocalID:        msg.LocalID,
			SequenceNumber: msg.SequenceNumber,
			PreviousID:     msg.PreviousID,
			Role:           msg.Role,
			Contents:       msg.Contents,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}
	}
	return items
}

func (h *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
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

	messages, err := h.messageRepo.GetByConversation(r.Context(), conversationID)
	if err != nil {
		respondError(w, "internal_error", "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}

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

	var lastSyncedAtStr string
	if lastSyncedAt != nil {
		lastSyncedAtStr = lastSyncedAt.Format(time.RFC3339)
	}

	response := &dto.SyncStatusResponse{
		ConversationID: conversationID,
		PendingCount:   pendingCount,
		SyncedCount:    syncedCount,
		ConflictCount:  conflictCount,
		LastSyncedAt:   lastSyncedAtStr,
	}

	respondJSON(w, response, http.StatusOK)
}
