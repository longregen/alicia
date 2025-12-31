package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// MemoryFilterService implements GEPA-optimized memory filtering
type MemoryFilterService struct {
	llmService      ports.LLMService
	optimizedPrompt string
	mu              sync.RWMutex
}

// NewMemoryFilterService creates a new memory filter service
func NewMemoryFilterService(llmService ports.LLMService) *MemoryFilterService {
	return &MemoryFilterService{
		llmService:      llmService,
		optimizedPrompt: baselines.MemorySelectionSeedPrompt,
	}
}

// FilterMemories applies GEPA filtering to memory candidates
func (s *MemoryFilterService) FilterMemories(
	ctx context.Context,
	userMessage string,
	conversationContext string,
	candidates []*ports.MemoryCandidate,
) (*ports.MemoryFilterResult, error) {
	if len(candidates) == 0 {
		return &ports.MemoryFilterResult{
			SelectedMemories: nil,
			ExcludedMemories: nil,
			Reasoning:        "No memory candidates to filter",
		}, nil
	}

	// Build the candidate memories JSON for the prompt
	candidateData := make([]map[string]any, len(candidates))
	for i, c := range candidates {
		candidateData[i] = map[string]any{
			"id":                c.Memory.ID,
			"content":           c.Memory.Content,
			"similarity_score":  c.SimilarityScore,
			"importance":        c.Importance,
			"days_since_access": c.DaysSinceAccess,
			"tags":              c.Memory.Tags,
		}
	}

	candidatesJSON, err := json.Marshal(candidateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal candidates: %w", err)
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

Candidate Memories:
%s

Select which memories (if any) are genuinely relevant and should be included in the conversation context. Return your response as JSON with:
- "selected_memory_ids": array of memory IDs to include
- "relevance_reasoning": brief explanation of your selection

Be conservative: only include memories that truly help answer this specific message.`,
				userMessage, conversationContext, string(candidatesJSON)),
		},
	}

	response, err := s.llmService.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM filter call failed: %w", err)
	}

	// Parse the response
	selectedIDs, reasoning := s.parseFilterResponse(response.Content)

	// Build result
	selectedIDSet := make(map[string]bool)
	for _, id := range selectedIDs {
		selectedIDSet[id] = true
	}

	result := &ports.MemoryFilterResult{
		Reasoning: reasoning,
	}

	for _, c := range candidates {
		if selectedIDSet[c.Memory.ID] {
			result.SelectedMemories = append(result.SelectedMemories, c.Memory)
		} else {
			result.ExcludedMemories = append(result.ExcludedMemories, c.Memory)
		}
	}

	return result, nil
}

// parseFilterResponse extracts selected IDs and reasoning from LLM response
func (s *MemoryFilterService) parseFilterResponse(response string) ([]string, string) {
	// Try to parse as JSON
	var parsed struct {
		SelectedMemoryIDs  []string `json:"selected_memory_ids"`
		RelevanceReasoning string   `json:"relevance_reasoning"`
	}

	// Find JSON in response (may be wrapped in markdown code blocks)
	jsonStr := response
	if idx := strings.Index(response, "{"); idx != -1 {
		if endIdx := strings.LastIndex(response, "}"); endIdx > idx {
			jsonStr = response[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
		return parsed.SelectedMemoryIDs, parsed.RelevanceReasoning
	}

	// Fallback: try to extract IDs from text
	var ids []string
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "mem_") || strings.Contains(line, "mem_") {
			// Extract mem_* patterns
			parts := strings.Fields(line)
			for _, part := range parts {
				part = strings.Trim(part, `"[],'`)
				if strings.HasPrefix(part, "mem_") {
					ids = append(ids, part)
				}
			}
		}
	}

	return ids, response
}

// SetOptimizedPrompt updates the filter with a GEPA-optimized prompt
func (s *MemoryFilterService) SetOptimizedPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.optimizedPrompt = prompt
}

// GetCurrentPrompt returns the current filtering prompt
func (s *MemoryFilterService) GetCurrentPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.optimizedPrompt
}

// Ensure MemoryFilterService implements the interface
var _ ports.MemoryFilterService = (*MemoryFilterService)(nil)
