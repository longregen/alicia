package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// toolResultMemorizationPrompt is the system prompt for analyzing tool results for memorization.
const toolResultMemorizationPrompt = `You are a memory analysis assistant. Your task is to determine if a tool result contains information worth memorizing for future reference.

Analyze tool results to identify:
- User-specific information (preferences, settings, account details)
- Factual data the user explicitly requested
- Information that would be useful in future conversations
- Resolved errors or solutions to problems

Do NOT memorize:
- Generic/templated responses
- Transient data (current time, live prices)
- Results that are too large/complex to summarize
- Error or failure messages
- Information only relevant to this specific request

Respond with a JSON object containing your analysis.`

// MemorizeFromToolUseInput contains the input for extracting memories from tool use results
type MemorizeFromToolUseInput struct {
	// ToolUse is the tool use record containing the result
	ToolUse *models.ToolUse
	// UserQuery is the original user query that triggered the tool use
	UserQuery string
	// ConversationID links the memory to a conversation
	ConversationID string
	// MessageID links the memory to the message containing the tool use
	MessageID string
}

// MemorizeFromToolUseOutput contains the result of memory extraction from tool use
type MemorizeFromToolUseOutput struct {
	// ShouldMemorize indicates if the tool result was deemed worth memorizing
	ShouldMemorize bool
	// MemoriesCreated is the number of new memories created
	MemoriesCreated int
	// MemoriesSkipped is the number of memories skipped (duplicates or low importance)
	MemoriesSkipped int
	// Memories contains the created memory objects
	Memories []*models.Memory
	// Reasoning explains why the result was or wasn't memorized
	Reasoning string
}

// MemorizeFromToolUse analyzes tool use results and creates memories for relevant information
type MemorizeFromToolUse struct {
	llmService      ports.LLMService
	memoryService   ports.MemoryService
	extractMemories *ExtractMemories
}

// NewMemorizeFromToolUse creates a new MemorizeFromToolUse use case
func NewMemorizeFromToolUse(
	llmService ports.LLMService,
	memoryService ports.MemoryService,
	extractMemories *ExtractMemories,
) *MemorizeFromToolUse {
	return &MemorizeFromToolUse{
		llmService:      llmService,
		memoryService:   memoryService,
		extractMemories: extractMemories,
	}
}

// Execute analyzes a tool use result and potentially creates memories from it
func (uc *MemorizeFromToolUse) Execute(ctx context.Context, input *MemorizeFromToolUseInput) (*MemorizeFromToolUseOutput, error) {
	if input.ToolUse == nil {
		return &MemorizeFromToolUseOutput{
			Reasoning: "No tool use provided",
		}, nil
	}

	// Skip failed tool uses
	if input.ToolUse.Status != models.ToolStatusSuccess {
		return &MemorizeFromToolUseOutput{
			Reasoning: fmt.Sprintf("Tool use not successful (status: %s)", input.ToolUse.Status),
		}, nil
	}

	// Skip if result is empty
	if input.ToolUse.Result == nil {
		return &MemorizeFromToolUseOutput{
			Reasoning: "Tool result is empty",
		}, nil
	}

	// Format the tool result for analysis
	resultStr := formatToolResult(input.ToolUse.Result)
	if len(resultStr) < 20 {
		return &MemorizeFromToolUseOutput{
			Reasoning: "Tool result too short for meaningful memory extraction",
		}, nil
	}

	// First, analyze if the tool result contains information worth memorizing
	shouldMemorize, analysisReasoning, err := uc.analyzeToolResult(ctx, input.ToolUse, input.UserQuery, resultStr)
	if err != nil {
		log.Printf("warning: failed to analyze tool result for memorization: %v\n", err)
		// Don't fail the whole operation, just skip memorization
		return &MemorizeFromToolUseOutput{
			Reasoning: fmt.Sprintf("Failed to analyze tool result: %v", err),
		}, nil
	}

	if !shouldMemorize {
		return &MemorizeFromToolUseOutput{
			ShouldMemorize: false,
			Reasoning:      analysisReasoning,
		}, nil
	}

	// Build context for memory extraction
	contextText := fmt.Sprintf(
		"Tool: %s\nUser Query: %s\nTool Result:\n%s",
		input.ToolUse.ToolName,
		input.UserQuery,
		resultStr,
	)

	// Extract memories using the dedicated use case
	extractOutput, err := uc.extractMemories.Execute(ctx, &ExtractMemoriesInput{
		ConversationText:    contextText,
		ConversationContext: fmt.Sprintf("This is the result of using the '%s' tool. Extract facts that would be useful to remember for future conversations.", input.ToolUse.ToolName),
		ConversationID:      input.ConversationID,
		MessageID:           input.MessageID,
		DuplicateThreshold:  0.85,
		MinImportance:       0.5, // Higher threshold for tool results - only truly relevant facts
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract memories from tool result: %w", err)
	}

	log.Printf("info: extracted %d memories from tool '%s' result (skipped %d)\n",
		len(extractOutput.CreatedMemories), input.ToolUse.ToolName, extractOutput.SkippedCount)

	return &MemorizeFromToolUseOutput{
		ShouldMemorize:  true,
		MemoriesCreated: len(extractOutput.CreatedMemories),
		MemoriesSkipped: extractOutput.SkippedCount,
		Memories:        extractOutput.CreatedMemories,
		Reasoning:       analysisReasoning + " " + extractOutput.Reasoning,
	}, nil
}

// analyzeToolResult determines if a tool result contains information worth memorizing
func (uc *MemorizeFromToolUse) analyzeToolResult(
	ctx context.Context,
	toolUse *models.ToolUse,
	userQuery string,
	resultStr string,
) (bool, string, error) {
	// Build the analysis prompt
	systemPrompt := toolResultMemorizationPrompt

	userPrompt := fmt.Sprintf(`Analyze this tool result to determine if it contains information worth memorizing for future reference.

Tool Name: %s
User Query: %s
Tool Result:
%s

Respond with a JSON object:
{
  "should_memorize": true/false,
  "reasoning": "explanation of why this should or shouldn't be memorized"
}

Consider memorizing if:
- Contains user-specific information (preferences, settings, account details)
- Contains factual data the user explicitly asked about
- Contains information that would be useful in future conversations
- Contains resolved errors or solutions to problems

Do NOT memorize if:
- Result is generic/templated (e.g., weather for today that will be stale tomorrow)
- Result is transient (e.g., current time, live stock prices)
- Result is too large/complex to summarize meaningfully
- Result is an error or failure message
- Result would only be relevant for this specific request`,
		toolUse.ToolName,
		userQuery,
		truncateForAnalysis(resultStr, 2000),
	)

	messages := []ports.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := uc.llmService.Chat(ctx, messages)
	if err != nil {
		return false, "", fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse the response
	type analysisResponse struct {
		ShouldMemorize bool   `json:"should_memorize"`
		Reasoning      string `json:"reasoning"`
	}

	var analysis analysisResponse
	if err := json.Unmarshal([]byte(response.Content), &analysis); err != nil {
		// Try to extract from non-JSON response
		log.Printf("warning: failed to parse memorization analysis as JSON, defaulting to false: %v\n", err)
		return false, "Failed to parse analysis response", nil
	}

	return analysis.ShouldMemorize, analysis.Reasoning, nil
}

// formatToolResult converts a tool result to a string representation
func formatToolResult(result any) string {
	if result == nil {
		return ""
	}

	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// Try JSON marshaling for complex types
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", result)
		}
		return string(data)
	}
}

// truncateForAnalysis truncates a string for analysis while keeping it meaningful
func truncateForAnalysis(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
