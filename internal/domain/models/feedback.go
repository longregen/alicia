package models

import (
	"time"
)

// Vote represents a user vote (upvote/downvote) on a message or sentence
type Vote struct {
	ID            string    `json:"id"`
	TargetType    string    `json:"target_type"`              // "message", "tool_use", "memory", or "reasoning"
	TargetID      string    `json:"target_id"`                // ID of the target entity
	MessageID     string    `json:"message_id"`               // Parent message ID
	Value         int       `json:"value"`                    // 1 for upvote, -1 for downvote
	QuickFeedback string    `json:"quick_feedback,omitempty"` // Optional structured feedback category
	Note          string    `json:"note,omitempty"`           // Optional freeform note
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// VoteAggregates represents aggregated vote counts for a target
type VoteAggregates struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Upvotes    int    `json:"upvotes"`
	Downvotes  int    `json:"downvotes"`
	NetScore   int    `json:"net_score"` // upvotes - downvotes
}

// Vote target types
const (
	VoteTargetMessage   = "message"
	VoteTargetSentence  = "sentence"
	VoteTargetToolUse   = "tool_use"
	VoteTargetMemory    = "memory"
	VoteTargetReasoning = "reasoning"
)

// Vote values
const (
	VoteValueUp   = 1
	VoteValueDown = -1
)

func NewVote(id, targetType, targetID, messageID string, value int) *Vote {
	now := time.Now().UTC()
	return &Vote{
		ID:         id,
		TargetType: targetType,
		TargetID:   targetID,
		MessageID:  messageID,
		Value:      value,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewVoteWithFeedback creates a new Vote with optional quick feedback
func NewVoteWithFeedback(id, targetType, targetID, messageID string, value int, quickFeedback, note string) *Vote {
	vote := NewVote(id, targetType, targetID, messageID, value)
	vote.QuickFeedback = quickFeedback
	vote.Note = note
	return vote
}

// Note represents a user note attached to a message
type Note struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Content   string    `json:"content"`
	Category  string    `json:"category"` // "improvement", "correction", "context", "general"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Note category constants
const (
	NoteCategoryImprovement = "improvement"
	NoteCategoryCorrection  = "correction"
	NoteCategoryContext     = "context"
	NoteCategoryGeneral     = "general"
)

func NewNote(id, messageID, content, category string) *Note {
	now := time.Now().UTC()
	if category == "" {
		category = NoteCategoryGeneral
	}
	return &Note{
		ID:        id,
		MessageID: messageID,
		Content:   content,
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (n *Note) UpdateContent(content string) {
	n.Content = content
	n.UpdatedAt = time.Now().UTC()
}

// SessionStats represents statistics for a conversation session
type SessionStats struct {
	ID                string         `json:"id"`
	ConversationID    string         `json:"conversation_id"`
	MessageCount      int            `json:"message_count"`
	UserMessageCount  int            `json:"user_message_count"`
	TotalTokensUsed   int            `json:"total_tokens_used"`
	TotalLatencyMs    int64          `json:"total_latency_ms"`
	AverageLatencyMs  float64        `json:"average_latency_ms"`
	ToolCallCount     int            `json:"tool_call_count"`
	MemoryRetrievals  int            `json:"memory_retrievals"`
	ErrorCount        int            `json:"error_count"`
	SessionDurationMs int64          `json:"session_duration_ms"`
	StartedAt         time.Time      `json:"started_at"`
	LastActivityAt    time.Time      `json:"last_activity_at"`
	Meta              map[string]any `json:"meta,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func NewSessionStats(id, conversationID string) *SessionStats {
	now := time.Now().UTC()
	return &SessionStats{
		ID:             id,
		ConversationID: conversationID,
		Meta:           make(map[string]any),
		StartedAt:      now,
		LastActivityAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *SessionStats) RecordMessage(isUser bool, tokensUsed int, latencyMs int64) {
	s.MessageCount++
	if isUser {
		s.UserMessageCount++
	}
	s.TotalTokensUsed += tokensUsed
	s.TotalLatencyMs += latencyMs
	if s.MessageCount > 0 {
		s.AverageLatencyMs = float64(s.TotalLatencyMs) / float64(s.MessageCount)
	}
	s.LastActivityAt = time.Now().UTC()
	s.UpdatedAt = time.Now().UTC()
}

func (s *SessionStats) RecordToolCall() {
	s.ToolCallCount++
	s.UpdatedAt = time.Now().UTC()
}

func (s *SessionStats) RecordMemoryRetrieval() {
	s.MemoryRetrievals++
	s.UpdatedAt = time.Now().UTC()
}

func (s *SessionStats) RecordError() {
	s.ErrorCount++
	s.UpdatedAt = time.Now().UTC()
}

func (s *SessionStats) CalculateDuration() {
	s.SessionDurationMs = time.Since(s.StartedAt).Milliseconds()
	s.UpdatedAt = time.Now().UTC()
}
