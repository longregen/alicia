package models

import (
	"time"
)

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
	ConversationStatusDeleted  ConversationStatus = "deleted"
)

type ConversationPreferences struct {
	TTSVoice          string `json:"tts_voice,omitempty"`
	Language          string `json:"language,omitempty"`
	ResponseStyle     string `json:"response_style,omitempty"`
	EnableReasoning   bool   `json:"enable_reasoning,omitempty"`
	EnableMemory      bool   `json:"enable_memory,omitempty"`
	MaxResponseTokens int    `json:"max_response_tokens,omitempty"`
}

// Conversation represents a conversation session with Alicia
type Conversation struct {
	ID              string                   `json:"id"`
	UserID          string                   `json:"user_id"`
	Title           string                   `json:"title"`
	Status          ConversationStatus       `json:"status"`
	LiveKitRoomName string                   `json:"livekit_room_name,omitempty"`
	Preferences     *ConversationPreferences `json:"preferences,omitempty"`

	// Reconnection semantics: track last stanzaId sent by client and server
	LastClientStanzaID int32 `json:"last_client_stanza_id"`
	LastServerStanzaID int32 `json:"last_server_stanza_id"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func NewConversation(id, userID, title string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:     id,
		UserID: userID,
		Title:  title,
		Status: ConversationStatusActive,
		Preferences: &ConversationPreferences{
			EnableMemory:    true,
			EnableReasoning: false,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (c *Conversation) IsActive() bool {
	return c.Status == ConversationStatusActive && c.DeletedAt == nil
}

// Archive transitions the conversation to archived state with validation
func (c *Conversation) Archive() error {
	if err := ValidateTransition(c.Status, ConversationStatusArchived); err != nil {
		return err
	}
	c.Status = ConversationStatusArchived
	c.UpdatedAt = time.Now()
	return nil
}

// Unarchive transitions the conversation back to active state with validation
func (c *Conversation) Unarchive() error {
	if err := ValidateTransition(c.Status, ConversationStatusActive); err != nil {
		return err
	}
	c.Status = ConversationStatusActive
	c.UpdatedAt = time.Now()
	return nil
}

// MarkAsDeleted transitions the conversation to deleted state with validation
func (c *Conversation) MarkAsDeleted() error {
	if err := ValidateTransition(c.Status, ConversationStatusDeleted); err != nil {
		return err
	}
	c.Status = ConversationStatusDeleted
	now := time.Now()
	c.DeletedAt = &now
	c.UpdatedAt = now
	return nil
}

// ChangeStatus transitions the conversation to a new status with validation
// This is a generic method that validates any status transition
func (c *Conversation) ChangeStatus(newStatus ConversationStatus) error {
	if err := ValidateTransition(c.Status, newStatus); err != nil {
		return err
	}
	c.Status = newStatus
	c.UpdatedAt = time.Now()

	// Set DeletedAt if transitioning to deleted
	if newStatus == ConversationStatusDeleted && c.DeletedAt == nil {
		now := time.Now()
		c.DeletedAt = &now
	}

	return nil
}

// CanTransitionTo checks if the conversation can transition to the given status
func (c *Conversation) CanTransitionTo(newStatus ConversationStatus) bool {
	return IsValidTransition(c.Status, newStatus)
}

// SetLiveKitRoom associates a LiveKit room with the conversation
func (c *Conversation) SetLiveKitRoom(roomName string) {
	c.LiveKitRoomName = roomName
	c.UpdatedAt = time.Now()
}

// UpdateLastClientStanzaID updates the last stanzaId received from the client
func (c *Conversation) UpdateLastClientStanzaID(stanzaID int32) {
	if stanzaID > c.LastClientStanzaID {
		c.LastClientStanzaID = stanzaID
		c.UpdatedAt = time.Now()
	}
}

// UpdateLastServerStanzaID updates the last stanzaId sent by the server
func (c *Conversation) UpdateLastServerStanzaID(stanzaID int32) {
	// Server stanzaIDs are negative, so we compare absolute values
	if -stanzaID > -c.LastServerStanzaID {
		c.LastServerStanzaID = stanzaID
		c.UpdatedAt = time.Now()
	}
}
