package dto

import (
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// SyncMessageRequest represents a message from the client that needs to be synced
type SyncMessageRequest struct {
	LocalID        string `json:"local_id" msgpack:"local_id"`                            // Client-generated ID
	SequenceNumber int    `json:"sequence_number" msgpack:"sequence_number"`              // Client's sequence number
	PreviousID     string `json:"previous_id,omitempty" msgpack:"previous_id,omitempty"`  // ID of previous message
	Role           string `json:"role" msgpack:"role"`                                    // Message role (user, assistant, system)
	Contents       string `json:"contents" msgpack:"contents"`                            // Message content
	CreatedAt      string `json:"created_at" msgpack:"created_at"`                        // Client timestamp (ISO 8601)
	UpdatedAt      string `json:"updated_at,omitempty" msgpack:"updated_at,omitempty"`    // Client timestamp (ISO 8601)
}

// SyncRequest is the request body for syncing messages
type SyncRequest struct {
	Messages []SyncMessageRequest `json:"messages" msgpack:"messages"` // Messages to sync
}

// SyncedMessage represents a message that has been synced to the server
type SyncedMessage struct {
	LocalID  string           `json:"local_id" msgpack:"local_id"`                    // Original client-generated ID
	ServerID string           `json:"server_id" msgpack:"server_id"`                  // Server-assigned ID
	Status   string           `json:"status" msgpack:"status"`                        // Sync status: "synced" or "conflict"
	Message  *MessageResponse `json:"message,omitempty" msgpack:"message,omitempty"`  // Full message details (if synced)
	Conflict *ConflictDetails `json:"conflict,omitempty" msgpack:"conflict,omitempty"` // Conflict details (if conflict)
}

// ConflictDetails provides information about sync conflicts
type ConflictDetails struct {
	Reason        string           `json:"reason" msgpack:"reason"`                                  // Reason for conflict
	ServerMessage *MessageResponse `json:"server_message,omitempty" msgpack:"server_message,omitempty"` // Existing message on server
	Resolution    string           `json:"resolution" msgpack:"resolution"`                          // Conflict resolution strategy
}

// SyncResponse is the response body for sync requests
type SyncResponse struct {
	SyncedMessages []SyncedMessage `json:"synced_messages" msgpack:"synced_messages"` // Results for each message
	SyncedAt       time.Time       `json:"synced_at" msgpack:"synced_at"`             // Server timestamp of sync
}

// SyncStatusResponse provides sync status for a conversation
type SyncStatusResponse struct {
	ConversationID string     `json:"conversation_id" msgpack:"conversation_id"`
	PendingCount   int        `json:"pending_count" msgpack:"pending_count"`                  // Number of pending messages
	SyncedCount    int        `json:"synced_count" msgpack:"synced_count"`                    // Number of synced messages
	ConflictCount  int        `json:"conflict_count" msgpack:"conflict_count"`                // Number of conflicted messages
	LastSyncedAt   *time.Time `json:"last_synced_at,omitempty" msgpack:"last_synced_at,omitempty"` // Last sync timestamp
}

// ToSyncedMessage converts a domain message to a synced message response
func ToSyncedMessage(msg *models.Message) SyncedMessage {
	status := string(msg.SyncStatus)
	if status == "" {
		status = "synced"
	}

	return SyncedMessage{
		LocalID:  msg.LocalID,
		ServerID: msg.ServerID,
		Status:   status,
		Message:  (&MessageResponse{}).FromModel(msg),
	}
}

// ToSyncedMessageWithConflict creates a synced message response with conflict details
func ToSyncedMessageWithConflict(localID string, reason string, serverMsg *models.Message) SyncedMessage {
	var serverMsgResp *MessageResponse
	if serverMsg != nil {
		serverMsgResp = (&MessageResponse{}).FromModel(serverMsg)
	}

	return SyncedMessage{
		LocalID: localID,
		Status:  "conflict",
		Conflict: &ConflictDetails{
			Reason:        reason,
			ServerMessage: serverMsgResp,
			Resolution:    "manual", // Default to manual resolution
		},
	}
}
