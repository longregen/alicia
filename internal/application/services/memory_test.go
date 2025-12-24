package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

var (
	errNotFound = errors.New("not found")
)

// Mock implementations

type mockMemoryRepo struct {
	store map[string]*models.Memory
}

func newMockMemoryRepo() *mockMemoryRepo {
	return &mockMemoryRepo{
		store: make(map[string]*models.Memory),
	}
}

func (m *mockMemoryRepo) Create(ctx context.Context, memory *models.Memory) error {
	m.store[memory.ID] = memory
	return nil
}

func (m *mockMemoryRepo) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	if mem, ok := m.store[id]; ok {
		return mem, nil
	}
	return nil, errNotFound
}

func (m *mockMemoryRepo) Update(ctx context.Context, memory *models.Memory) error {
	if _, ok := m.store[memory.ID]; !ok {
		return errNotFound
	}
	m.store[memory.ID] = memory
	return nil
}

func (m *mockMemoryRepo) Delete(ctx context.Context, id string) error {
	if mem, ok := m.store[id]; ok {
		now := time.Now()
		mem.DeletedAt = &now
		mem.UpdatedAt = now
		m.store[id] = mem
		return nil
	}
	return errNotFound
}

func (m *mockMemoryRepo) Search(ctx context.Context, embedding []float32, limit int) ([]*models.Memory, error) {
	memories := make([]*models.Memory, 0)
	count := 0
	for _, mem := range m.store {
		if mem.DeletedAt == nil && len(mem.Embeddings) > 0 {
			memories = append(memories, mem)
			count++
			if count >= limit {
				break
			}
		}
	}
	return memories, nil
}

func (m *mockMemoryRepo) SearchWithThreshold(ctx context.Context, embedding []float32, threshold float32, limit int) ([]*models.Memory, error) {
	return m.Search(ctx, embedding, limit)
}

func (m *mockMemoryRepo) SearchWithScores(ctx context.Context, embedding []float32, limit int) ([]*ports.MemoryWithScore, error) {
	memories, err := m.Search(ctx, embedding, limit)
	if err != nil {
		return nil, err
	}
	results := make([]*ports.MemoryWithScore, len(memories))
	for i, mem := range memories {
		results[i] = &ports.MemoryWithScore{
			Memory:          mem,
			SimilarityScore: 0.9, // Mock similarity score
		}
	}
	return results, nil
}

func (m *mockMemoryRepo) SearchWithThresholdAndScores(ctx context.Context, embedding []float32, threshold float32, limit int) ([]*ports.MemoryWithScore, error) {
	return m.SearchWithScores(ctx, embedding, limit)
}

func (m *mockMemoryRepo) SearchMemories(ctx context.Context, opts ports.MemorySearchOptions) ([]*ports.MemorySearchResult, error) {
	memories, err := m.Search(ctx, opts.Embedding, opts.Limit)
	if err != nil {
		return nil, err
	}
	results := make([]*ports.MemorySearchResult, len(memories))
	for i, mem := range memories {
		similarity := float32(0.0)
		if opts.IncludeScores {
			similarity = 0.9 // Mock similarity score
		}
		results[i] = &ports.MemorySearchResult{
			Memory:     mem,
			Similarity: similarity,
		}
	}
	return results, nil
}

func (m *mockMemoryRepo) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	memories := make([]*models.Memory, 0)
	for _, mem := range m.store {
		if mem.DeletedAt == nil {
			for _, tag := range tags {
				for _, memTag := range mem.Tags {
					if tag == memTag {
						memories = append(memories, mem)
						break
					}
				}
			}
		}
	}
	return memories, nil
}

type mockMemoryUsageRepo struct {
	store map[string]*models.MemoryUsage
}

func newMockMemoryUsageRepo() *mockMemoryUsageRepo {
	return &mockMemoryUsageRepo{
		store: make(map[string]*models.MemoryUsage),
	}
}

func (m *mockMemoryUsageRepo) Create(ctx context.Context, usage *models.MemoryUsage) error {
	m.store[usage.ID] = usage
	return nil
}

func (m *mockMemoryUsageRepo) GetByID(ctx context.Context, id string) (*models.MemoryUsage, error) {
	if usage, ok := m.store[id]; ok {
		return usage, nil
	}
	return nil, errNotFound
}

func (m *mockMemoryUsageRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	usages := make([]*models.MemoryUsage, 0)
	for _, usage := range m.store {
		if usage.MessageID == messageID {
			usages = append(usages, usage)
		}
	}
	return usages, nil
}

