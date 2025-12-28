package dto

import (
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Contents string `json:"contents" msgpack:"contents"`
}

// MessageResponse represents a message in API responses
type MessageResponse struct {
	ID             string `json:"id" msgpack:"id"`
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	SequenceNumber int    `json:"sequence_number" msgpack:"sequenceNumber"`
	PreviousID     string `json:"previous_id,omitempty" msgpack:"previousId,omitempty"`
	Role           string `json:"role" msgpack:"role"`
	Contents       string `json:"contents" msgpack:"contents"`
	CreatedAt      string `json:"created_at" msgpack:"createdAt"`
	UpdatedAt      string `json:"updated_at" msgpack:"updatedAt"`
}

// MessageListResponse represents a list of messages
type MessageListResponse struct {
	Messages []*MessageResponse `json:"messages" msgpack:"messages"`
	Total    int                `json:"total" msgpack:"total"`
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
		CreatedAt:      msg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      msg.UpdatedAt.Format(time.RFC3339),
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
