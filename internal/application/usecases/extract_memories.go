package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// ExtractMemoriesInput contains the input for memory extraction
type ExtractMemoriesInput struct {
	// ConversationText is the text to extract memories from
	ConversationText string
	// ConversationContext provides additional context (optional)
	ConversationContext string
	// ConversationID links extracted memories to a conversation (optional)
	ConversationID string
	// MessageID links extracted memories to a specific message (optional)
	MessageID string
	// DuplicateThreshold is the similarity threshold above which a memory is considered duplicate (default: 0.85)
	DuplicateThreshold float32
	// MinImportance is the minimum importance score to store a memory (default: 0.3)
	MinImportance float64
}

// ExtractedMemory represents a single extracted memory with metadata
type ExtractedMemory struct {
	Content    string  `json:"content"`
	Importance float64 `json:"importance"`
	IsDupe     bool    `json:"is_duplicate"`
	DupeOf     string  `json:"duplicate_of,omitempty"` // ID of existing memory if duplicate
}

// ExtractMemoriesOutput contains the result of memory extraction
type ExtractMemoriesOutput struct {
	// ExtractedMemories contains all extracted facts with their metadata
	ExtractedMemories []*ExtractedMemory
	// CreatedMemories contains only the memories that were actually created (non-duplicates)
	CreatedMemories []*models.Memory
	// Reasoning explains why these memories were extracted
	Reasoning string
	// SkippedCount is the number of memories skipped due to duplicates or low importance
	SkippedCount int
}

// ExtractMemories is a use case for extracting and storing memories from conversation text
type ExtractMemories struct {
	memoryService ports.MemoryService
	llmService    ports.LLMService
	idGenerator   ports.IDGenerator
}

// NewExtractMemories creates a new ExtractMemories use case
func NewExtractMemories(
	memoryService ports.MemoryService,
	llmService ports.LLMService,
	idGenerator ports.IDGenerator,
) *ExtractMemories {
	return &ExtractMemories{
		memoryService: memoryService,
		llmService:    llmService,
		idGenerator:   idGenerator,
	}
}

// Execute extracts memories from the given text and stores non-duplicate ones
func (uc *ExtractMemories) Execute(ctx context.Context, input *ExtractMemoriesInput) (*ExtractMemoriesOutput, error) {
	if input.ConversationText == "" {
		return &ExtractMemoriesOutput{}, nil
	}

	// Set defaults
	if input.DuplicateThreshold == 0 {
		input.DuplicateThreshold = 0.85
	}
	if input.MinImportance == 0 {
		input.MinImportance = 0.3
	}

	// Skip if text is too short
	if len(input.ConversationText) < 50 {
		return &ExtractMemoriesOutput{
			Reasoning: "Text too short for meaningful memory extraction",
		}, nil
	}

	// Extract facts using GEPA-optimized prompt
	extractedFacts, importanceScores, reasoning, err := uc.extractFacts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to extract facts: %w", err)
	}

	if len(extractedFacts) == 0 {
		return &ExtractMemoriesOutput{
			Reasoning: reasoning,
		}, nil
	}

	output := &ExtractMemoriesOutput{
		ExtractedMemories: make([]*ExtractedMemory, 0, len(extractedFacts)),
		CreatedMemories:   make([]*models.Memory, 0),
		Reasoning:         reasoning,
	}

	// Process each extracted fact
	for i, fact := range extractedFacts {
		importance := 0.5 // default
		if i < len(importanceScores) {
			importance = importanceScores[i]
		}

		extracted := &ExtractedMemory{
			Content:    fact,
			Importance: importance,
		}

		// Skip if importance is too low
		if importance < input.MinImportance {
			extracted.IsDupe = false
			output.ExtractedMemories = append(output.ExtractedMemories, extracted)
			output.SkippedCount++
			log.Printf("info: skipping memory due to low importance (%.2f): %s\n", importance, truncateForLog(fact, 50))
			continue
		}

		// Check for duplicates by searching existing memories
		isDuplicate, existingMemoryID, err := uc.checkDuplicate(ctx, fact, input.DuplicateThreshold)
		if err != nil {
			log.Printf("warning: failed to check for duplicate memory: %v\n", err)
			// Continue anyway, don't skip the memory
		}

		extracted.IsDupe = isDuplicate
		extracted.DupeOf = existingMemoryID
		output.ExtractedMemories = append(output.ExtractedMemories, extracted)

		if isDuplicate {
			output.SkippedCount++
			log.Printf("info: skipping duplicate memory (similar to %s): %s\n", existingMemoryID, truncateForLog(fact, 50))
			continue
		}

		// Create the memory
		var memory *models.Memory
		if input.ConversationID != "" && input.MessageID != "" {
			memory, err = uc.memoryService.CreateFromConversation(ctx, fact, input.ConversationID, input.MessageID)
		} else {
			memory, err = uc.memoryService.CreateWithEmbeddings(ctx, fact)
		}

		if err != nil {
			log.Printf("warning: failed to create memory '%s': %v\n", truncateForLog(fact, 50), err)
			continue
		}

		// Set importance score
		_, err = uc.memoryService.SetImportance(ctx, memory.ID, float32(importance))
		if err != nil {
			log.Printf("warning: failed to set memory importance: %v\n", err)
		}

		output.CreatedMemories = append(output.CreatedMemories, memory)
		log.Printf("info: created memory (importance=%.2f): %s\n", importance, truncateForLog(fact, 50))
	}

	return output, nil
}

