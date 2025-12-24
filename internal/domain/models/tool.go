package models

import (
	"time"
)

type ToolStatus string

const (
	ToolStatusPending   ToolStatus = "pending"
	ToolStatusRunning   ToolStatus = "running"
	ToolStatusSuccess   ToolStatus = "success"
	ToolStatusError     ToolStatus = "error"
	ToolStatusCancelled ToolStatus = "cancelled"
)

// Tool represents a registered tool that Alicia can use
type Tool struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty"`
}

func NewTool(id, name, description string, schema map[string]any) *Tool {
	now := time.Now()
	return &Tool{
		ID:          id,
		Name:        name,
		Description: description,
		Schema:      schema,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (t *Tool) Disable() {
	t.Enabled = false
	t.UpdatedAt = time.Now()
}

func (t *Tool) Enable() {
	t.Enabled = true
	t.UpdatedAt = time.Now()
}

type ToolUse struct {
	ID             string         `json:"id"`
	MessageID      string         `json:"message_id"`
	ToolName       string         `json:"tool_name"`
	Arguments      map[string]any `json:"arguments,omitempty"`
	Result         any            `json:"result,omitempty"`
	Status         ToolStatus     `json:"status"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	SequenceNumber int            `json:"sequence_number"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`

	// Related tool (loaded separately)
	Tool *Tool `json:"tool,omitempty"`
}

func NewToolUse(id, messageID, toolName string, sequence int, arguments map[string]any) *ToolUse {
	now := time.Now()
	return &ToolUse{
		ID:             id,
		MessageID:      messageID,
		ToolName:       toolName,
		Arguments:      arguments,
		Status:         ToolStatusPending,
		SequenceNumber: sequence,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (tu *ToolUse) Start() {
	tu.Status = ToolStatusRunning
	tu.UpdatedAt = time.Now()
}

func (tu *ToolUse) Complete(result any) {
	tu.Status = ToolStatusSuccess
	tu.Result = result
	now := time.Now()
	tu.CompletedAt = &now
	tu.UpdatedAt = now
}

func (tu *ToolUse) Fail(errorMessage string) {
	tu.Status = ToolStatusError
	tu.ErrorMessage = errorMessage
	now := time.Now()
	tu.CompletedAt = &now
	tu.UpdatedAt = now
}

func (tu *ToolUse) Cancel() {
	tu.Status = ToolStatusCancelled
	now := time.Now()
	tu.CompletedAt = &now
	tu.UpdatedAt = now
}

// IsComplete returns true if the tool use has finished (success, error, or cancelled)
func (tu *ToolUse) IsComplete() bool {
	return tu.Status == ToolStatusSuccess || tu.Status == ToolStatusError || tu.Status == ToolStatusCancelled
}

func (tu *ToolUse) IsPending() bool {
	return tu.Status == ToolStatusPending
}

func (tu *ToolUse) IsRunning() bool {
	return tu.Status == ToolStatusRunning
}
