package prompt_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
)

// TestExampleMemoryAwareModule demonstrates basic usage of MemoryAwareModule
func TestExampleMemoryAwareModule(t *testing.T) {
	t.Skip("This is a documentation example, not a runnable test")

	// ExampleMemoryAwareModule demonstrates basic usage of MemoryAwareModule
	// Create a mock memory service with some pre-populated memories
	memService := &mockMemoryService{
		searchResults: []*ports.MemorySearchResult{
			{
				Memory: &models.Memory{
					ID:      "mem1",
					Content: "Q: What is Go? A: A programming language developed by Google",
					Tags:    []string{"qa", "programming"},
				},
				Similarity: 0.9,
			},
			{
				Memory: &models.Memory{
					ID:      "mem2",
					Content: "Q: What is Python? A: A high-level interpreted programming language",
					Tags:    []string{"qa", "programming"},
				},
				Similarity: 0.85,
			},
		},
	}

	// Create signature for Q&A
	sig := prompt.MustParseSignature("question -> answer")

	// Create memory-aware module with custom options
	module := prompt.NewMemoryAwareModule(
		sig,
		memService,
		prompt.WithMaxDemonstrations(5),
		prompt.WithSimilarityThreshold(0.7),
	)

	// Process a question (in production, this would call the LLM)
	inputs := map[string]any{
		"question": "Tell me about programming languages",
	}

	ctx := context.Background()
	_, err := module.Process(ctx, inputs)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Successfully processed with memory-augmented context")
	}

	// Output:
	// Successfully processed with memory-augmented context
}

// TestExampleMemoryConverter demonstrates memory-to-example conversion
func TestExampleMemoryConverter(t *testing.T) {
	t.Skip("This is a documentation example, not a runnable test")
	converter := prompt.NewMemoryConverter()

	// Example 1: Q&A format
	memory1 := &models.Memory{
		ID:        "mem1",
		Content:   "Q: What is AI? A: Artificial Intelligence",
		Tags:      []string{"qa"},
		CreatedAt: time.Now(),
	}

	example1, ok := converter.ConvertMemory(memory1)
	if ok {
		fmt.Printf("Question: %v\n", example1.Inputs["question"])
		fmt.Printf("Answer: %v\n", example1.Outputs["answer"])
	}

	// Example 2: Input/Output format
	memory2 := &models.Memory{
		ID:        "mem2",
		Content:   "Input: raw text Output: processed text",
		Tags:      []string{},
		CreatedAt: time.Now(),
	}

	example2, ok := converter.ConvertMemory(memory2)
	if ok {
		fmt.Printf("Input: %v\n", example2.Inputs["input"])
		fmt.Printf("Output: %v\n", example2.Outputs["output"])
	}

	// Output:
	// Question: What is AI?
	// Answer: Artificial Intelligence
	// Input: raw text
	// Output: processed text
}

// TestExampleRankMemoriesByRelevance demonstrates memory ranking
func TestExampleRankMemoriesByRelevance(t *testing.T) {
	t.Skip("This is a documentation example, not a runnable test")
	now := time.Now()

	memories := []*ports.MemorySearchResult{
		{
			Memory: &models.Memory{
				ID:         "mem1",
				Content:    "User prefers dark mode",
				Importance: 0.9,
				Tags:       []string{"preference"},
				CreatedAt:  now,
			},
			Similarity: 0.85,
		},
		{
			Memory: &models.Memory{
				ID:         "mem2",
				Content:    "User knows Python",
				Importance: 0.6,
				Tags:       []string{"fact"},
				CreatedAt:  now.Add(-24 * time.Hour),
			},
			Similarity: 0.9,
		},
		{
			Memory: &models.Memory{
				ID:         "mem3",
				Content:    "Always validate user input",
				Importance: 0.8,
				Tags:       []string{"instruction"},
				CreatedAt:  now.Add(-48 * time.Hour),
			},
			Similarity: 0.8,
		},
	}

	// Rank with preference for "preference" category
	ranked := prompt.RankMemoriesByRelevance(memories, []string{"preference"})

	for i, score := range ranked {
		fmt.Printf("%d. Memory %s - Combined Score: %.2f (Category Match: %v)\n",
			i+1, score.Memory.ID, score.CombinedScore, score.CategoryMatch)
	}

	// Output will show mem1 ranked highest due to category match and high importance
	// (exact output depends on recency calculation)
}

// TestExampleCreateMemoryFromExample demonstrates creating memories from examples
func TestExampleCreateMemoryFromExample(t *testing.T) {
	t.Skip("This is a documentation example, not a runnable test")
	// Create an example
	example := prompt.Example{
		Inputs: map[string]any{
			"question": "What is Go?",
		},
		Outputs: map[string]any{
			"answer": "A programming language",
		},
	}

	// Convert to memory
	memory, err := prompt.CreateMemoryFromExample(example, []string{"qa", "programming"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Memory Content:\n%s\n", memory.Content)
	fmt.Printf("Tags: %v\n", memory.Tags)
	fmt.Printf("Importance: %.1f\n", memory.Importance)

	// Output:
	// Memory Content:
	// Q: What is Go?
	// A: A programming language
	// Tags: [qa programming]
	// Importance: 0.5
}

// TestExampleEnrichExampleWithMemory demonstrates example enrichment
func TestExampleEnrichExampleWithMemory(t *testing.T) {
	t.Skip("This is a documentation example, not a runnable test")
	example := prompt.Example{
		Inputs: map[string]any{
			"question": "What is machine learning?",
		},
		Outputs: map[string]any{
			"answer": "A subset of AI",
		},
	}

	memories := []*models.Memory{
		{
			ID:      "mem1",
			Content: "Machine learning uses statistical techniques",
		},
		{
			ID:      "mem2",
			Content: "Deep learning is a type of machine learning",
		},
	}

	enriched := prompt.EnrichExampleWithMemory(example, memories)

	// Check that original fields are preserved
	fmt.Printf("Question: %v\n", enriched.Inputs["question"])

	// Check that memory context was added
	if memContext, ok := enriched.Inputs["memories"].(string); ok {
		fmt.Printf("Has memory context: %v\n", len(memContext) > 0)
	}

	// Output:
	// Question: What is machine learning?
	// Has memory context: true
}

// mockMemoryService implements ports.MemoryService for examples
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

func (m *mockMemoryService) DeleteByConversationID(ctx context.Context, conversationID string) error {
	return nil
}
