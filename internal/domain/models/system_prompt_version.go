package models

import "time"

// Prompt type constants
const (
	PromptTypeMain            = "main"
	PromptTypeToolSelection   = "tool_selection"
	PromptTypeMemorySelection = "memory_selection"
)

// SystemPromptVersion tracks versions of system prompts used in conversations.
// This allows tracking which prompt version generated each conversation response,
// useful for A/B testing and prompt iteration tracking.
type SystemPromptVersion struct {
	ID            string     `json:"id"`
	VersionNumber int        `json:"version_number"`
	PromptHash    string     `json:"prompt_hash"`
	PromptContent string     `json:"prompt_content"`
	PromptType    string     `json:"prompt_type"` // "main", "tool_selection", etc.
	Description   string     `json:"description,omitempty"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// NewSystemPromptVersion creates a new SystemPromptVersion with the given parameters
func NewSystemPromptVersion(id, hash, content, promptType, description string) *SystemPromptVersion {
	return &SystemPromptVersion{
		ID:            id,
		PromptHash:    hash,
		PromptContent: content,
		PromptType:    promptType,
		Description:   description,
		Active:        false,
		CreatedAt:     time.Now(),
	}
}
