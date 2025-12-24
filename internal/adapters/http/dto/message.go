package dto

import (
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Contents string `json:"contents"`
}

// MessageResponse represents a message in API responses
type MessageResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SequenceNumber int       `json:"sequence_number"`
	PreviousID     string    `json:"previous_id,omitempty"`
	Role           string    `json:"role"`
	Contents       string    `json:"contents"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessageListResponse represents a list of messages
type MessageListResponse struct {
	Messages []*MessageResponse `json:"messages"`
	Total    int                `json:"total"`
}

// FromModel converts a domain model to a response DTO
func (r *MessageResponse) FromModel(msg *models.Message) *MessageResponse {
	return &MessageResponse{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SequenceNumber: msg.SequenceNumber,
		PreviousID:     msg.PreviousID,
		Role:           string(msg.Role),
		Contents:       msg.Contents,
		CreatedAt:      msg.CreatedAt,
		UpdatedAt:      msg.UpdatedAt,
	}
}

// FromModelList converts a list of domain models to response DTOs
func FromMessageModelList(msgs []*models.Message) []*MessageResponse {
	responses := make([]*MessageResponse, len(msgs))
	for i, msg := range msgs {
		responses[i] = (&MessageResponse{}).FromModel(msg)
	}
	return responses
}
