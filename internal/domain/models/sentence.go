package models

import (
	"time"
)

type AudioType string

const (
	AudioTypeInput  AudioType = "input"
	AudioTypeOutput AudioType = "output"
)

type Sentence struct {
	ID               string           `json:"id"`
	MessageID        string           `json:"message_id"`
	SequenceNumber   int              `json:"sequence_number"`
	Text             string           `json:"text"`
	AudioType        AudioType        `json:"audio_type,omitempty"`
	AudioFormat      string           `json:"audio_format,omitempty"`
	DurationMs       int              `json:"duration_ms,omitempty"`
	AudioBytesize    int              `json:"audio_bytesize,omitempty"`
	AudioData        []byte           `json:"audio_data,omitempty"`
	Meta             map[string]any   `json:"meta,omitempty"`
	CompletionStatus CompletionStatus `json:"completion_status,omitempty"` // Tracks streaming state
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	DeletedAt        *time.Time       `json:"deleted_at,omitempty"`
}

func NewSentence(id, messageID string, sequence int, text string) *Sentence {
	now := time.Now()
	return &Sentence{
		ID:               id,
		MessageID:        messageID,
		SequenceNumber:   sequence,
		Text:             text,
		Meta:             make(map[string]any),
		CompletionStatus: CompletionStatusCompleted, // Default to completed
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (s *Sentence) SetAudio(audioType AudioType, format string, data []byte, durationMs int) {
	s.AudioType = audioType
	s.AudioFormat = format
	s.AudioData = data
	s.AudioBytesize = len(data)
	s.DurationMs = durationMs
	s.UpdatedAt = time.Now()
}

// MarkAsStreaming marks the sentence as currently being streamed
func (s *Sentence) MarkAsStreaming() {
	s.CompletionStatus = CompletionStatusStreaming
	s.UpdatedAt = time.Now()
}

// MarkAsCompleted marks the sentence as completed
func (s *Sentence) MarkAsCompleted() {
	s.CompletionStatus = CompletionStatusCompleted
	s.UpdatedAt = time.Now()
}

// MarkAsFailed marks the sentence as failed
func (s *Sentence) MarkAsFailed() {
	s.CompletionStatus = CompletionStatusFailed
	s.UpdatedAt = time.Now()
}

// IsCompleted returns true if the sentence streaming is completed
func (s *Sentence) IsCompleted() bool {
	return s.CompletionStatus == CompletionStatusCompleted
}

// IsStreaming returns true if the sentence is currently being streamed
func (s *Sentence) IsStreaming() bool {
	return s.CompletionStatus == CompletionStatusStreaming
}

// IsFailed returns true if the sentence streaming failed
func (s *Sentence) IsFailed() bool {
	return s.CompletionStatus == CompletionStatusFailed
}
