package models

import (
	"time"
)

// EmbeddingsInfo contains metadata about how embeddings were generated
type EmbeddingsInfo struct {
	Model      string `json:"model,omitempty"`
	Dimensions int    `json:"dimensions,omitempty"`
	Version    string `json:"version,omitempty"`
}

type SourceInfo struct {
	ConversationID string `json:"conversation_id,omitempty"`
	MessageID      string `json:"message_id,omitempty"`
	DocumentID     string `json:"document_id,omitempty"`
	URL            string `json:"url,omitempty"`
	Title          string `json:"title,omitempty"`
}

// Memory represents a piece of long-term memory for RAG
type Memory struct {
	ID             string          `json:"id"`
	Content        string          `json:"content"`
	Embeddings     []float32       `json:"embeddings,omitempty"`
	EmbeddingsInfo *EmbeddingsInfo `json:"embeddings_info,omitempty"`
	Importance     float32         `json:"importance"`
	Confidence     float32         `json:"confidence"`
	UserRating     *int            `json:"user_rating,omitempty"`
	CreatedBy      string          `json:"created_by,omitempty"`
	SourceType     string          `json:"source_type,omitempty"`
	SourceInfo     *SourceInfo     `json:"source_info,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Pinned         bool            `json:"pinned"`
	Archived       bool            `json:"archived"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty"`
}

// Common source types
const (
	SourceTypeConversation = "conversation"
	SourceTypeManual       = "manual"
)

func NewMemory(id, content string) *Memory {
	now := time.Now()
	return &Memory{
		ID:         id,
		Content:    content,
		Importance: 0.5,
		Confidence: 1.0,
		Tags:       []string{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (m *Memory) SetEmbeddings(embeddings []float32, info *EmbeddingsInfo) {
	m.Embeddings = embeddings
	m.EmbeddingsInfo = info
	m.UpdatedAt = time.Now()
}

// SetImportance sets the importance score (0-1)
func (m *Memory) SetImportance(importance float32) {
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}
	m.Importance = importance
	m.UpdatedAt = time.Now()
}

// SetConfidence sets the confidence score (0-1)
func (m *Memory) SetConfidence(confidence float32) {
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	m.Confidence = confidence
	m.UpdatedAt = time.Now()
}

// SetUserRating sets the user rating (1-5)
func (m *Memory) SetUserRating(rating int) {
	if rating < 1 {
		rating = 1
	}
	if rating > 5 {
		rating = 5
	}
	m.UserRating = &rating
	m.UpdatedAt = time.Now()
}

func (m *Memory) AddTag(tag string) {
	for _, t := range m.Tags {
		if t == tag {
			return
		}
	}
	m.Tags = append(m.Tags, tag)
	m.UpdatedAt = time.Now()
}

func (m *Memory) RemoveTag(tag string) {
	for i, t := range m.Tags {
		if t == tag {
			m.Tags = append(m.Tags[:i], m.Tags[i+1:]...)
			m.UpdatedAt = time.Now()
			return
		}
	}
}

func (m *Memory) HasEmbeddings() bool {
	return len(m.Embeddings) > 0
}

// MemoryUsage tracks when and how a memory was used in a conversation
type MemoryUsage struct {
	ID                string         `json:"id"`
	ConversationID    string         `json:"conversation_id"`
	MessageID         string         `json:"message_id"`
	MemoryID          string         `json:"memory_id"`
	QueryPrompt       string         `json:"query_prompt,omitempty"`
	QueryPromptMeta   map[string]any `json:"query_prompt_meta,omitempty"`
	SimilarityScore   float32        `json:"similarity_score,omitempty"`
	Meta              map[string]any `json:"meta,omitempty"`
	PositionInResults int            `json:"position_in_results,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         *time.Time     `json:"deleted_at,omitempty"`

	// Related memory (loaded separately)
	Memory *Memory `json:"memory,omitempty"`
}

func NewMemoryUsage(id, conversationID, messageID, memoryID string) *MemoryUsage {
	now := time.Now()
	return &MemoryUsage{
		ID:              id,
		ConversationID:  conversationID,
		MessageID:       messageID,
		MemoryID:        memoryID,
		QueryPromptMeta: make(map[string]any),
		Meta:            make(map[string]any),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
