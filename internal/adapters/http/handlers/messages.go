package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type MessagesHandler struct {
	conversationRepo        ports.ConversationRepository
	messageRepo             ports.MessageRepository
	idGen                   ports.IDGenerator
	generateResponseUseCase ports.GenerateResponseUseCase
	broadcaster             *SSEBroadcaster
	wsBroadcaster           *WebSocketBroadcaster
}

func NewMessagesHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	idGen ports.IDGenerator,
	generateResponseUseCase ports.GenerateResponseUseCase,
	broadcaster *SSEBroadcaster,
	wsBroadcaster *WebSocketBroadcaster,
) *MessagesHandler {
	return &MessagesHandler{
		conversationRepo:        conversationRepo,
		messageRepo:             messageRepo,
		idGen:                   idGen,
		generateResponseUseCase: generateResponseUseCase,
		broadcaster:             broadcaster,
		wsBroadcaster:           wsBroadcaster,
	}
}

func (h *MessagesHandler) List(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, "internal_error", "Failed to list messages", http.StatusInternalServerError)
		return
	}

	response := &dto.MessageListResponse{
		Messages: dto.FromMessageModelList(messages),
		Total:    len(messages),
	}

	respondJSON(w, response, http.StatusOK)
}

func (h *MessagesHandler) Send(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Limit request body size to prevent memory exhaustion (10MB for messages with audio)
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit
	defer r.Body.Close()
	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[dto.SendMessageRequest](r, w)
	if !ok {
		return
	}

	if req.Contents == "" {
		respondError(w, "validation_error", "Message contents is required", http.StatusBadRequest)
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

	sequenceNumber, err := h.messageRepo.GetNextSequenceNumber(r.Context(), conversationID)
	if err != nil {
		respondError(w, "internal_error", "Failed to get sequence number", http.StatusInternalServerError)
		return
	}

	id := h.idGen.GenerateMessageID()
	message := models.NewUserMessage(id, conversationID, sequenceNumber, req.Contents)

	messages, err := h.messageRepo.GetLatestByConversation(r.Context(), conversationID, 1)
	if err == nil && len(messages) > 0 {
		message.SetPreviousMessage(messages[0].ID)
	}

	if err := h.messageRepo.Create(r.Context(), message); err != nil {
		respondError(w, "internal_error", "Failed to create message", http.StatusInternalServerError)
		return
	}

	// Broadcast message to SSE and WebSocket subscribers
	messageResponse := (&dto.MessageResponse{}).FromModel(message)
	if h.broadcaster != nil {
		h.broadcaster.BroadcastMessageEvent(conversationID, messageResponse)
	}
	if h.wsBroadcaster != nil {
		h.wsBroadcaster.BroadcastMessage(conversationID, messageResponse)
	}

	// Trigger response generation asynchronously (if use case is available)
	if h.generateResponseUseCase != nil {
		go func() {
			// 5 minute timeout for LLM generation to prevent indefinite hangs
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			input := &ports.GenerateResponseInput{
				ConversationID:   conversationID,
				UserMessageID:    message.ID,
				RelevantMemories: nil, // No memory support in REST API yet
				EnableTools:      true,
				EnableReasoning:  true,
				EnableStreaming:  false, // REST API doesn't support streaming
				PreviousID:       message.ID,
			}

			output, err := h.generateResponseUseCase.Execute(genCtx, input)
			if err != nil {
				log.Printf("Failed to generate response for REST API message %s: %v", message.ID, err)
				return
			}

			if output == nil || output.Message == nil {
				log.Printf("Generate response returned nil output or message for REST API message %s", message.ID)
				return
			}

			log.Printf("Generated response for REST API message %s: %s", message.ID, output.Message.ID)

			// Broadcast AI response to SSE and WebSocket subscribers
			responseMsg := (&dto.MessageResponse{}).FromModel(output.Message)
			if h.broadcaster != nil {
				h.broadcaster.BroadcastMessageEvent(conversationID, responseMsg)
			}
			if h.wsBroadcaster != nil {
				h.wsBroadcaster.BroadcastMessage(conversationID, responseMsg)
			}
		}()
	}

	response := (&dto.MessageResponse{}).FromModel(message)
	respondJSON(w, response, http.StatusCreated)
}
