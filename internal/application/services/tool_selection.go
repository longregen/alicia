package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// ToolSelectionService implements GEPA-optimized tool selection
type ToolSelectionService struct {
	llmService      ports.LLMService
	optimizedPrompt string
	mu              sync.RWMutex
}

// NewToolSelectionService creates a new tool selection service
func NewToolSelectionService(llmService ports.LLMService) *ToolSelectionService {
	return &ToolSelectionService{
		llmService:      llmService,
		optimizedPrompt: baselines.ToolSelectionSeedPrompt,
	}
}

// SelectTool determines which tool (if any) to use for the user message
func (s *ToolSelectionService) SelectTool(
	ctx context.Context,
	userMessage string,
	conversationContext string,
	availableTools []*models.Tool,
) (*ports.ToolSelectionResult, error) {
	if len(availableTools) == 0 {
		return &ports.ToolSelectionResult{
			SelectedTool: "none",
			Reasoning:    "No tools available",
			Confidence:   1.0,
		}, nil
	}

	// Build tool descriptions for the prompt
	toolData := make([]map[string]any, len(availableTools))
	for i, t := range availableTools {
		toolData[i] = map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Schema,
		}
	}

	toolsJSON, err := json.Marshal(toolData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools: %w", err)
	}

	// Build the LLM prompt
	s.mu.RLock()
	systemPrompt := s.optimizedPrompt
	s.mu.RUnlock()

	messages := []ports.LLMMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`User Message: %s

Conversation Context: %s

Available Tools:
%s

Determine which tool (if any) should be used to respond to this message. Return your response as JSON with:
- "selected_tool": the tool name, or "none" if no tool is needed
- "arguments": JSON object with tool parameters (empty if no tool selected)
- "reasoning": brief explanation of your decision

Be conservative: only select a tool if the user's intent clearly requires it.`,
				userMessage, conversationContext, string(toolsJSON)),
		},
	}

	response, err := s.llmService.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM tool selection call failed: %w", err)
	}

	// Parse the response
	return s.parseSelectionResponse(response.Content), nil
}

// parseSelectionResponse extracts tool selection from LLM response
func (s *ToolSelectionService) parseSelectionResponse(response string) *ports.ToolSelectionResult {
	result := &ports.ToolSelectionResult{
		SelectedTool: "none",
		Arguments:    make(map[string]any),
		Confidence:   0.5,
	}

	// Try to parse as JSON
	var parsed struct {
		SelectedTool string         `json:"selected_tool"`
		Arguments    map[string]any `json:"arguments"`
		Reasoning    string         `json:"reasoning"`
	}

	// Find JSON in response (may be wrapped in markdown code blocks)
	jsonStr := response
	if idx := strings.Index(response, "{"); idx != -1 {
		if endIdx := strings.LastIndex(response, "}"); endIdx > idx {
			jsonStr = response[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
		result.SelectedTool = parsed.SelectedTool
		result.Arguments = parsed.Arguments
		result.Reasoning = parsed.Reasoning
		result.Confidence = 0.9 // High confidence for parsed responses
		return result
	}

	// Fallback: try to extract from text
	lowerResponse := strings.ToLower(response)
	if strings.Contains(lowerResponse, "none") ||
		strings.Contains(lowerResponse, "no tool") ||
		strings.Contains(lowerResponse, "don't need") {
		result.SelectedTool = "none"
		result.Reasoning = response
		result.Confidence = 0.7
	}

	return result
}

// SetOptimizedPrompt updates the selector with a GEPA-optimized prompt
func (s *ToolSelectionService) SetOptimizedPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.optimizedPrompt = prompt
}

// GetCurrentPrompt returns the current selection prompt
func (s *ToolSelectionService) GetCurrentPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.optimizedPrompt
}

// Ensure ToolSelectionService implements the interface
var _ ports.ToolSelectionService = (*ToolSelectionService)(nil)
