package models

import (
	"time"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// SyncStatus represents the synchronization state of a message
type SyncStatus string

const (
	// SyncStatusPending indicates the message exists locally but hasn't been synced to server
	SyncStatusPending SyncStatus = "pending"
	// SyncStatusSynced indicates the message has been successfully synced with the server
	SyncStatusSynced SyncStatus = "synced"
	// SyncStatusConflict indicates there's a conflict between local and server versions
	SyncStatusConflict SyncStatus = "conflict"
)

// CompletionStatus represents the completion state of a message during generation/streaming
type CompletionStatus string

const (
	// CompletionStatusPending indicates the message has been created but generation hasn't started
	CompletionStatusPending CompletionStatus = "pending"
	// CompletionStatusStreaming indicates the message is currently being generated/streamed
	CompletionStatusStreaming CompletionStatus = "streaming"
	// CompletionStatusCompleted indicates the message generation is complete
	CompletionStatusCompleted CompletionStatus = "completed"
	// CompletionStatusFailed indicates the message generation failed
	CompletionStatusFailed CompletionStatus = "failed"
)

type Message struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversation_id"`
	SequenceNumber int         `json:"sequence_number"`
	PreviousID     string      `json:"previous_id,omitempty"`
	Role           MessageRole `json:"role"`
	Contents       string      `json:"contents"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"`

	// Offline sync tracking
	LocalID    string     `json:"local_id,omitempty"`    // Client-generated ID before server assignment
	ServerID   string     `json:"server_id,omitempty"`   // Canonical server-assigned ID
	SyncStatus SyncStatus `json:"sync_status,omitempty"` // Sync state: pending, synced, conflict
	SyncedAt   *time.Time `json:"synced_at,omitempty"`   // When the message was last synced

	// Streaming completion tracking
	CompletionStatus CompletionStatus `json:"completion_status,omitempty"` // Tracks streaming state

	// Related entities (loaded separately)
	Sentences      []*Sentence      `json:"sentences,omitempty"`
	Audio          *Audio           `json:"audio,omitempty"`
	ToolUses       []*ToolUse       `json:"tool_uses,omitempty"`
	ReasoningSteps []*ReasoningStep `json:"reasoning_steps,omitempty"`
	MemoryUsages   []*MemoryUsage   `json:"memory_usages,omitempty"`
}

func NewMessage(id, conversationID string, sequence int, role MessageRole, contents string) *Message {
	now := time.Now().UTC() // Always use UTC for consistent timezone handling
	return &Message{
		ID:               id,
		ConversationID:   conversationID,
		SequenceNumber:   sequence,
		Role:             role,
		Contents:         contents,
		CreatedAt:        now,
		UpdatedAt:        now,
		SyncStatus:       SyncStatusSynced,          // Default to synced for server-created messages
		CompletionStatus: CompletionStatusCompleted, // Default to completed for non-streaming messages
	}
}

// NewLocalMessage creates a new message with offline sync tracking
func NewLocalMessage(localID, conversationID string, sequence int, role MessageRole, contents string) *Message {
	now := time.Now().UTC() // Always use UTC for consistent timezone handling
	return &Message{
		ID:             localID, // Use local ID initially
		LocalID:        localID,
		ConversationID: conversationID,
		SequenceNumber: sequence,
		Role:           role,
		Contents:       contents,
		CreatedAt:      now,
		UpdatedAt:      now,
		SyncStatus:     SyncStatusPending, // Pending until synced to server
	}
}

func NewUserMessage(id, conversationID string, sequence int, contents string) *Message {
	return NewMessage(id, conversationID, sequence, MessageRoleUser, contents)
}

func NewAssistantMessage(id, conversationID string, sequence int, contents string) *Message {
	return NewMessage(id, conversationID, sequence, MessageRoleAssistant, contents)
}

func NewSystemMessage(id, conversationID string, sequence int, contents string) *Message {
	return NewMessage(id, conversationID, sequence, MessageRoleSystem, contents)
}

func (m *Message) SetPreviousMessage(previousID string) {
	m.PreviousID = previousID
	m.UpdatedAt = time.Now().UTC()
}

// AppendContent appends content to the message (for streaming)
func (m *Message) AppendContent(content string) {
	m.Contents += content
	m.UpdatedAt = time.Now().UTC()
}

func (m *Message) IsFromUser() bool {
	return m.Role == MessageRoleUser
}

func (m *Message) IsFromAssistant() bool {
	return m.Role == MessageRoleAssistant
}

// MarkAsSynced marks the message as synced with the given server ID
func (m *Message) MarkAsSynced(serverID string) {
	now := time.Now().UTC()
	m.ServerID = serverID
	m.SyncStatus = SyncStatusSynced
	m.SyncedAt = &now
	m.UpdatedAt = now
}

// MarkAsConflict marks the message as having a sync conflict
func (m *Message) MarkAsConflict() {
	m.SyncStatus = SyncStatusConflict
	m.UpdatedAt = time.Now().UTC()
}

// IsPendingSync returns true if the message is pending synchronization
func (m *Message) IsPendingSync() bool {
	return m.SyncStatus == SyncStatusPending
}

// IsSynced returns true if the message has been synced to the server
func (m *Message) IsSynced() bool {
	return m.SyncStatus == SyncStatusSynced
}

// HasConflict returns true if the message has a sync conflict
func (m *Message) HasConflict() bool {
	return m.SyncStatus == SyncStatusConflict
}

// MarkAsStreaming marks the message as currently being streamed
func (m *Message) MarkAsStreaming() {
	m.CompletionStatus = CompletionStatusStreaming
	m.UpdatedAt = time.Now().UTC()
}

// MarkAsCompleted marks the message as completed
func (m *Message) MarkAsCompleted() {
	m.CompletionStatus = CompletionStatusCompleted
	m.UpdatedAt = time.Now().UTC()
}

// MarkAsFailed marks the message as failed
func (m *Message) MarkAsFailed() {
	m.CompletionStatus = CompletionStatusFailed
	m.UpdatedAt = time.Now().UTC()
}

// IsCompleted returns true if the message generation is completed
func (m *Message) IsCompleted() bool {
	return m.CompletionStatus == CompletionStatusCompleted
}

// IsStreaming returns true if the message is currently being streamed
func (m *Message) IsStreaming() bool {
	return m.CompletionStatus == CompletionStatusStreaming
}

// IsFailed returns true if the message generation failed
func (m *Message) IsFailed() bool {
	return m.CompletionStatus == CompletionStatusFailed
}
