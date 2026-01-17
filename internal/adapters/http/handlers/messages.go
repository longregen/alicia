package handlers

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

type MessagesHandler struct {
	conversationRepo            ports.ConversationRepository
	messageRepo                 ports.MessageRepository
	toolUseRepo                 ports.ToolUseRepository
	memoryUsageRepo             ports.MemoryUsageRepository
	sendMessageUseCase          ports.SendMessageUseCase
	processUserMessageUseCase   ports.ProcessUserMessageUseCase
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase
	editUserMessageUseCase      ports.EditUserMessageUseCase
	regenerateResponseUseCase   ports.RegenerateResponseUseCase
	continueResponseUseCase     ports.ContinueResponseUseCase
	wsBroadcaster               *WebSocketBroadcaster
	idGen                       ports.IDGenerator
}

func NewMessagesHandler(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	sendMessageUseCase ports.SendMessageUseCase,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase,
	editUserMessageUseCase ports.EditUserMessageUseCase,
	regenerateResponseUseCase ports.RegenerateResponseUseCase,
	continueResponseUseCase ports.ContinueResponseUseCase,
	wsBroadcaster *WebSocketBroadcaster,
	idGen ports.IDGenerator,
) *MessagesHandler {
	return &MessagesHandler{
		conversationRepo:            conversationRepo,
		messageRepo:                 messageRepo,
		toolUseRepo:                 toolUseRepo,
		memoryUsageRepo:             memoryUsageRepo,
		sendMessageUseCase:          sendMessageUseCase,
		processUserMessageUseCase:   processUserMessageUseCase,
		editAssistantMessageUseCase: editAssistantMessageUseCase,
		editUserMessageUseCase:      editUserMessageUseCase,
		regenerateResponseUseCase:   regenerateResponseUseCase,
		continueResponseUseCase:     continueResponseUseCase,
		wsBroadcaster:               wsBroadcaster,
		idGen:                       idGen,
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

	// Verify user has access to the conversation
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

	// Create user message using ProcessUserMessageUseCase
	if h.processUserMessageUseCase != nil {
		processOutput, err := h.processUserMessageUseCase.Execute(r.Context(), &ports.ProcessUserMessageInput{
			ConversationID: conversationID,
			TextContent:    req.Contents,
		})
		if err != nil {
			log.Printf("Failed to process user message for conversation %s: %v", conversationID, err)
			respondError(w, "internal_error", "Failed to process message", http.StatusInternalServerError)
			return
		}

		// Broadcast user message to WebSocket subscribers
		if processOutput.Message != nil && h.wsBroadcaster != nil {
			userMsgResponse := (&dto.MessageResponse{}).FromModel(processOutput.Message)
			h.wsBroadcaster.BroadcastMessage(conversationID, userMsgResponse)
		}

		// Broadcast ResponseGenerationRequest for agent to pick up
		if h.wsBroadcaster != nil && processOutput.Message != nil {
			h.wsBroadcaster.BroadcastResponseGenerationRequest(conversationID, &protocol.ResponseGenerationRequest{
				ID:              h.idGen.GenerateRequestID(),
				MessageID:       processOutput.Message.ID,
				ConversationID:  conversationID,
				RequestType:     "send",
				EnableTools:     true,
				EnableReasoning: true,
				EnableStreaming: false,
				Timestamp:       time.Now().UnixMilli(),
			})
		}

		// Return accepted status with message ID
		respondJSON(w, map[string]string{
			"status":          "accepted",
			"conversation_id": conversationID,
			"message_id":      processOutput.Message.ID,
		}, http.StatusAccepted)
		return
	}

	// Fallback: use sendMessageUseCase if processUserMessageUseCase is not available
	// This maintains backwards compatibility during migration
	if h.sendMessageUseCase != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in send message for conversation %s: %v\n%s", conversationID, r, debug.Stack())
					if h.wsBroadcaster != nil {
						h.wsBroadcaster.BroadcastError(conversationID, "internal_error", "Message processing failed unexpectedly")
					}
				}
			}()

			// 5 minute timeout for LLM generation to prevent indefinite hangs
			genCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			output, err := h.sendMessageUseCase.Execute(genCtx, &ports.SendMessageInput{
				ConversationID:  conversationID,
				TextContent:     req.Contents,
				LocalID:         req.LocalID,
				EnableTools:     true,
				EnableReasoning: true,
				EnableStreaming: false, // REST doesn't support streaming
			})
			if err != nil {
				log.Printf("Failed to send message for REST API conversation %s: %v", conversationID, err)
				if h.wsBroadcaster != nil {
					h.wsBroadcaster.BroadcastError(conversationID, "send_failed", err.Error())
				}
				return
			}

			if output == nil {
				log.Printf("Send message returned nil output for REST API conversation %s", conversationID)
				if h.wsBroadcaster != nil {
					h.wsBroadcaster.BroadcastError(conversationID, "send_failed", "Message processing returned no output")
				}
				return
			}

			// Broadcast user message to WebSocket subscribers
			if output.UserMessage != nil {
				userMsgResponse := (&dto.MessageResponse{}).FromModel(output.UserMessage)
				if h.wsBroadcaster != nil {
					h.wsBroadcaster.BroadcastMessage(conversationID, userMsgResponse)
				}
			}

			// Broadcast assistant response to WebSocket subscribers
			if output.AssistantMessage != nil {
				log.Printf("Generated response for REST API conversation %s: %s", conversationID, output.AssistantMessage.ID)
				assistantMsgResponse := (&dto.MessageResponse{}).FromModel(output.AssistantMessage)
				if h.wsBroadcaster != nil {
					h.wsBroadcaster.BroadcastMessage(conversationID, assistantMsgResponse)
				}
			}
		}()
	}

	// Return accepted status since processing happens asynchronously
	respondJSON(w, map[string]string{
		"status":          "accepted",
		"conversation_id": conversationID,
	}, http.StatusAccepted)
}

