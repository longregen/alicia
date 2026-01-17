package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock implementations for ExtractMemories tests

type mockMemoryServiceForExtract struct {
	createFromConversationFunc func(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error)
	createWithEmbeddingsFunc   func(ctx context.Context, content string) (*models.Memory, error)
	searchWithScoresFunc       func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error)
	setImportanceFunc          func(ctx context.Context, id string, importance float32) (*models.Memory, error)

	// Track calls for assertions
	createdMemories []string
	searchQueries   []string
}

func newMockMemoryServiceForExtract() *mockMemoryServiceForExtract {
	return &mockMemoryServiceForExtract{
		createdMemories: []string{},
		searchQueries:   []string{},
	}
}

func (m *mockMemoryServiceForExtract) Create(ctx context.Context, content string) (*models.Memory, error) {
	return models.NewMemory("mem_test", content), nil
}

func (m *mockMemoryServiceForExtract) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	m.createdMemories = append(m.createdMemories, content)
	if m.createWithEmbeddingsFunc != nil {
		return m.createWithEmbeddingsFunc(ctx, content)
	}
	return models.NewMemory("mem_"+content[:min(10, len(content))], content), nil
}

func (m *mockMemoryServiceForExtract) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	m.createdMemories = append(m.createdMemories, content)
	if m.createFromConversationFunc != nil {
		return m.createFromConversationFunc(ctx, content, conversationID, messageID)
	}
	return models.NewMemory("mem_"+content[:min(10, len(content))], content), nil
}

func (m *mockMemoryServiceForExtract) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	return nil, errors.New("not found")
}

func (m *mockMemoryServiceForExtract) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryServiceForExtract) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryServiceForExtract) SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryServiceForExtract) SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return []*models.Memory{}, nil
}

func (m *mockMemoryServiceForExtract) SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
	m.searchQueries = append(m.searchQueries, query)
	if m.searchWithScoresFunc != nil {
		return m.searchWithScoresFunc(ctx, query, threshold, limit)
	}
	return []*ports.MemorySearchResult{}, nil
}

func (m *mockMemoryServiceForExtract) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryServiceForExtract) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryServiceForExtract) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return []*models.MemoryUsage{}, nil
}

func (m *mockMemoryServiceForExtract) Update(ctx context.Context, memory *models.Memory) error {
	return nil
}

func (m *mockMemoryServiceForExtract) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryServiceForExtract) DeleteByConversationID(ctx context.Context, conversationID string) error {
	return nil
}

func (m *mockMemoryServiceForExtract) SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error) {
	if m.setImportanceFunc != nil {
		return m.setImportanceFunc(ctx, id, importance)
	}
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) SetConfidence(ctx context.Context, id string, confidence float32) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) AddTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) RemoveTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) Pin(ctx context.Context, id string, pinned bool) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

func (m *mockMemoryServiceForExtract) Archive(ctx context.Context, id string) (*models.Memory, error) {
	return &models.Memory{ID: id}, nil
}

type mockLLMServiceForExtract struct {
	chatResponse *ports.LLMResponse
	chatError    error
}

func newMockLLMServiceForExtract() *mockLLMServiceForExtract {
	return &mockLLMServiceForExtract{}
}

func (m *mockLLMServiceForExtract) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	if m.chatError != nil {
		return nil, m.chatError
	}
	if m.chatResponse != nil {
		return m.chatResponse, nil
	}
	return &ports.LLMResponse{Content: "NONE"}, nil
}

func (m *mockLLMServiceForExtract) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	return m.Chat(ctx, messages)
}

func (m *mockLLMServiceForExtract) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

func (m *mockLLMServiceForExtract) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

type mockIDGeneratorForExtract struct {
	counter int
}

func newMockIDGeneratorForExtract() *mockIDGeneratorForExtract {
	return &mockIDGeneratorForExtract{}
}

func (m *mockIDGeneratorForExtract) GenerateConversationID() string {
	m.counter++
	return "ac_test"
}

func (m *mockIDGeneratorForExtract) GenerateMessageID() string {
	m.counter++
	return "am_test"
}

func (m *mockIDGeneratorForExtract) GenerateSentenceID() string {
	m.counter++
	return "ams_test"
}

func (m *mockIDGeneratorForExtract) GenerateAudioID() string {
	m.counter++
	return "aa_test"
}

func (m *mockIDGeneratorForExtract) GenerateMemoryID() string {
	m.counter++
	return "amem_test"
}

