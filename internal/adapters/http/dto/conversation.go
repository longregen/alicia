package dto

import (
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// CreateConversationRequest represents a request to create a new conversation
type CreateConversationRequest struct {
	Title       string                          `json:"title"`
	Preferences *models.ConversationPreferences `json:"preferences,omitempty"`
}

// UpdateConversationRequest represents a request to update a conversation
type UpdateConversationRequest struct {
	Title       *string                         `json:"title,omitempty"`
	Preferences *models.ConversationPreferences `json:"preferences,omitempty"`
}

// ConversationResponse represents a conversation in API responses
type ConversationResponse struct {
	ID              string                          `json:"id"`
	UserID          string                          `json:"user_id"`
	Title           string                          `json:"title"`
	Status          string                          `json:"status"`
	LiveKitRoomName string                          `json:"livekit_room_name,omitempty"`
	Preferences     *models.ConversationPreferences `json:"preferences,omitempty"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

// ConversationListResponse represents a list of conversations
type ConversationListResponse struct {
	Conversations []*ConversationResponse `json:"conversations"`
	Total         int                     `json:"total"`
	Limit         int                     `json:"limit"`
	Offset        int                     `json:"offset"`
}

// FromModel converts a domain model to a response DTO
func (r *ConversationResponse) FromModel(conv *models.Conversation) *ConversationResponse {
	return &ConversationResponse{
		ID:              conv.ID,
		UserID:          conv.UserID,
		Title:           conv.Title,
		Status:          string(conv.Status),
		LiveKitRoomName: conv.LiveKitRoomName,
		Preferences:     conv.Preferences,
		CreatedAt:       conv.CreatedAt,
		UpdatedAt:       conv.UpdatedAt,
	}
}

// FromModelList converts a list of domain models to response DTOs
func FromConversationModelList(convs []*models.Conversation) []*ConversationResponse {
	responses := make([]*ConversationResponse, len(convs))
	for i, conv := range convs {
		responses[i] = (&ConversationResponse{}).FromModel(conv)
	}
	return responses
}