// EditAssistantMessage edits an assistant message's content in place
func (h *MessagesHandler) EditAssistantMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[dto.EditAssistantMessageRequest](r, w)
	if !ok {
		return
	}

	if req.Contents == "" {
		respondError(w, "validation_error", "Contents is required", http.StatusBadRequest)
		return
	}

	// Get the message to verify access and role
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

	// Verify it's an assistant message
	if message.Role != models.MessageRoleAssistant {
		respondError(w, "validation_error", "Can only edit assistant messages with this endpoint", http.StatusBadRequest)
		return
	}

	if h.editAssistantMessageUseCase == nil {
		respondError(w, "internal_error", "Edit assistant message use case not available", http.StatusInternalServerError)
		return
	}

	output, err := h.editAssistantMessageUseCase.Execute(r.Context(), &ports.EditAssistantMessageInput{
		ConversationID:  message.ConversationID,
		TargetMessageID: messageID,
		NewContent:      req.Contents,
	})
	if err != nil {
		log.Printf("Failed to edit assistant message %s: %v", messageID, err)
		respondError(w, "internal_error", "Failed to edit message", http.StatusInternalServerError)
		return
	}

	// Broadcast the updated message
	if output.UpdatedMessage != nil && h.wsBroadcaster != nil {
		msgResponse := (&dto.MessageResponse{}).FromModel(output.UpdatedMessage)
		h.wsBroadcaster.BroadcastMessage(message.ConversationID, msgResponse)
	}

	response := &dto.EditMessageResponse{
		UpdatedMessage: (&dto.MessageResponse{}).FromModel(output.UpdatedMessage),
	}

	respondJSON(w, response, http.StatusOK)
}

