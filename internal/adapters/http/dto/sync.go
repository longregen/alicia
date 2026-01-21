package dto

import (
	"github.com/longregen/alicia/internal/domain/models"
)

type SyncMessageRequest struct {
	LocalID        string `json:"local_id" msgpack:"localId"`
	SequenceNumber int    `json:"sequence_number" msgpack:"sequenceNumber"`
	PreviousID     string `json:"previous_id,omitempty" msgpack:"previousId,omitempty"`
	Role           string `json:"role" msgpack:"role"`
	Contents       string `json:"contents" msgpack:"contents"`
	CreatedAt      string `json:"created_at" msgpack:"createdAt"`
	UpdatedAt      string `json:"updated_at,omitempty" msgpack:"updatedAt,omitempty"`
}

type SyncRequest struct {
	Messages []SyncMessageRequest `json:"messages" msgpack:"messages"`
}

type SyncedMessage struct {
	LocalID  string           `json:"local_id" msgpack:"localId"`
	ServerID string           `json:"server_id" msgpack:"serverId"`
	Status   string           `json:"status" msgpack:"status"`
	Message  *MessageResponse `json:"message,omitempty" msgpack:"message,omitempty"`
	Conflict *ConflictDetails `json:"conflict,omitempty" msgpack:"conflict,omitempty"`
}

type ConflictDetails struct {
	Reason        string           `json:"reason" msgpack:"reason"`
	ServerMessage *MessageResponse `json:"server_message,omitempty" msgpack:"serverMessage,omitempty"`
	Resolution    string           `json:"resolution" msgpack:"resolution"`
}

type SyncResponse struct {
	SyncedMessages []SyncedMessage `json:"synced_messages" msgpack:"syncedMessages"`
	SyncedAt       string          `json:"synced_at" msgpack:"syncedAt"`
}

type SyncStatusResponse struct {
	ConversationID string `json:"conversation_id" msgpack:"conversationId"`
	PendingCount   int    `json:"pending_count" msgpack:"pendingCount"`
	SyncedCount    int    `json:"synced_count" msgpack:"syncedCount"`
	ConflictCount  int    `json:"conflict_count" msgpack:"conflictCount"`
	LastSyncedAt   string `json:"last_synced_at,omitempty" msgpack:"lastSyncedAt,omitempty"`
}

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
			Resolution:    "manual",
		},
	}
}
