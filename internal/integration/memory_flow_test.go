//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/domain/models"
)

func TestMemoryFlow_CreateAndRetrieve(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	memoryRepo := postgres.NewMemoryRepository(db.Pool)
	idGen := id.NewGenerator()
	mockEmbedder := &mockEmbedder{}

	memorySvc := services.NewMemoryService(memoryRepo, nil, mockEmbedder, idGen)

	// Test: Create a memory
	memory, err := memorySvc.Create(ctx, &services.CreateMemoryInput{
		Content:    "The user prefers dark mode in their IDE",
		Importance: 0.8,
		Confidence: 0.9,
		SourceType: models.SourceTypeManual,
		Tags:       []string{"preferences", "ui"},
	})
	if err != nil {
		t.Fatalf("failed to create memory: %v", err)
	}

	if memory.ID == "" {
		t.Fatal("memory ID should not be empty")
	}
	if memory.Content != "The user prefers dark mode in their IDE" {
		t.Errorf("unexpected content: %s", memory.Content)
	}
	if memory.Importance != 0.8 {
		t.Errorf("expected importance 0.8, got %f", memory.Importance)
	}
	if len(memory.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(memory.Tags))
	}

	// Test: Retrieve the memory
	retrieved, err := memoryRepo.GetByID(ctx, memory.ID)
	if err != nil {
		t.Fatalf("failed to retrieve memory: %v", err)
	}

	if retrieved.ID != memory.ID {
		t.Errorf("expected ID %s, got %s", memory.ID, retrieved.ID)
	}
	if retrieved.Content != memory.Content {
		t.Errorf("expected content %s, got %s", memory.Content, retrieved.Content)
	}

	// Test: Update memory
	retrieved.SetImportance(0.9)
	retrieved.AddTag("important")

	err = memoryRepo.Update(ctx, retrieved)
	if err != nil {
		t.Fatalf("failed to update memory: %v", err)
	}

	updated, err := memoryRepo.GetByID(ctx, memory.ID)
	if err != nil {
		t.Fatalf("failed to retrieve updated memory: %v", err)
	}

	if updated.Importance != 0.9 {
		t.Errorf("expected importance 0.9, got %f", updated.Importance)
	}
	if len(updated.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(updated.Tags))
	}
}

