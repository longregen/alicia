package models

import (
	"time"
)

// ReasoningStep represents a step in the assistant's chain-of-thought reasoning
type ReasoningStep struct {
	ID             string     `json:"id"`
	MessageID      string     `json:"message_id"`
	Content        string     `json:"content"`
	SequenceNumber int        `json:"sequence_number"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

func NewReasoningStep(id, messageID, content string, sequence int) *ReasoningStep {
	now := time.Now()
	return &ReasoningStep{
		ID:             id,
		MessageID:      messageID,
		Content:        content,
		SequenceNumber: sequence,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Commentary represents assistant's internal commentary about a conversation
type Commentary struct {
	ID             string         `json:"id"`
	Content        string         `json:"content"`
	ConversationID string         `json:"conversation_id"`
	MessageID      string         `json:"message_id,omitempty"`
	CreatedBy      string         `json:"created_by,omitempty"`
	Meta           map[string]any `json:"meta,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`
}

func NewCommentary(id, conversationID, content string) *Commentary {
	now := time.Now()
	return &Commentary{
		ID:             id,
		ConversationID: conversationID,
		Content:        content,
		Meta:           make(map[string]any),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

