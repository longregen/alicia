package dto

import (
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

type SendMessageRequest struct {
	Contents string `json:"contents" msgpack:"contents"`
	LocalID  string `json:"local_id,omitempty" msgpack:"localId,omitempty"`
}

type SwitchBranchRequest struct {
	TipMessageID string `json:"tip_message_id" msgpack:"tipMessageId"`
}

type ToolUseResponse struct {
	ID             string         `json:"id" msgpack:"id"`
	MessageID      string         `json:"message_id" msgpack:"messageId"`
	ToolName       string         `json:"tool_name" msgpack:"toolName"`
	Arguments      map[string]any `json:"arguments,omitempty" msgpack:"arguments,omitempty"`
	Result         any            `json:"result,omitempty" msgpack:"result,omitempty"`
	Status         string         `json:"status" msgpack:"status"`
	ErrorMessage   string         `json:"error_message,omitempty" msgpack:"errorMessage,omitempty"`
	SequenceNumber int            `json:"sequence_number" msgpack:"sequenceNumber"`
	CompletedAt    *string        `json:"completed_at,omitempty" msgpack:"completedAt,omitempty"`
	CreatedAt      string         `json:"created_at" msgpack:"createdAt"`
	UpdatedAt      string         `json:"updated_at" msgpack:"updatedAt"`
}

func FromToolUseModel(tu *models.ToolUse) *ToolUseResponse {
	var completedAt *string
	if tu.CompletedAt != nil {
		formatted := tu.CompletedAt.Format(time.RFC3339)
		completedAt = &formatted
	}
	return &ToolUseResponse{
		ID:             tu.ID,
		MessageID:      tu.MessageID,
		ToolName:       tu.ToolName,
		Arguments:      tu.Arguments,
		Result:         tu.Result,
		Status:         string(tu.Status),
		ErrorMessage:   tu.ErrorMessage,
		SequenceNumber: tu.SequenceNumber,
		CompletedAt:    completedAt,
		CreatedAt:      tu.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      tu.UpdatedAt.Format(time.RFC3339),
	}
}

func FromToolUseModelList(toolUses []*models.ToolUse) []*ToolUseResponse {
	if toolUses == nil {
		return nil
	}
	responses := make([]*ToolUseResponse, len(toolUses))
	for i, tu := range toolUses {
		responses[i] = FromToolUseModel(tu)
	}
	return responses
}

type MemoryUsageResponse struct {
	ID                string  `json:"id" msgpack:"id"`
	ConversationID    string  `json:"conversation_id" msgpack:"conversationId"`
	MessageID         string  `json:"message_id" msgpack:"messageId"`
	MemoryID          string  `json:"memory_id" msgpack:"memoryId"`
	MemoryContent     string  `json:"memory_content,omitempty" msgpack:"memoryContent,omitempty"`
	SimilarityScore   float32 `json:"similarity_score" msgpack:"similarityScore"`
	PositionInResults int     `json:"position_in_results" msgpack:"positionInResults"`
	CreatedAt         string  `json:"created_at" msgpack:"createdAt"`
}

func FromMemoryUsageModel(mu *models.MemoryUsage) *MemoryUsageResponse {
	resp := &MemoryUsageResponse{
		ID:                mu.ID,
		ConversationID:    mu.ConversationID,
		MessageID:         mu.MessageID,
		MemoryID:          mu.MemoryID,
		SimilarityScore:   mu.SimilarityScore,
		PositionInResults: mu.PositionInResults,
		CreatedAt:         mu.CreatedAt.Format(time.RFC3339),
	}
	if mu.Memory != nil {
		resp.MemoryContent = mu.Memory.Content
	}
	return resp
}

func FromMemoryUsageModelList(memoryUsages []*models.MemoryUsage) []*MemoryUsageResponse {
	if memoryUsages == nil {
		return nil
	}
	responses := make([]*MemoryUsageResponse, len(memoryUsages))
	for i, mu := range memoryUsages {
		responses[i] = FromMemoryUsageModel(mu)
	}
	return responses
}

type MessageResponse struct {
	ID             string                 `json:"id" msgpack:"id"`
	ConversationID string                 `json:"conversation_id" msgpack:"conversationId"`
	SequenceNumber int                    `json:"sequence_number" msgpack:"sequenceNumber"`
	PreviousID     string                 `json:"previous_id,omitempty" msgpack:"previousId,omitempty"`
	Role           string                 `json:"role" msgpack:"role"`
	Contents       string                 `json:"contents" msgpack:"contents"`
	LocalID        string                 `json:"local_id,omitempty" msgpack:"localId,omitempty"`
	ServerID       string                 `json:"server_id,omitempty" msgpack:"serverId,omitempty"`
	ToolUses       []*ToolUseResponse     `json:"tool_uses,omitempty" msgpack:"toolUses,omitempty"`
	MemoryUsages   []*MemoryUsageResponse `json:"memory_usages,omitempty" msgpack:"memoryUsages,omitempty"`
	CreatedAt      string                 `json:"created_at" msgpack:"createdAt"`
	UpdatedAt      string                 `json:"updated_at" msgpack:"updatedAt"`
}

type MessageListResponse struct {
	Messages []*MessageResponse `json:"messages" msgpack:"messages"`
	Total    int                `json:"total" msgpack:"total"`
}

func (r *MessageResponse) FromModel(msg *models.Message) *MessageResponse {
	return &MessageResponse{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SequenceNumber: msg.SequenceNumber,
		PreviousID:     msg.PreviousID,
		Role:           string(msg.Role),
		Contents:       msg.Contents,
		LocalID:        msg.LocalID,
		ServerID:       msg.ServerID,
		ToolUses:       FromToolUseModelList(msg.ToolUses),
		MemoryUsages:   FromMemoryUsageModelList(msg.MemoryUsages),
		CreatedAt:      msg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      msg.UpdatedAt.Format(time.RFC3339),
	}
}

func FromMessageModelList(msgs []*models.Message) []*MessageResponse {
	responses := make([]*MessageResponse, len(msgs))
	for i, msg := range msgs {
		responses[i] = (&MessageResponse{}).FromModel(msg)
	}
	return responses
}

type EditAssistantMessageRequest struct {
	Contents string `json:"contents" msgpack:"contents"`
}

type EditUserMessageRequest struct {
	Contents string `json:"contents" msgpack:"contents"`
}

type EditMessageResponse struct {
	UpdatedMessage   *MessageResponse `json:"updated_message" msgpack:"updatedMessage"`
	AssistantMessage *MessageResponse `json:"assistant_message,omitempty" msgpack:"assistantMessage,omitempty"`
	DeletedCount     int              `json:"deleted_count,omitempty" msgpack:"deletedCount,omitempty"`
}

type RegenerateResponse struct {
	DeletedMessageID string           `json:"deleted_message_id" msgpack:"deletedMessageId"`
	NewMessage       *MessageResponse `json:"new_message,omitempty" msgpack:"newMessage,omitempty"`
}

type ContinueResponse struct {
	TargetMessage   *MessageResponse `json:"target_message" msgpack:"targetMessage"`
	AppendedContent string           `json:"appended_content" msgpack:"appendedContent"`
}
