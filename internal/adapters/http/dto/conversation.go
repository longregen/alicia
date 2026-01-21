package dto

import (
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

type CreateConversationRequest struct {
	Title       string                          `json:"title" msgpack:"title"`
	Preferences *models.ConversationPreferences `json:"preferences,omitempty" msgpack:"preferences,omitempty"`
}

type UpdateConversationRequest struct {
	Title       *string                         `json:"title,omitempty" msgpack:"title,omitempty"`
	Preferences *models.ConversationPreferences `json:"preferences,omitempty" msgpack:"preferences,omitempty"`
}

type ConversationResponse struct {
	ID              string                          `json:"id" msgpack:"id"`
	UserID          string                          `json:"user_id" msgpack:"userId"`
	Title           string                          `json:"title" msgpack:"title"`
	Status          string                          `json:"status" msgpack:"status"`
	LiveKitRoomName string                          `json:"livekit_room_name,omitempty" msgpack:"liveKitRoomName,omitempty"`
	Preferences     *models.ConversationPreferences `json:"preferences,omitempty" msgpack:"preferences,omitempty"`
	TipMessageID    *string                         `json:"tip_message_id,omitempty" msgpack:"tipMessageId,omitempty"`
	CreatedAt       time.Time                       `json:"created_at" msgpack:"createdAt"`
	UpdatedAt       time.Time                       `json:"updated_at" msgpack:"updatedAt"`
}

type ConversationListResponse struct {
	Conversations []*ConversationResponse `json:"conversations" msgpack:"conversations"`
	Total         int                     `json:"total" msgpack:"total"`
	Limit         int                     `json:"limit" msgpack:"limit"`
	Offset        int                     `json:"offset" msgpack:"offset"`
}

func (r *ConversationResponse) FromModel(conv *models.Conversation) *ConversationResponse {
	return &ConversationResponse{
		ID:              conv.ID,
		UserID:          conv.UserID,
		Title:           conv.Title,
		Status:          string(conv.Status),
		LiveKitRoomName: conv.LiveKitRoomName,
		Preferences:     conv.Preferences,
		TipMessageID:    conv.TipMessageID,
		CreatedAt:       conv.CreatedAt,
		UpdatedAt:       conv.UpdatedAt,
	}
}

func FromConversationModelList(convs []*models.Conversation) []*ConversationResponse {
	responses := make([]*ConversationResponse, len(convs))
	for i, conv := range convs {
		responses[i] = (&ConversationResponse{}).FromModel(conv)
	}
	return responses
}