func (m *mockMemoryUsageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	usages := make([]*models.MemoryUsage, 0)
	for _, usage := range m.store {
		if usage.ConversationID == conversationID {
			usages = append(usages, usage)
		}
	}
	return usages, nil
}

func (m *mockMemoryUsageRepo) GetByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error) {
	usages := make([]*models.MemoryUsage, 0)
	for _, usage := range m.store {
		if usage.MemoryID == memoryID {
			usages = append(usages, usage)
		}
	}
	return usages, nil
}

type mockEmbeddingService struct{}

func (m *mockEmbeddingService) Embed(ctx context.Context, text string) (*ports.EmbeddingResult, error) {
	// Return dummy embeddings
	return &ports.EmbeddingResult{
		Embedding:  []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		Model:      "test-model",
		Dimensions: 5,
	}, nil
}

func (m *mockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([]*ports.EmbeddingResult, error) {
	results := make([]*ports.EmbeddingResult, len(texts))
	for i := range texts {
		results[i] = &ports.EmbeddingResult{
			Embedding:  []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			Model:      "test-model",
			Dimensions: 5,
		}
	}
	return results, nil
}

func (m *mockEmbeddingService) GetDimensions() int {
	return 5
}

func (m *mockMemoryUsageRepo) GetUsageStats(ctx context.Context, memoryID string) (*ports.MemoryUsageStats, error) {
	usages := make([]*models.MemoryUsage, 0)
	for _, usage := range m.store {
		if usage.MemoryID == memoryID {
			usages = append(usages, usage)
		}
	}

	if len(usages) == 0 {
		return &ports.MemoryUsageStats{
			TotalUsageCount:   0,
			AverageSimilarity: 0,
			LastUsedAt:        nil,
		}, nil
	}

	// Calculate stats
	totalScore := float32(0)
	var lastUsedAt *time.Time

	for _, usage := range usages {
		totalScore += usage.SimilarityScore
		if lastUsedAt == nil || usage.CreatedAt.After(*lastUsedAt) {
			t := usage.CreatedAt
			lastUsedAt = &t
		}
	}

	return &ports.MemoryUsageStats{
		TotalUsageCount:   len(usages),
		AverageSimilarity: totalScore / float32(len(usages)),
		LastUsedAt:        lastUsedAt,
	}, nil
}

// Tests

func TestMemoryService_Create(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	mem, err := svc.Create(context.Background(), "Test memory content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mem.ID != "mem_test1" {
		t.Errorf("expected ID mem_test1, got %s", mem.ID)
	}

	if mem.Content != "Test memory content" {
		t.Errorf("expected content 'Test memory content', got %s", mem.Content)
	}

	// Verify it was stored
	stored, _ := memRepo.GetByID(context.Background(), mem.ID)
	if stored == nil {
		t.Error("memory not stored in repository")
	}
}

func TestMemoryService_Create_EmptyContent(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	_, err := svc.Create(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

func TestMemoryService_CreateWithEmbeddings(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	mem, err := svc.CreateWithEmbeddings(context.Background(), "Test memory with embeddings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mem.Embeddings) == 0 {
		t.Error("expected embeddings to be set")
	}

	if mem.EmbeddingsInfo == nil {
		t.Error("expected embeddings info to be set")
	}

	if mem.EmbeddingsInfo.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %s", mem.EmbeddingsInfo.Model)
	}

	if mem.EmbeddingsInfo.Dimensions != 5 {
		t.Errorf("expected dimensions 5, got %d", mem.EmbeddingsInfo.Dimensions)
	}
}

func TestMemoryService_Search(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create some memories with embeddings
	svc.CreateWithEmbeddings(context.Background(), "Memory 1")
	svc.CreateWithEmbeddings(context.Background(), "Memory 2")
	svc.CreateWithEmbeddings(context.Background(), "Memory 3")

	// Search
	results, err := svc.Search(context.Background(), "test query", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}

	// All results should have embeddings
	for _, mem := range results {
		if len(mem.Embeddings) == 0 {
			t.Error("search result missing embeddings")
		}
	}
}

func TestMemoryService_Search_EmptyQuery(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	_, err := svc.Search(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
}

func TestMemoryService_SetImportance(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Set importance
	updated, err := svc.SetImportance(context.Background(), mem.ID, 0.8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Importance != 0.8 {
		t.Errorf("expected importance 0.8, got %f", updated.Importance)
	}
}

func TestMemoryService_SetImportance_Clamping(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Test clamping to max (1.0)
	updated, _ := svc.SetImportance(context.Background(), mem.ID, 1.5)
	if updated.Importance != 1.0 {
		t.Errorf("expected importance clamped to 1.0, got %f", updated.Importance)
	}

	// Test clamping to min (0.0)
	updated, _ = svc.SetImportance(context.Background(), mem.ID, -0.5)
	if updated.Importance != 0.0 {
		t.Errorf("expected importance clamped to 0.0, got %f", updated.Importance)
	}
}

func TestMemoryService_SetConfidence(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Set confidence
	updated, err := svc.SetConfidence(context.Background(), mem.ID, 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", updated.Confidence)
	}
}

func TestMemoryService_AddTag(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Add tag
	updated, err := svc.AddTag(context.Background(), mem.ID, "important")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updated.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(updated.Tags))
	}

	if updated.Tags[0] != "important" {
		t.Errorf("expected tag 'important', got %s", updated.Tags[0])
	}
}

func TestMemoryService_RemoveTag(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory and add a tag
	mem, _ := svc.Create(context.Background(), "Test")
	mem, _ = svc.AddTag(context.Background(), mem.ID, "important")

	// Remove tag
	updated, err := svc.RemoveTag(context.Background(), mem.ID, "important")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updated.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(updated.Tags))
	}
}

