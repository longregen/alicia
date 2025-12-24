package handlers

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

const (
	// MaxConversationTitleLength is the maximum allowed length for conversation titles
	MaxConversationTitleLength = 500
)

type ConversationsHandler struct {
	conversationRepo ports.ConversationRepository
	idGen            ports.IDGenerator
}

func NewConversationsHandler(
	conversationRepo ports.ConversationRepository,
	idGen ports.IDGenerator,
) *ConversationsHandler {
	return &ConversationsHandler{
		conversationRepo: conversationRepo,
		idGen:            idGen,
	}
}

func (h *ConversationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024) // 1MB limit
	defer r.Body.Close()
	req, ok := decodeJSON[dto.CreateConversationRequest](r, w)
	if !ok {
		return
	}

	// Trim whitespace from title
	req.Title = strings.TrimSpace(req.Title)

	// Validate title is not empty
	if req.Title == "" {
		respondError(w, "validation_error", "Title is required", http.StatusBadRequest)
		return
	}

	// Validate title length
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
		respondError(w, "internal_error", "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	response := (&dto.ConversationResponse{}).FromModel(conversation)
	respondJSON(w, response, http.StatusCreated)
}

func (h *ConversationsHandler) List(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
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
	// Extract user ID from context
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
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	response := (&dto.ConversationResponse{}).FromModel(conversation)
	respondJSON(w, response, http.StatusOK)
}

func (h *ConversationsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	id, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	if err := h.conversationRepo.DeleteByIDAndUserID(r.Context(), id, userID); err != nil {
		respondError(w, "internal_error", "Failed to delete conversation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