func TestMemoryFlow_SearchBySimilarity(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	memoryRepo := postgres.NewMemoryRepository(db.Pool)

	// Create memories with embeddings
	embedding1 := fixtures.GenerateEmbedding(1536)
	embedding2 := fixtures.GenerateEmbedding(1536)
	embedding3 := fixtures.GenerateEmbedding(1536)

	// Modify embeddings to create different similarities
	for i := range embedding1 {
		embedding1[i] = 0.1
	}
	for i := range embedding2 {
		embedding2[i] = 0.1 // Very similar to embedding1
	}
	for i := range embedding3 {
		embedding3[i] = 0.9 // Different from embedding1 and embedding2
	}

	memory1 := fixtures.CreateMemoryWithEmbedding(ctx, t, "mem1", "User likes Go programming", embedding1)
	memory2 := fixtures.CreateMemoryWithEmbedding(ctx, t, "mem2", "User prefers statically typed languages", embedding2)
	memory3 := fixtures.CreateMemoryWithEmbedding(ctx, t, "mem3", "User's favorite color is blue", embedding3)

	// Search for similar memories using embedding1 as query
	results, err := memoryRepo.SearchBySimilarity(ctx, embedding1, 10, 0.0)
	if err != nil {
		t.Fatalf("failed to search memories: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}

	// The most similar should be memory1 itself
	if results[0].ID != memory1.ID {
		t.Errorf("expected first result to be memory1, got %s", results[0].ID)
	}

	// memory2 should be more similar than memory3
	found2 := false
	found3 := false
	pos2 := 0
	pos3 := 0

	for i, mem := range results {
		if mem.ID == memory2.ID {
			found2 = true
			pos2 = i
		}
		if mem.ID == memory3.ID {
			found3 = true
			pos3 = i
		}
	}

	if !found2 || !found3 {
		t.Error("expected to find both memory2 and memory3 in results")
	}

	if found2 && found3 && pos2 > pos3 {
		t.Error("expected memory2 to be ranked higher than memory3")
	}
}

func TestMemoryFlow_TrackMemoryUsage(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	memoryRepo := postgres.NewMemoryRepository(db.Pool)
	memoryUsageRepo := postgres.NewMemoryUsageRepository(db.Pool)
	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	idGen := id.NewGenerator()

	// Create test data
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Test Conversation")
	message := fixtures.CreateMessage(ctx, t, "msg1", conversation.ID, models.MessageRoleUser, "Tell me about Go", 1)
	memory := fixtures.CreateMemory(ctx, t, "mem1", "Go is a statically typed language")

	// Create memory usage tracking
	memoryUsage := models.NewMemoryUsage(idGen.GenerateMemoryUsageID(), conversation.ID, message.ID, memory.ID)
	memoryUsage.SimilarityScore = 0.85
	memoryUsage.PositionInResults = 1

	err := memoryUsageRepo.Create(ctx, memoryUsage)
	if err != nil {
		t.Fatalf("failed to create memory usage: %v", err)
	}

	// Retrieve usage by conversation
	usages, err := memoryUsageRepo.ListByConversation(ctx, conversation.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list memory usages: %v", err)
	}

	if len(usages) != 1 {
		t.Errorf("expected 1 memory usage, got %d", len(usages))
	}

	if usages[0].MemoryID != memory.ID {
		t.Errorf("expected memory ID %s, got %s", memory.ID, usages[0].MemoryID)
	}
	if usages[0].SimilarityScore != 0.85 {
		t.Errorf("expected similarity score 0.85, got %f", usages[0].SimilarityScore)
	}

	// Retrieve usage by message
	messageUsages, err := memoryUsageRepo.ListByMessage(ctx, message.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list memory usages by message: %v", err)
	}

	if len(messageUsages) != 1 {
		t.Errorf("expected 1 memory usage, got %d", len(messageUsages))
	}

	// Retrieve usage by memory
	memoryUsagesByMem, err := memoryUsageRepo.ListByMemory(ctx, memory.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list memory usages by memory: %v", err)
	}

	if len(memoryUsagesByMem) != 1 {
		t.Errorf("expected 1 memory usage, got %d", len(memoryUsagesByMem))
	}
}

func TestMemoryFlow_ListAndFilter(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	memoryRepo := postgres.NewMemoryRepository(db.Pool)

	// Create memories with different tags
	mem1 := fixtures.CreateMemory(ctx, t, "mem1", "Memory about preferences")
	mem1.AddTag("preferences")
	mem1.AddTag("ui")
	memoryRepo.Update(ctx, mem1)

	mem2 := fixtures.CreateMemory(ctx, t, "mem2", "Memory about skills")
	mem2.AddTag("skills")
	mem2.AddTag("programming")
	memoryRepo.Update(ctx, mem2)

	mem3 := fixtures.CreateMemory(ctx, t, "mem3", "Memory about preferences and skills")
	mem3.AddTag("preferences")
	mem3.AddTag("skills")
	memoryRepo.Update(ctx, mem3)

	// Test: List all memories
	allMemories, err := memoryRepo.List(ctx, 100, 0)
	if err != nil {
		t.Fatalf("failed to list memories: %v", err)
	}

	if len(allMemories) != 3 {
		t.Errorf("expected 3 memories, got %d", len(allMemories))
	}

	// Test: Search by tag
	prefMemories, err := memoryRepo.SearchByTags(ctx, []string{"preferences"}, 100, 0)
	if err != nil {
		t.Fatalf("failed to search by tags: %v", err)
	}

	if len(prefMemories) != 2 {
		t.Errorf("expected 2 memories with 'preferences' tag, got %d", len(prefMemories))
	}

	// Test: Delete memory
	err = memoryRepo.Delete(ctx, mem1.ID)
	if err != nil {
		t.Fatalf("failed to delete memory: %v", err)
	}

	// Verify deletion
	allMemories, err = memoryRepo.List(ctx, 100, 0)
	if err != nil {
		t.Fatalf("failed to list memories after deletion: %v", err)
	}

	if len(allMemories) != 2 {
		t.Errorf("expected 2 memories after deletion, got %d", len(allMemories))
	}
}

// mockEmbedder simulates embedding generation
type mockEmbedder struct{}

func (m *mockEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Return a simple mock embedding
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.5
	}
	return embedding, nil
}
