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
	toolUseRepo             ports.ToolUseRepository
	memoryUsageRepo         ports.MemoryUsageRepository
	idGen                   ports.IDGenerator
	generateResponseUseCase ports.GenerateResponseUseCase
	wsBroadcaster           *WebSocketBroadcaster
}

func NewMessagesHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	idGen ports.IDGenerator,
	generateResponseUseCase ports.GenerateResponseUseCase,
	wsBroadcaster *WebSocketBroadcaster,
) *MessagesHandler {
	return &MessagesHandler{
		conversationRepo:        conversationRepo,
		messageRepo:             messageRepo,
		toolUseRepo:             toolUseRepo,
		memoryUsageRepo:         memoryUsageRepo,
		idGen:                   idGen,
		generateResponseUseCase: generateResponseUseCase,
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
			log.Printf("Failed to retrieve conversation %s for listing messages: %v", conversationID, err)
			respondError(w, "internal_error", "Failed to retrieve conversation", http.StatusInternalServerError)
		}
		return
	}

	if !requireActiveConversation(conversation, w) {
		return
	}

	var messages []*models.Message

	// If conversation has a tip, get the chain from the tip
	// Otherwise fall back to getting all messages (for backwards compatibility)
	if conversation.TipMessageID != nil && *conversation.TipMessageID != "" {
		messages, err = h.messageRepo.GetChainFromTip(r.Context(), *conversation.TipMessageID)
	} else {
		messages, err = h.messageRepo.GetByConversation(r.Context(), conversationID)
	}

	if err != nil {
		log.Printf("Failed to list messages for conversation %s: %v", conversationID, err)
		respondError(w, "internal_error", "Failed to list messages", http.StatusInternalServerError)
		return
	}

	// Load tool uses for each message
	if h.toolUseRepo != nil {
		for _, msg := range messages {
			toolUses, err := h.toolUseRepo.GetByMessage(r.Context(), msg.ID)
			if err != nil {
				// Log error but don't fail the request - tool uses are supplementary
				log.Printf("Warning: Failed to load tool uses for message %s: %v", msg.ID, err)
				continue
			}
			msg.ToolUses = toolUses
		}
	}

	// Load memory usages for each message
	if h.memoryUsageRepo != nil {
		for _, msg := range messages {
			memoryUsages, err := h.memoryUsageRepo.GetByMessage(r.Context(), msg.ID)
			if err != nil {
				// Log error but don't fail the request - memory usages are supplementary
				log.Printf("Warning: Failed to load memory usages for message %s: %v", msg.ID, err)
				continue
			}
			msg.MemoryUsages = memoryUsages
		}
	}

	response := &dto.MessageListResponse{
		Messages: dto.FromMessageModelList(messages),
		Total:    len(messages),
	}

	respondJSON(w, response, http.StatusOK)
}

// GetSiblings returns all sibling messages (messages that branch from the same parent)
func (h *MessagesHandler) GetSiblings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	// Get the message to verify access
	message, err := h.messageRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Message not found", http.StatusNotFound)
		} else {
			respondError(w, "internal_error", "Failed to retrieve message", http.StatusInternalServerError)
		}
		return
	}

	// Verify user has access to the conversation
	conversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), message.ConversationID, userID)
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

	siblings, err := h.messageRepo.GetSiblings(r.Context(), messageID)
	if err != nil {
		respondError(w, "internal_error", "Failed to retrieve siblings", http.StatusInternalServerError)
		return
	}

	response := &dto.MessageListResponse{
		Messages: dto.FromMessageModelList(siblings),
		Total:    len(siblings),
	}

	respondJSON(w, response, http.StatusOK)
}

// SwitchBranch updates the conversation's tip to point to a different message
func (h *MessagesHandler) SwitchBranch(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	conversationID, ok := validateURLParam(r, w, "id", "Conversation ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[dto.SwitchBranchRequest](r, w)
	if !ok {
		return
	}

	if req.TipMessageID == "" {
		respondError(w, "validation_error", "Tip message ID is required", http.StatusBadRequest)
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

	// Verify the message exists and belongs to this conversation
	message, err := h.messageRepo.GetByID(r.Context(), req.TipMessageID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, "not_found", "Message not found", http.StatusNotFound)
		} else {
			respondError(w, "internal_error", "Failed to retrieve message", http.StatusInternalServerError)
		}
		return
	}

	if message.ConversationID != conversationID {
		respondError(w, "validation_error", "Message does not belong to this conversation", http.StatusBadRequest)
		return
	}

	// Update the tip
	if err := h.conversationRepo.UpdateTip(r.Context(), conversationID, req.TipMessageID); err != nil {
		respondError(w, "internal_error", "Failed to update conversation tip", http.StatusInternalServerError)
		return
	}

	// Return the updated conversation
	updatedConversation, err := h.conversationRepo.GetByIDAndUserID(r.Context(), conversationID, userID)
	if err != nil {
		respondError(w, "internal_error", "Failed to retrieve updated conversation", http.StatusInternalServerError)
		return
	}

	respondJSON(w, updatedConversation, http.StatusOK)
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

	// Preserve client's local_id for deduplication and sync
	if req.LocalID != "" {
		message.LocalID = req.LocalID
		message.ServerID = id
	}

	// Set previous_id to the current conversation tip for message branching
	if conversation.TipMessageID != nil && *conversation.TipMessageID != "" {
		message.SetPreviousMessage(*conversation.TipMessageID)
	}

	if err := h.messageRepo.Create(r.Context(), message); err != nil {
		respondError(w, "internal_error", "Failed to create message", http.StatusInternalServerError)
		return
	}

	// Update conversation tip to point to the new message
	if err := h.conversationRepo.UpdateTip(r.Context(), conversationID, message.ID); err != nil {
		respondError(w, "internal_error", "Failed to update conversation tip", http.StatusInternalServerError)
		return
	}

	// Broadcast message to WebSocket subscribers
	messageResponse := (&dto.MessageResponse{}).FromModel(message)
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
				// Broadcast error to WebSocket subscribers
				if h.wsBroadcaster != nil {
					h.wsBroadcaster.BroadcastError(conversationID, "generation_failed", err.Error())
				}
				return
			}

			if output == nil || output.Message == nil {
				log.Printf("Generate response returned nil output or message for REST API message %s", message.ID)
				// Broadcast error to WebSocket subscribers
				if h.wsBroadcaster != nil {
					h.wsBroadcaster.BroadcastError(conversationID, "generation_failed", "AI response generation returned no output")
				}
				return
			}

			log.Printf("Generated response for REST API message %s: %s", message.ID, output.Message.ID)

			// Broadcast AI response to WebSocket subscribers
			responseMsg := (&dto.MessageResponse{}).FromModel(output.Message)
			if h.wsBroadcaster != nil {
				h.wsBroadcaster.BroadcastMessage(conversationID, responseMsg)
			}
		}()
	}

	response := (&dto.MessageResponse{}).FromModel(message)
	respondJSON(w, response, http.StatusCreated)
}