func TestMemoryService_GetByTags(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create memories with tags
	mem1, _ := svc.Create(context.Background(), "Memory 1")
	svc.AddTag(context.Background(), mem1.ID, "work")

	mem2, _ := svc.Create(context.Background(), "Memory 2")
	svc.AddTag(context.Background(), mem2.ID, "personal")

	mem3, _ := svc.Create(context.Background(), "Memory 3")
	svc.AddTag(context.Background(), mem3.ID, "work")

	// Get by tag
	results, err := svc.GetByTags(context.Background(), []string{"work"}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 memories with 'work' tag, got %d", len(results))
	}
}

func TestMemoryService_Delete(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Delete it
	err := svc.Delete(context.Background(), mem.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's soft-deleted
	stored, _ := memRepo.GetByID(context.Background(), mem.ID)
	if stored.DeletedAt == nil {
		t.Error("memory not soft-deleted")
	}

	// GetByID should now return error for deleted memory
	_, err = svc.GetByID(context.Background(), mem.ID)
	if err == nil {
		t.Error("expected error when getting deleted memory")
	}
}

func TestMemoryService_TrackUsage(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Track usage
	usage, err := svc.TrackUsage(context.Background(), mem.ID, "conv_123", "msg_456", 0.95)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.MemoryID != mem.ID {
		t.Errorf("expected memory ID %s, got %s", mem.ID, usage.MemoryID)
	}

	if usage.ConversationID != "conv_123" {
		t.Errorf("expected conversation ID conv_123, got %s", usage.ConversationID)
	}

	if usage.SimilarityScore != 0.95 {
		t.Errorf("expected similarity score 0.95, got %f", usage.SimilarityScore)
	}
}

func TestMemoryService_GetUsageByMemory(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory
	mem, _ := svc.Create(context.Background(), "Test")

	// Track multiple usages
	svc.TrackUsage(context.Background(), mem.ID, "conv_1", "msg_1", 0.9)
	svc.TrackUsage(context.Background(), mem.ID, "conv_2", "msg_2", 0.8)

	// Get usage
	usages, err := svc.GetUsageByMemory(context.Background(), mem.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(usages) != 2 {
		t.Errorf("expected 2 usages, got %d", len(usages))
	}
}

func TestMemoryService_RegenerateEmbeddings(t *testing.T) {
	memRepo := newMockMemoryRepo()
	usageRepo := newMockMemoryUsageRepo()
	embedding := &mockEmbeddingService{}
	idGen := &mockIDGenerator{}
	txManager := &mockTransactionManager{}

	svc := NewMemoryService(memRepo, usageRepo, embedding, idGen, txManager)

	// Create a memory without embeddings
	mem, _ := svc.Create(context.Background(), "Test")

	if len(mem.Embeddings) > 0 {
		t.Error("new memory should not have embeddings initially")
	}

	// Regenerate embeddings
	updated, err := svc.RegenerateEmbeddings(context.Background(), mem.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updated.Embeddings) == 0 {
		t.Error("expected embeddings to be generated")
	}

	if updated.EmbeddingsInfo == nil {
		t.Error("expected embeddings info to be set")
	}
}