// EditUserMessage edits a user message and triggers regeneration of the assistant response
func (h *MessagesHandler) EditUserMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[dto.EditUserMessageRequest](r, w)
	if !ok {
		return
	}

	if req.Contents == "" {
		respondError(w, "validation_error", "Contents is required", http.StatusBadRequest)
		return
	}

	// Get the message to verify access and role
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

	// Verify it's a user message
	if message.Role != models.MessageRoleUser {
		respondError(w, "validation_error", "Can only edit user messages with this endpoint", http.StatusBadRequest)
		return
	}

	if h.editUserMessageUseCase == nil {
		respondError(w, "internal_error", "Edit user message use case not available", http.StatusInternalServerError)
		return
	}

	// Execute the edit synchronously (only updates the message, skips generation)
	output, err := h.editUserMessageUseCase.Execute(r.Context(), &ports.EditUserMessageInput{
		ConversationID:  message.ConversationID,
		TargetMessageID: messageID,
		NewContent:      req.Contents,
		EnableTools:     true,
		EnableReasoning: true,
		EnableStreaming: false,
		SkipGeneration:  true, // Agent will handle response generation
	})
	if err != nil {
		log.Printf("Failed to edit user message %s: %v", messageID, err)
		respondError(w, "internal_error", "Failed to edit message", http.StatusInternalServerError)
		return
	}

	// Broadcast the updated user message
	if output.UpdatedMessage != nil && h.wsBroadcaster != nil {
		msgResponse := (&dto.MessageResponse{}).FromModel(output.UpdatedMessage)
		h.wsBroadcaster.BroadcastMessage(message.ConversationID, msgResponse)
	}

	// Broadcast ResponseGenerationRequest for agent to pick up
	if h.wsBroadcaster != nil && h.idGen != nil && output.UpdatedMessage != nil {
		h.wsBroadcaster.BroadcastResponseGenerationRequest(message.ConversationID, &protocol.ResponseGenerationRequest{
			ID:              h.idGen.GenerateRequestID(),
			MessageID:       output.UpdatedMessage.ID,
			ConversationID:  message.ConversationID,
			RequestType:     "edit",
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: false,
			Timestamp:       time.Now().UnixMilli(),
		})
	}

	respondJSON(w, map[string]string{
		"status":          "accepted",
		"conversation_id": message.ConversationID,
		"message_id":      output.UpdatedMessage.ID,
	}, http.StatusAccepted)
}

// Regenerate regenerates an assistant response
func (h *MessagesHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	// Get the message to verify access and role
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

	// Verify it's an assistant message
	if message.Role != models.MessageRoleAssistant {
		respondError(w, "validation_error", "Can only regenerate assistant messages", http.StatusBadRequest)
		return
	}

	// Broadcast ResponseGenerationRequest for agent to pick up
	if h.wsBroadcaster != nil && h.idGen != nil {
		h.wsBroadcaster.BroadcastResponseGenerationRequest(message.ConversationID, &protocol.ResponseGenerationRequest{
			ID:              h.idGen.GenerateRequestID(),
			MessageID:       messageID,
			ConversationID:  message.ConversationID,
			RequestType:     "regenerate",
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: false,
			Timestamp:       time.Now().UnixMilli(),
		})
	}

	respondJSON(w, map[string]string{
		"status":          "accepted",
		"conversation_id": message.ConversationID,
		"message_id":      messageID,
	}, http.StatusAccepted)
}

// Continue continues an assistant response
func (h *MessagesHandler) Continue(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	// Get the message to verify access and role
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

	// Verify it's an assistant message
	if message.Role != models.MessageRoleAssistant {
		respondError(w, "validation_error", "Can only continue assistant messages", http.StatusBadRequest)
		return
	}

	// Broadcast ResponseGenerationRequest for agent to pick up
	if h.wsBroadcaster != nil && h.idGen != nil {
		h.wsBroadcaster.BroadcastResponseGenerationRequest(message.ConversationID, &protocol.ResponseGenerationRequest{
			ID:              h.idGen.GenerateRequestID(),
			MessageID:       messageID,
			ConversationID:  message.ConversationID,
			RequestType:     "continue",
			EnableTools:     true,
			EnableReasoning: true,
			EnableStreaming: false,
			Timestamp:       time.Now().UnixMilli(),
		})
	}

	respondJSON(w, map[string]string{
		"status":          "accepted",
		"conversation_id": message.ConversationID,
		"message_id":      messageID,
	}, http.StatusAccepted)
}
