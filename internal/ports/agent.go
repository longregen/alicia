package ports

import (
	"context"

	"github.com/longregen/alicia/internal/domain/models"
)

// AgentInput represents input to the agent for response generation
type AgentInput struct {
	ConversationID  string
	UserMessageID   string
	MessageID       string // Pre-generated message ID for the response
	PreviousID      string // Previous message ID for branching
	EnableTools     bool
	EnableStreaming bool
	EnableReasoning bool
}

// AgentOutput represents the agent's response
type AgentOutput struct {
	Message        *models.Message
	Sentences      []*models.Sentence
	ToolUses       []*models.ToolUse
	ReasoningSteps []*models.ReasoningStep
	StreamChannel  <-chan *ResponseStreamChunk
	SelectedTools  []string         // Tools selected by GEPA
	UsedMemories   []*models.Memory // Memories selected by GEPA filter
}

// MemoryCandidate represents a memory candidate for GEPA filtering
type MemoryCandidate struct {
	Memory          *models.Memory
	SimilarityScore float32
	Importance      float32
	DaysSinceAccess int
}

// MemoryFilterResult represents the result of GEPA memory filtering
type MemoryFilterResult struct {
	SelectedMemories []*models.Memory
	ExcludedMemories []*models.Memory
	Reasoning        string
}

// ToolSelectionResult represents the result of GEPA tool selection
type ToolSelectionResult struct {
	SelectedTool string
	Arguments    map[string]any
	Reasoning    string
	Confidence   float32
}

// MemoryFilterService filters memory candidates using GEPA-optimized prompts
type MemoryFilterService interface {
	// FilterMemories applies GEPA filtering to memory candidates
	// Returns only the memories that are genuinely relevant to the user message
	FilterMemories(
		ctx context.Context,
		userMessage string,
		conversationContext string,
		candidates []*MemoryCandidate,
	) (*MemoryFilterResult, error)

	// SetOptimizedPrompt updates the filter with a GEPA-optimized prompt
	SetOptimizedPrompt(prompt string)

	// GetCurrentPrompt returns the current filtering prompt
	GetCurrentPrompt() string
}

// ToolSelectionService selects tools using GEPA-optimized prompts
type ToolSelectionService interface {
	// SelectTool determines which tool (if any) to use for the user message
	SelectTool(
		ctx context.Context,
		userMessage string,
		conversationContext string,
		availableTools []*models.Tool,
	) (*ToolSelectionResult, error)

	// SetOptimizedPrompt updates the selector with a GEPA-optimized prompt
	SetOptimizedPrompt(prompt string)

	// GetCurrentPrompt returns the current selection prompt
	GetCurrentPrompt() string
}

// AgentService orchestrates the GEPA-optimized agent flow
type AgentService interface {
	// GenerateResponse generates a response using the GEPA-optimized flow:
	// 1. Retrieve memory candidates via RAG
	// 2. Filter memories using GEPA
	// 3. Select tools using GEPA
	// 4. Generate response with filtered context
	GenerateResponse(ctx context.Context, input *AgentInput) (*AgentOutput, error)

	// GetMemoryFilterService returns the memory filter for configuration
	GetMemoryFilterService() MemoryFilterService

	// GetToolSelectionService returns the tool selector for configuration
	GetToolSelectionService() ToolSelectionService
}