func (m *mockIDGeneratorForExtract) GenerateMemoryUsageID() string {
	m.counter++
	return "amu_test"
}

func (m *mockIDGeneratorForExtract) GenerateToolID() string {
	m.counter++
	return "at_test"
}

func (m *mockIDGeneratorForExtract) GenerateToolUseID() string {
	m.counter++
	return "atu_test"
}

func (m *mockIDGeneratorForExtract) GenerateReasoningStepID() string {
	m.counter++
	return "ars_test"
}

func (m *mockIDGeneratorForExtract) GenerateCommentaryID() string {
	m.counter++
	return "aucc_test"
}

func (m *mockIDGeneratorForExtract) GenerateMetaID() string {
	m.counter++
	return "amt_test"
}

func (m *mockIDGeneratorForExtract) GenerateMCPServerID() string {
	m.counter++
	return "amcp_test"
}

func (m *mockIDGeneratorForExtract) GenerateVoteID() string {
	m.counter++
	return "av_test"
}

func (m *mockIDGeneratorForExtract) GenerateNoteID() string {
	m.counter++
	return "an_test"
}

func (m *mockIDGeneratorForExtract) GenerateSessionStatsID() string {
	m.counter++
	return "ass_test"
}

func (m *mockIDGeneratorForExtract) GenerateOptimizationRunID() string {
	m.counter++
	return "aor_test"
}

func (m *mockIDGeneratorForExtract) GeneratePromptCandidateID() string {
	m.counter++
	return "apc_test"
}

func (m *mockIDGeneratorForExtract) GeneratePromptEvaluationID() string {
	m.counter++
	return "ape_test"
}

func (m *mockIDGeneratorForExtract) GenerateTrainingExampleID() string {
	m.counter++
	return "gte_test"
}

func (m *mockIDGeneratorForExtract) GenerateSystemPromptVersionID() string {
	m.counter++
	return "spv_test"
}

func (m *mockIDGeneratorForExtract) GenerateRequestID() string {
	m.counter++
	return "areq_test"
}

// Helper to create use case for tests
func createExtractMemoriesUC(memService ports.MemoryService, llmService ports.LLMService) *ExtractMemories {
	return NewExtractMemories(memService, llmService, newMockIDGeneratorForExtract())
}

// Tests

func TestExtractMemories_EmptyInput(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.CreatedMemories) != 0 {
		t.Errorf("expected 0 created memories, got %d", len(output.CreatedMemories))
	}
}

func TestExtractMemories_ShortInput(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "Short text",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Reasoning != "Text too short for meaningful memory extraction" {
		t.Errorf("expected short text reasoning, got: %s", output.Reasoning)
	}
}

func TestExtractMemories_NoMemoriesToExtract(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.chatResponse = &ports.LLMResponse{Content: "NONE"}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "This is a long enough text but contains no important information to remember, just casual conversation filler.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.CreatedMemories) != 0 {
		t.Errorf("expected 0 created memories, got %d", len(output.CreatedMemories))
	}
}

func TestExtractMemories_ExtractsFactsWithJSONFormat(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()

	// Simulate structured JSON response from LLM
	jsonResponse := map[string]interface{}{
		"extracted_facts":      []string{"User's favorite color is blue", "User works as a software engineer"},
		"importance_scores":    []float64{0.7, 0.8},
		"extraction_reasoning": "Extracted personal preferences and biographical info",
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "User: My favorite color is blue and I work as a software engineer. Assistant: That's interesting!",
		ConversationID:   "conv_123",
		MessageID:        "msg_456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.CreatedMemories) != 2 {
		t.Errorf("expected 2 created memories, got %d", len(output.CreatedMemories))
	}
	if len(output.ExtractedMemories) != 2 {
		t.Errorf("expected 2 extracted memories, got %d", len(output.ExtractedMemories))
	}
	if output.Reasoning != "Extracted personal preferences and biographical info" {
		t.Errorf("unexpected reasoning: %s", output.Reasoning)
	}
}

func TestExtractMemories_ExtractsFactsWithLegacyFormat(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()

	// Simulate legacy MEMORY: format response
	llmService.chatResponse = &ports.LLMResponse{
		Content: "MEMORY: User prefers dark mode\nMEMORY: User's birthday is March 15th",
	}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "I prefer dark mode and my birthday is March 15th. This is a longer text to pass the minimum length requirement.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.CreatedMemories) != 2 {
		t.Errorf("expected 2 created memories, got %d", len(output.CreatedMemories))
	}
}

