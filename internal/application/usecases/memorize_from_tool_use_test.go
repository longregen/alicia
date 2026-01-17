package usecases

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
)

func TestMemorizeFromToolUse_NilToolUse(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse: nil,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false for nil tool use")
	}

	if output.Reasoning != "No tool use provided" {
		t.Errorf("unexpected reasoning: %s", output.Reasoning)
	}
}

func TestMemorizeFromToolUse_SkipsFailedToolUse(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "web_search", 1, nil)
	toolUse.Fail("connection timeout")

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse: toolUse,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false for failed tool use")
	}

	if output.MemoriesCreated != 0 {
		t.Errorf("expected 0 memories created, got %d", output.MemoriesCreated)
	}
}

func TestMemorizeFromToolUse_SkipsEmptyResult(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "web_search", 1, nil)
	toolUse.Complete(nil) // Empty result

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse: toolUse,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false for empty result")
	}
}

func TestMemorizeFromToolUse_SkipsShortResult(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "web_search", 1, nil)
	toolUse.Complete("ok") // Too short

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse: toolUse,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false for short result")
	}

	if output.Reasoning != "Tool result too short for meaningful memory extraction" {
		t.Errorf("unexpected reasoning: %s", output.Reasoning)
	}
}

func TestMemorizeFromToolUse_AnalyzesAndRejectsTransientData(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	// LLM determines this is transient data not worth memorizing
	llmService.response = `{
		"should_memorize": false,
		"reasoning": "This is current weather data which is transient and will be stale quickly"
	}`

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "weather_api", 1, nil)
	toolUse.Complete(map[string]any{
		"temperature": 72,
		"conditions":  "sunny",
		"location":    "Seattle",
	})

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse:   toolUse,
		UserQuery: "What's the weather in Seattle?",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false for transient weather data")
	}

	if output.MemoriesCreated != 0 {
		t.Errorf("expected 0 memories created, got %d", output.MemoriesCreated)
	}
}

func TestMemorizeFromToolUse_ExtractsFromUserSpecificData(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	// Track which call we're on to return different responses
	callCount := 0
	llmService.chatFunc = func(messages interface{}) string {
		callCount++
		if callCount == 1 {
			// First call: analysis
			return `{
				"should_memorize": true,
				"reasoning": "This contains user-specific account configuration that would be useful in future conversations"
			}`
		}
		// Second call: extraction
		return `{
			"extracted_facts": ["User has premium account", "User prefers dark mode"],
			"importance_scores": [0.7, 0.6],
			"extraction_reasoning": "Extracted user preferences from account settings"
		}`
	}

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "get_user_settings", 1, nil)
	toolUse.Complete(map[string]any{
		"account_type":  "premium",
		"theme":         "dark",
		"notifications": true,
	})

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse:        toolUse,
		UserQuery:      "Show me my account settings",
		ConversationID: "conv_789",
		MessageID:      "msg_456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be true for user-specific data")
	}

	if output.MemoriesCreated != 2 {
		t.Errorf("expected 2 memories created, got %d", output.MemoriesCreated)
	}
}

func TestMemorizeFromToolUse_HandlesJSONResult(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	callCount := 0
	llmService.chatFunc = func(messages interface{}) string {
		callCount++
		if callCount == 1 {
			return `{"should_memorize": true, "reasoning": "Contains project info"}`
		}
		return `{
			"extracted_facts": ["Project uses PostgreSQL database"],
			"importance_scores": [0.8],
			"extraction_reasoning": "Extracted technical configuration"
		}`
	}

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "read_config", 1, nil)
	toolUse.Complete(map[string]any{
		"database": map[string]any{
			"type": "postgresql",
			"host": "localhost",
			"port": 5432,
		},
	})

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse:   toolUse,
		UserQuery: "What database are we using?",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be true")
	}

	if output.MemoriesCreated != 1 {
		t.Errorf("expected 1 memory created, got %d", output.MemoriesCreated)
	}
}

func TestMemorizeFromToolUse_HandlesStringResult(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	idGen := newMockIDGeneratorForExtract()

	callCount := 0
	llmService.chatFunc = func(messages interface{}) string {
		callCount++
		if callCount == 1 {
			return `{"should_memorize": true, "reasoning": "Contains contact info"}`
		}
		return `{
			"extracted_facts": ["Project manager is Alice at alice@example.com"],
			"importance_scores": [0.9],
			"extraction_reasoning": "Extracted contact information"
		}`
	}

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "lookup_contact", 1, nil)
	toolUse.Complete("Project Manager: Alice (alice@example.com)")

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse:   toolUse,
		UserQuery: "Who is the project manager?",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be true")
	}

	if output.MemoriesCreated != 1 {
		t.Errorf("expected 1 memory created, got %d", output.MemoriesCreated)
	}
}

func TestMemorizeFromToolUse_HandlesLLMAnalysisError(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.chatError = ErrLLMService
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "web_search", 1, nil)
	toolUse.Complete("Some long enough result that would normally be analyzed")

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse:   toolUse,
		UserQuery: "Search for something",
	})

	// Should not error, just skip memorization
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false when analysis fails")
	}
}

func TestMemorizeFromToolUse_HandlesMalformedAnalysisJSON(t *testing.T) {
	memoryService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.response = "this is not valid json"
	idGen := newMockIDGeneratorForExtract()

	extractMemories := NewExtractMemories(memoryService, llmService, idGen)
	uc := NewMemorizeFromToolUse(llmService, memoryService, extractMemories)

	toolUse := models.NewToolUse("tu_123", "msg_456", "web_search", 1, nil)
	toolUse.Complete("Some long enough result that would normally be analyzed by the LLM")

	output, err := uc.Execute(context.Background(), &MemorizeFromToolUseInput{
		ToolUse:   toolUse,
		UserQuery: "Search for something",
	})

	// Should not error, just default to not memorizing
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ShouldMemorize {
		t.Error("expected ShouldMemorize to be false for malformed JSON")
	}
}
