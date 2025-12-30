package prompt

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// mockMemoryService implements ports.MemoryService for testing
type mockMemoryService struct {
	memories      []*models.Memory
	searchResults []*ports.MemorySearchResult
	searchError   error
}

func (m *mockMemoryService) Create(ctx context.Context, content string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error) {
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return m.memories, nil
}

func (m *mockMemoryService) SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *mockMemoryService) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryService) Update(ctx context.Context, memory *models.Memory) error {
	return nil
}

func (m *mockMemoryService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryService) SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) SetConfidence(ctx context.Context, id string, confidence float32) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) AddTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) RemoveTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) Pin(ctx context.Context, id string, pinned bool) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) Archive(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

func TestNewMemoryAwareModule(t *testing.T) {
	sig := MustParseSignature("question -> answer")
	memService := &mockMemoryService{}

	module := NewMemoryAwareModule(sig, memService)

	if module == nil {
		t.Fatal("expected module to be created")
	}

	if module.memoryService != memService {
		t.Error("expected memory service to be set")
	}

	if module.maxDemos != 5 {
		t.Errorf("expected default maxDemos to be 5, got %d", module.maxDemos)
	}

	if module.threshold != 0.7 {
		t.Errorf("expected default threshold to be 0.7, got %f", module.threshold)
	}
}

func TestMemoryAwareModuleWithOptions(t *testing.T) {
	sig := MustParseSignature("question -> answer")
	memService := &mockMemoryService{}

	module := NewMemoryAwareModule(
		sig,
		memService,
		WithMaxDemonstrations(10),
		WithSimilarityThreshold(0.8),
	)

	if module.maxDemos != 10 {
		t.Errorf("expected maxDemos to be 10, got %d", module.maxDemos)
	}

	if module.threshold != 0.8 {
		t.Errorf("expected threshold to be 0.8, got %f", module.threshold)
	}
}

func TestConstructQuery(t *testing.T) {
	sig := MustParseSignature("question -> answer")
	memService := &mockMemoryService{}
	module := NewMemoryAwareModule(sig, memService)

	tests := []struct {
		name     string
		inputs   map[string]any
		expected string
	}{
		{
			name: "user_message priority",
			inputs: map[string]any{
				"user_message": "What is AI?",
				"context":      "Some context",
				"question":     "Different question",
			},
			expected: "What is AI?",
		},
		{
			name: "context fallback",
			inputs: map[string]any{
				"context":  "Some context",
				"question": "What is AI?",
			},
			expected: "Some context",
		},
		{
			name: "question fallback",
			inputs: map[string]any{
				"question": "What is AI?",
			},
			expected: "What is AI?",
		},
		{
			name: "concatenate multiple strings",
			inputs: map[string]any{
				"field1": "hello",
				"field2": "world",
			},
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.constructQuery(tt.inputs)
			if result != tt.expected {
				t.Errorf("expected query '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDetectMemoryCategory(t *testing.T) {
	sig := MustParseSignature("question -> answer")
	memService := &mockMemoryService{}
	module := NewMemoryAwareModule(sig, memService)

	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "preference category",
			tags:     []string{"preference", "other"},
			expected: "preference",
		},
		{
			name:     "fact category",
			tags:     []string{"fact", "knowledge"},
			expected: "fact",
		},
		{
			name:     "instruction category",
			tags:     []string{"instruction"},
			expected: "instruction",
		},
		{
			name:     "context category",
			tags:     []string{"context", "background"},
			expected: "context",
		},
		{
			name:     "no category",
			tags:     []string{"random", "tags"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.detectMemoryCategory(tt.tags)
			if result != tt.expected {
				t.Errorf("expected category '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestRankMemoriesByRelevance(t *testing.T) {
	now := time.Now()
	memories := []*ports.MemorySearchResult{
		{
			Memory: &models.Memory{
				ID:         "mem1",
				Content:    "Memory 1",
				Importance: 0.8,
				Tags:       []string{"preference"},
				CreatedAt:  now,
			},
			Similarity: 0.9,
		},
		{
			Memory: &models.Memory{
				ID:         "mem2",
				Content:    "Memory 2",
				Importance: 0.5,
				Tags:       []string{"fact"},
				CreatedAt:  now.Add(-24 * time.Hour),
			},
			Similarity: 0.7,
		},
		{
			Memory: &models.Memory{
				ID:         "mem3",
				Content:    "Memory 3",
				Importance: 0.6,
				Tags:       []string{"preference"},
				CreatedAt:  now.Add(-48 * time.Hour),
			},
			Similarity: 0.85,
		},
	}

	categoryFilter := []string{"preference"}
	ranked := RankMemoriesByRelevance(memories, categoryFilter)

	if len(ranked) != 3 {
		t.Errorf("expected 3 ranked memories, got %d", len(ranked))
	}

	// First should be mem1 (high similarity, high importance, recent, category match)
	if ranked[0].Memory.ID != "mem1" {
		t.Errorf("expected mem1 to be ranked first, got %s", ranked[0].Memory.ID)
	}

	// Check that category bonus was applied
	if !ranked[0].CategoryMatch {
		t.Error("expected mem1 to have category match")
	}

	// Verify scores are in descending order
	for i := 0; i < len(ranked)-1; i++ {
		if ranked[i].CombinedScore < ranked[i+1].CombinedScore {
			t.Errorf("scores not in descending order at index %d", i)
		}
	}
}

func TestRetrieveRelevantMemories(t *testing.T) {
	sig := MustParseSignature("question -> answer")

	now := time.Now()
	memory1 := &models.Memory{
		ID:         "mem1",
		Content:    "Q: What is AI? A: Artificial Intelligence",
		Importance: 0.8,
		Tags:       []string{"qa"},
		CreatedAt:  now,
	}

	memService := &mockMemoryService{
		searchResults: []*ports.MemorySearchResult{
			{
				Memory:     memory1,
				Similarity: 0.9,
			},
		},
	}

	module := NewMemoryAwareModule(sig, memService)

	inputs := map[string]any{
		"question": "Tell me about AI",
	}

	memories, err := module.retrieveRelevantMemories(context.Background(), inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(memories) != 1 {
		t.Errorf("expected 1 memory, got %d", len(memories))
	}

	if memories[0].ID != "mem1" {
		t.Errorf("expected mem1, got %s", memories[0].ID)
	}
}

func TestRetrieveRelevantMemoriesEmptyQuery(t *testing.T) {
	sig := MustParseSignature("question -> answer")
	memService := &mockMemoryService{}
	module := NewMemoryAwareModule(sig, memService)

	inputs := map[string]any{}

	memories, err := module.retrieveRelevantMemories(context.Background(), inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if memories != nil {
		t.Errorf("expected nil memories for empty query, got %d", len(memories))
	}
}