func TestExtractMemories_DuplicateDetection(t *testing.T) {
	memService := newMockMemoryServiceForExtract()

	// Set up mock to return a similar existing memory
	existingMemory := models.NewMemory("existing_mem_1", "User's favorite color is blue")
	memService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		// Return high similarity for the first fact (duplicate)
		if query == "User's favorite color is blue" {
			return []*ports.MemorySearchResult{
				{Memory: existingMemory, Similarity: 0.95},
			}, nil
		}
		// Return no matches for other facts
		return []*ports.MemorySearchResult{}, nil
	}

	llmService := newMockLLMServiceForExtract()
	jsonResponse := map[string]interface{}{
		"extracted_facts":      []string{"User's favorite color is blue", "User has a cat named Whiskers"},
		"importance_scores":    []float64{0.8, 0.7},
		"extraction_reasoning": "Extracted preferences and pet info",
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "My favorite color is blue and I have a cat named Whiskers. This is enough text to pass the minimum length check.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 extracted but only 1 created (one is duplicate)
	if len(output.ExtractedMemories) != 2 {
		t.Errorf("expected 2 extracted memories, got %d", len(output.ExtractedMemories))
	}
	if len(output.CreatedMemories) != 1 {
		t.Errorf("expected 1 created memory (1 duplicate skipped), got %d", len(output.CreatedMemories))
	}
	if output.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", output.SkippedCount)
	}

	// Check that the duplicate was marked correctly
	var foundDupe bool
	for _, em := range output.ExtractedMemories {
		if em.Content == "User's favorite color is blue" {
			if !em.IsDupe {
				t.Error("expected memory to be marked as duplicate")
			}
			if em.DupeOf != "existing_mem_1" {
				t.Errorf("expected DupeOf to be 'existing_mem_1', got '%s'", em.DupeOf)
			}
			foundDupe = true
		}
	}
	if !foundDupe {
		t.Error("did not find the duplicate memory in extracted memories")
	}
}

func TestExtractMemories_LowImportanceFiltering(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()

	// Return facts with varying importance scores
	jsonResponse := map[string]interface{}{
		"extracted_facts":      []string{"Important fact about user", "Minor detail", "Critical preference"},
		"importance_scores":    []float64{0.8, 0.2, 0.9},
		"extraction_reasoning": "Mixed importance facts",
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "This is a conversation with important and minor details that needs to be long enough for extraction.",
		MinImportance:    0.3, // Skip anything below 0.3
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should skip the "Minor detail" with 0.2 importance
	if len(output.CreatedMemories) != 2 {
		t.Errorf("expected 2 created memories (1 low importance skipped), got %d", len(output.CreatedMemories))
	}
	if output.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", output.SkippedCount)
	}
}

func TestExtractMemories_CustomDuplicateThreshold(t *testing.T) {
	memService := newMockMemoryServiceForExtract()

	// Track the threshold passed to SearchWithScores
	var capturedThreshold float32
	existingMemory := models.NewMemory("existing_mem_1", "Some memory")
	memService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		capturedThreshold = threshold
		// Return a result with 0.80 similarity
		return []*ports.MemorySearchResult{
			{Memory: existingMemory, Similarity: 0.80},
		}, nil
	}

	llmService := newMockLLMServiceForExtract()
	jsonResponse := map[string]interface{}{
		"extracted_facts":   []string{"Test fact"},
		"importance_scores": []float64{0.7},
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	// With threshold 0.75, the 0.80 similarity should be considered a duplicate
	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText:   "This is a test conversation that is long enough for memory extraction to proceed.",
		DuplicateThreshold: 0.75,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedThreshold != 0.75 {
		t.Errorf("expected threshold 0.75, got %f", capturedThreshold)
	}
	if len(output.CreatedMemories) != 0 {
		t.Errorf("expected 0 created memories (duplicate), got %d", len(output.CreatedMemories))
	}
}

func TestExtractMemories_LLMError(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()
	llmService.chatError = errors.New("LLM service unavailable")

	uc := createExtractMemoriesUC(memService, llmService)

	_, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "This is a test conversation that is long enough for memory extraction to proceed with the LLM.",
	})

	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
	if err.Error() != "failed to extract facts: LLM chat failed: LLM service unavailable" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExtractMemories_MemoryCreationError(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	memService.createFromConversationFunc = func(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
		return nil, errors.New("database error")
	}

	llmService := newMockLLMServiceForExtract()
	jsonResponse := map[string]interface{}{
		"extracted_facts":   []string{"Test fact that should fail to create"},
		"importance_scores": []float64{0.8},
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "This is a test conversation for memory creation failure testing with enough length.",
		ConversationID:   "conv_123",
		MessageID:        "msg_456",
	})

	// Should not return error, just log warning and continue
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Memory should be in extracted but not in created
	if len(output.ExtractedMemories) != 1 {
		t.Errorf("expected 1 extracted memory, got %d", len(output.ExtractedMemories))
	}
	if len(output.CreatedMemories) != 0 {
		t.Errorf("expected 0 created memories (creation failed), got %d", len(output.CreatedMemories))
	}
}

