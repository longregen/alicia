package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

const (
	MaxConversationTitleLength = 500
)

type ConversationsHandler struct {
	conversationRepo ports.ConversationRepository
	memoryService    ports.MemoryService
	idGen            ports.IDGenerator
}

func NewConversationsHandler(
	conversationRepo ports.ConversationRepository,
	memoryService ports.MemoryService,
	idGen ports.IDGenerator,
) *ConversationsHandler {
	return &ConversationsHandler{
		conversationRepo: conversationRepo,
		memoryService:    memoryService,
		idGen:            idGen,
	}
}

func (h *ConversationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024)
	defer r.Body.Close()
	req, ok := decodeJSON[dto.CreateConversationRequest](r, w)
	if !ok {
		return
	}

	req.Title = strings.TrimSpace(req.Title)

	if req.Title == "" {
		respondError(w, "validation_error", "Title is required", http.StatusBadRequest)
		return
	}

	if len(req.Title) > MaxConversationTitleLength {
		respondError(w, "validation_error", "Title exceeds maximum length of 500 characters", http.StatusBadRequest)
		return
	}

	id := h.idGen.GenerateConversationID()
	conversation := models.NewConversation(id, userID, req.Title)
	if req.Preferences != nil {
		conversation.Preferences = req.Preferences
	}

	if err := h.conversationRepo.Create(r.Context(), conversation); err != nil {
		log.Printf("Failed to create conversation for user %s: %v", userID, err)
		respondError(w, "internal_error", "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	response := (&dto.ConversationResponse{}).FromModel(conversation)
	respondJSON(w, response, http.StatusCreated)
}

func (h *ConversationsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)
	activeOnly := r.URL.Query().Get("active") == "true"

	var conversations []*models.Conversation
	var err error

	if activeOnly {
		conversations, err = h.conversationRepo.ListActiveByUserID(r.Context(), userID, limit, offset)
	} else {
		conversations, err = h.conversationRepo.ListByUserID(r.Context(), userID, limit, offset)
	}

	if err != nil {
		log.Printf("Failed to list conversations for user %s: %v", userID, err)
		respondError(w, "internal_error", "Failed to list conversations", http.StatusInternalServerError)
		return
	}

	response := &dto.ConversationListResponse{
		Conversations: dto.FromConversationModelList(conversations),
		Total:         len(conversations),
		Limit:         limit,
		Offset:        offset,
	}

	respondJSON(w, response, http.StatusOK)
}

func (h *ConversationsHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	id, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	conversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), id, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Conversation not found or access denied", http.StatusNotFound)
		} else {
			log.Printf("Failed to get conversation %s for user %s: %v", id, userID, err)
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	response := (&dto.ConversationResponse{}).FromModel(conversation)
	respondJSON(w, response, http.StatusOK)
}

func (h *ConversationsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	id, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	if err := h.memoryService.DeleteByConversationID(r.Context(), id); err != nil {
		log.Printf("Failed to delete memories for conversation %s: %v", id, err)
	}

	if err := h.conversationRepo.DeleteByIDAndUserID(r.Context(), id, userID); err != nil {
		log.Printf("Failed to delete conversation %s for user %s: %v", id, userID, err)
		respondError(w, "internal_error", "Failed to delete conversation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ConversationsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024)
	defer r.Body.Close()

	id, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	conversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), id, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Conversation not found or access denied", http.StatusNotFound)
		} else {
			log.Printf("Failed to get conversation %s for patch (user %s): %v", id, userID, err)
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	req, ok := decodeJSON[dto.UpdateConversationRequest](r, w)
	if !ok {
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			respondError(w, "validation_error", "Title cannot be empty", http.StatusBadRequest)
			return
		}
		if len(title) > MaxConversationTitleLength {
			respondError(w, "validation_error", "Title exceeds maximum length of 500 characters", http.StatusBadRequest)
			return
		}
		conversation.Title = title
	}
	if req.Preferences != nil {
		conversation.Preferences = req.Preferences
	}

	conversation.UpdatedAt = time.Now()

	if err := h.conversationRepo.Update(r.Context(), conversation); err != nil {
		log.Printf("Failed to update conversation %s for user %s: %v", id, userID, err)
		respondError(w, "internal_error", "Failed to update conversation", http.StatusInternalServerError)
		return
	}

	response := (&dto.ConversationResponse{}).FromModel(conversation)
	respondJSON(w, response, http.StatusOK)
}