// extractFacts uses the LLM to extract facts from the conversation text
func (uc *ExtractMemories) extractFacts(ctx context.Context, input *ExtractMemoriesInput) ([]string, []float64, string, error) {
	// Build the extraction prompt using GEPA seed prompt
	systemPrompt := baselines.MemoryExtractionSeedPrompt

	userPrompt := fmt.Sprintf("Extract memories from the following conversation:\n\nConversation:\n%s", input.ConversationText)
	if input.ConversationContext != "" {
		userPrompt = fmt.Sprintf("Context: %s\n\n%s", input.ConversationContext, userPrompt)
	}

	messages := []ports.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := uc.llmService.Chat(ctx, messages)
	if err != nil {
		return nil, nil, "", fmt.Errorf("LLM chat failed: %w", err)
	}

	// Parse the structured response
	facts, scores, reasoning := uc.parseExtractionResponse(response.Content)
	return facts, scores, reasoning, nil
}

// parseExtractionResponse parses the LLM response to extract facts and importance scores
func (uc *ExtractMemories) parseExtractionResponse(content string) ([]string, []float64, string) {
	var facts []string
	var scores []float64
	var reasoning string

	// Try to parse as JSON first (structured output)
	type structuredResponse struct {
		ExtractedFacts      json.RawMessage `json:"extracted_facts"`
		ImportanceScores    json.RawMessage `json:"importance_scores"`
		ExtractionReasoning string          `json:"extraction_reasoning"`
	}

	var structured structuredResponse
	if err := json.Unmarshal([]byte(content), &structured); err == nil {
		// Successfully parsed as JSON
		json.Unmarshal(structured.ExtractedFacts, &facts)
		json.Unmarshal(structured.ImportanceScores, &scores)
		reasoning = structured.ExtractionReasoning
		return facts, scores, reasoning
	}

	// Fall back to parsing key-value format
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "extracted_facts:") || strings.HasPrefix(line, "- extracted_facts:") {
			jsonPart := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "extracted_facts:")
			jsonPart = strings.TrimSpace(jsonPart)
			json.Unmarshal([]byte(jsonPart), &facts)
		} else if strings.HasPrefix(line, "importance_scores:") || strings.HasPrefix(line, "- importance_scores:") {
			jsonPart := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "importance_scores:")
			jsonPart = strings.TrimSpace(jsonPart)
			json.Unmarshal([]byte(jsonPart), &scores)
		} else if strings.HasPrefix(line, "extraction_reasoning:") || strings.HasPrefix(line, "- extraction_reasoning:") {
			reasoning = strings.TrimPrefix(strings.TrimPrefix(line, "- "), "extraction_reasoning:")
			reasoning = strings.TrimSpace(reasoning)
		} else if strings.HasPrefix(line, "MEMORY:") {
			// Legacy format support
			fact := strings.TrimSpace(strings.TrimPrefix(line, "MEMORY:"))
			if fact != "" && len(fact) >= 10 {
				facts = append(facts, fact)
				scores = append(scores, 0.5) // default importance
			}
		}
	}

	// Handle NONE response
	if strings.TrimSpace(content) == "NONE" || len(facts) == 0 {
		return nil, nil, "No significant memories to extract from the conversation."
	}

	return facts, scores, reasoning
}

// checkDuplicate checks if a similar memory already exists
func (uc *ExtractMemories) checkDuplicate(ctx context.Context, content string, threshold float32) (bool, string, error) {
	// Search for similar existing memories
	results, err := uc.memoryService.SearchWithScores(ctx, content, threshold, 1)
	if err != nil {
		return false, "", err
	}

	if len(results) > 0 && results[0].Similarity >= threshold {
		return true, results[0].Memory.ID, nil
	}

	return false, "", nil
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