func TestExtractMemories_WithoutConversationContext(t *testing.T) {
	memService := newMockMemoryServiceForExtract()

	// Track which creation method is called
	var usedCreateWithEmbeddings bool
	memService.createWithEmbeddingsFunc = func(ctx context.Context, content string) (*models.Memory, error) {
		usedCreateWithEmbeddings = true
		return models.NewMemory("mem_test", content), nil
	}

	llmService := newMockLLMServiceForExtract()
	jsonResponse := map[string]interface{}{
		"extracted_facts":   []string{"Standalone fact"},
		"importance_scores": []float64{0.7},
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	// No ConversationID or MessageID provided
	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "This is standalone text for extraction without conversation context, needs to be long enough.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !usedCreateWithEmbeddings {
		t.Error("expected CreateWithEmbeddings to be called when no conversation context provided")
	}
	if len(output.CreatedMemories) != 1 {
		t.Errorf("expected 1 created memory, got %d", len(output.CreatedMemories))
	}
}

func TestExtractMemories_ParsesKeyValueFormat(t *testing.T) {
	memService := newMockMemoryServiceForExtract()
	llmService := newMockLLMServiceForExtract()

	// Test key-value format that some LLMs might return
	llmService.chatResponse = &ports.LLMResponse{
		Content: `extracted_facts: ["User likes coffee", "User is from Seattle"]
importance_scores: [0.6, 0.7]
extraction_reasoning: Key-value format response`,
	}

	uc := createExtractMemoriesUC(memService, llmService)

	output, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "I really like coffee and I'm originally from Seattle. This needs to be a longer conversation text.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.CreatedMemories) != 2 {
		t.Errorf("expected 2 created memories, got %d", len(output.CreatedMemories))
	}
}

func TestParseExtractionResponse_VariousFormats(t *testing.T) {
	uc := &ExtractMemories{}

	tests := []struct {
		name          string
		content       string
		expectedFacts int
	}{
		{
			name: "JSON format",
			content: `{"extracted_facts": ["fact1", "fact2"], "importance_scores": [0.8, 0.7], "extraction_reasoning": "test"}`,
			expectedFacts: 2,
		},
		{
			name: "NONE response",
			content: "NONE",
			expectedFacts: 0,
		},
		{
			name: "Legacy MEMORY format",
			content: "MEMORY: First memory\nMEMORY: Second memory\nMEMORY: Third memory",
			expectedFacts: 3,
		},
		{
			name: "Key-value format",
			content: `extracted_facts: ["kvfact1"]
importance_scores: [0.5]`,
			expectedFacts: 1,
		},
		{
			name: "Empty string",
			content: "",
			expectedFacts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts, _, _ := uc.parseExtractionResponse(tt.content)
			if len(facts) != tt.expectedFacts {
				t.Errorf("expected %d facts, got %d", tt.expectedFacts, len(facts))
			}
		})
	}
}

func TestExtractMemories_DefaultThresholds(t *testing.T) {
	memService := newMockMemoryServiceForExtract()

	var capturedThreshold float32
	memService.searchWithScoresFunc = func(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
		capturedThreshold = threshold
		return []*ports.MemorySearchResult{}, nil
	}

	llmService := newMockLLMServiceForExtract()
	jsonResponse := map[string]interface{}{
		"extracted_facts":   []string{"Test fact"},
		"importance_scores": []float64{0.5},
	}
	jsonBytes, _ := json.Marshal(jsonResponse)
	llmService.chatResponse = &ports.LLMResponse{Content: string(jsonBytes)}

	uc := createExtractMemoriesUC(memService, llmService)

	// Don't set any thresholds, rely on defaults
	_, err := uc.Execute(context.Background(), &ExtractMemoriesInput{
		ConversationText: "This is a test conversation that is long enough for memory extraction testing.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default duplicate threshold should be 0.85
	if capturedThreshold != 0.85 {
		t.Errorf("expected default threshold 0.85, got %f", capturedThreshold)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
