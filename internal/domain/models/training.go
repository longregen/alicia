package models

import "time"

// TrainingExample represents a GEPA training example derived from a vote
type TrainingExample struct {
	ID           string         `json:"id"`
	TaskType     string         `json:"task_type"` // tool_selection, memory_selection, memory_extraction
	VoteID       *string        `json:"vote_id,omitempty"`
	IsPositive   bool           `json:"is_positive"`
	Inputs       map[string]any `json:"inputs"`
	Outputs      map[string]any `json:"outputs"`
	VoteMetadata *VoteMetadata  `json:"vote_metadata,omitempty"`
	Source       string         `json:"source"` // vote, synthetic
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty"`
}

// VoteMetadata captures vote context for diagnostic feedback in GEPA
type VoteMetadata struct {
	QuickFeedback string `json:"quick_feedback,omitempty"`
	Note          string `json:"note,omitempty"`
	VoteValue     int    `json:"vote_value"` // 1=up, -1=down
}

// SystemPromptVersion tracks versions of system prompts for GEPA optimization
type SystemPromptVersion struct {
	ID            string     `json:"id"`
	VersionNumber int        `json:"version_number"`
	PromptHash    string     `json:"prompt_hash"`
	PromptContent string     `json:"prompt_content"`
	PromptType    string     `json:"prompt_type"` // main, tool_selection, memory_selection, memory_extraction
	Description   string     `json:"description,omitempty"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// Prompt type constants
const (
	PromptTypeMain             = "main"
	PromptTypeToolSelection    = "tool_selection"
	PromptTypeMemorySelection  = "memory_selection"
	PromptTypeMemoryExtraction = "memory_extraction"
)

// Task type constants (match prompt types for GEPA tasks)
const (
	TaskTypeToolSelection    = "tool_selection"
	TaskTypeMemorySelection  = "memory_selection"
	TaskTypeMemoryExtraction = "memory_extraction"
)

// Training example source constants
const (
	SourceVote      = "vote"
	SourceSynthetic = "synthetic"
)

// NewTrainingExample creates a new training example
func NewTrainingExample(id, taskType, source string, isPositive bool, inputs, outputs map[string]any) *TrainingExample {
	return &TrainingExample{
		ID:         id,
		TaskType:   taskType,
		IsPositive: isPositive,
		Inputs:     inputs,
		Outputs:    outputs,
		Source:     source,
		CreatedAt:  time.Now().UTC(),
	}
}

// SetVoteMetadata associates vote metadata with the training example
func (te *TrainingExample) SetVoteMetadata(voteID string, quickFeedback, note string, voteValue int) {
	te.VoteID = &voteID
	te.VoteMetadata = &VoteMetadata{
		QuickFeedback: quickFeedback,
		Note:          note,
		VoteValue:     voteValue,
	}
}

// NewSystemPromptVersion creates a new system prompt version
func NewSystemPromptVersion(id, promptHash, promptContent, promptType, description string) *SystemPromptVersion {
	return &SystemPromptVersion{
		ID:            id,
		PromptHash:    promptHash,
		PromptContent: promptContent,
		PromptType:    promptType,
		Description:   description,
		Active:        false,
		CreatedAt:     time.Now().UTC(),
	}
}

// Activate marks this prompt version as active
func (spv *SystemPromptVersion) Activate() {
	now := time.Now().UTC()
	spv.Active = true
	spv.ActivatedAt = &now
}

// Deactivate marks this prompt version as inactive
func (spv *SystemPromptVersion) Deactivate() {
	now := time.Now().UTC()
	spv.Active = false
	spv.DeactivatedAt = &now
}
