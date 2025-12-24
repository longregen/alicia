package services

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type MemoryService struct {
	memoryRepo      ports.MemoryRepository
	memoryUsageRepo ports.MemoryUsageRepository
	embedding       ports.EmbeddingService
	idGenerator     ports.IDGenerator
	txManager       ports.TransactionManager
}

func NewMemoryService(
	memoryRepo ports.MemoryRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	embedding ports.EmbeddingService,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *MemoryService {
	return &MemoryService{
		memoryRepo:      memoryRepo,
		memoryUsageRepo: memoryUsageRepo,
		embedding:       embedding,
		idGenerator:     idGenerator,
		txManager:       txManager,
	}
}

func (s *MemoryService) Create(ctx context.Context, content string) (*models.Memory, error) {
	if content == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "memory content cannot be empty")
	}

	id := s.idGenerator.GenerateMemoryID()
	memory := models.NewMemory(id, content)

	if err := s.memoryRepo.Create(ctx, memory); err != nil {
		return nil, domain.NewDomainError(err, "failed to create memory")
	}

	return memory, nil
}

func (s *MemoryService) CreateWithEmbeddings(ctx context.Context, content string) (*models.Memory, error) {
	if content == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "memory content cannot be empty")
	}

	result, err := s.embedding.Embed(ctx, content)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate embeddings")
	}

	id := s.idGenerator.GenerateMemoryID()
	memory := models.NewMemory(id, content)

	embeddingsInfo := &models.EmbeddingsInfo{
		Model:      result.Model,
		Dimensions: result.Dimensions,
	}
	memory.SetEmbeddings(result.Embedding, embeddingsInfo)

	if err := s.memoryRepo.Create(ctx, memory); err != nil {
		return nil, domain.NewDomainError(err, "failed to create memory")
	}

	return memory, nil
}

func (s *MemoryService) CreateFromConversation(ctx context.Context, content, conversationID, messageID string) (*models.Memory, error) {
	if content == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "memory content cannot be empty")
	}

	// Generate embeddings before starting transaction (external API call)
	result, err := s.embedding.Embed(ctx, content)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate embeddings")
	}

	id := s.idGenerator.GenerateMemoryID()
	memory := models.NewMemory(id, content)

	embeddingsInfo := &models.EmbeddingsInfo{
		Model:      result.Model,
		Dimensions: result.Dimensions,
	}
	memory.SetEmbeddings(result.Embedding, embeddingsInfo)

	// Set source information
	memory.SourceType = models.SourceTypeConversation
	memory.SourceInfo = &models.SourceInfo{
		ConversationID: conversationID,
		MessageID:      messageID,
	}

	// Wrap memory creation in a transaction to ensure atomicity
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create memory with source attribution in one atomic operation
		if err := s.memoryRepo.Create(txCtx, memory); err != nil {
			return domain.NewDomainError(err, "failed to create memory")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	if id == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "memory ID cannot be empty")
	}

	memory, err := s.memoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMemoryNotFound, "memory not found")
	}

	if memory.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrMemoryNotFound, "memory has been deleted")
	}

	return memory, nil
}

func (s *MemoryService) Update(ctx context.Context, memory *models.Memory) error {
	if memory == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "memory cannot be nil")
	}

	if memory.ID == "" {
		return domain.NewDomainError(domain.ErrInvalidID, "memory ID cannot be empty")
	}

	existing, err := s.memoryRepo.GetByID(ctx, memory.ID)
	if err != nil {
		return domain.NewDomainError(domain.ErrMemoryNotFound, "memory not found")
	}

	if existing.DeletedAt != nil {
		return domain.NewDomainError(domain.ErrMemoryNotFound, "cannot update deleted memory")
	}

	if err := s.memoryRepo.Update(ctx, memory); err != nil {
		return domain.NewDomainError(err, "failed to update memory")
	}

	return nil
}

func (s *MemoryService) RegenerateEmbeddings(ctx context.Context, id string) (*models.Memory, error) {
	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result, err := s.embedding.Embed(ctx, memory.Content)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate embeddings")
	}

	embeddingsInfo := &models.EmbeddingsInfo{
		Model:      result.Model,
		Dimensions: result.Dimensions,
	}
	memory.SetEmbeddings(result.Embedding, embeddingsInfo)

	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) Delete(ctx context.Context, id string) error {
	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.memoryRepo.Delete(ctx, memory.ID); err != nil {
		return domain.NewDomainError(err, "failed to delete memory")
	}

	return nil
}

func (s *MemoryService) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	if query == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "search query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	result, err := s.embedding.Embed(ctx, query)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate query embeddings")
	}

	results, err := s.memoryRepo.SearchMemories(ctx, ports.MemorySearchOptions{
		Embedding:     result.Embedding,
		Limit:         limit,
		Threshold:     nil,
		IncludeScores: false,
	})
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMemorySearchFailed, "memory search failed")
	}

	memories := make([]*models.Memory, len(results))
	for i, r := range results {
		memories[i] = r.Memory
	}

	return memories, nil
}

func (s *MemoryService) SearchWithThreshold(ctx context.Context, query string, threshold float32, limit int) ([]*models.Memory, error) {
	if query == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "search query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	if threshold < 0 || threshold > 1 {
		threshold = 0.5
	}

	result, err := s.embedding.Embed(ctx, query)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate query embeddings")
	}

	results, err := s.memoryRepo.SearchMemories(ctx, ports.MemorySearchOptions{
		Embedding:     result.Embedding,
		Limit:         limit,
		Threshold:     &threshold,
		IncludeScores: false,
	})
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMemorySearchFailed, "memory search failed")
	}

	memories := make([]*models.Memory, len(results))
	for i, r := range results {
		memories[i] = r.Memory
	}

	return memories, nil
}

func (s *MemoryService) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]*models.Memory, error) {
	if len(embedding) == 0 {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "embedding cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	results, err := s.memoryRepo.SearchMemories(ctx, ports.MemorySearchOptions{
		Embedding:     embedding,
		Limit:         limit,
		Threshold:     nil,
		IncludeScores: false,
	})
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMemorySearchFailed, "memory search failed")
	}

	memories := make([]*models.Memory, len(results))
	for i, r := range results {
		memories[i] = r.Memory
	}

	return memories, nil
}

func (s *MemoryService) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	if len(tags) == 0 {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "at least one tag is required")
	}

	if limit <= 0 {
		limit = 50
	}

	memories, err := s.memoryRepo.GetByTags(ctx, tags, limit)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get memories by tags")
	}

	return memories, nil
}

func (s *MemoryService) SetImportance(ctx context.Context, id string, importance float32) (*models.Memory, error) {
	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	memory.SetImportance(importance)
	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) SetConfidence(ctx context.Context, id string, confidence float32) (*models.Memory, error) {
	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	memory.SetConfidence(confidence)
	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) SetUserRating(ctx context.Context, id string, rating int) (*models.Memory, error) {
	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	memory.SetUserRating(rating)
	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) AddTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	if tag == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tag cannot be empty")
	}

	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	memory.AddTag(tag)
	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) RemoveTag(ctx context.Context, id, tag string) (*models.Memory, error) {
	memory, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	memory.RemoveTag(tag)
	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryService) TrackUsage(ctx context.Context, memoryID, conversationID, messageID string, similarityScore float32) (*models.MemoryUsage, error) {
	if memoryID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "memory ID cannot be empty")
	}

	if conversationID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "conversation ID cannot be empty")
	}

	if messageID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "message ID cannot be empty")
	}

	if _, err := s.GetByID(ctx, memoryID); err != nil {
		return nil, err
	}

	id := s.idGenerator.GenerateMemoryUsageID()
	usage := models.NewMemoryUsage(id, conversationID, messageID, memoryID)
	usage.SimilarityScore = similarityScore

	if err := s.memoryUsageRepo.Create(ctx, usage); err != nil {
		return nil, domain.NewDomainError(err, "failed to track memory usage")
	}

	return usage, nil
}

func (s *MemoryService) GetUsageByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	if messageID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "message ID cannot be empty")
	}

	usage, err := s.memoryUsageRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get memory usage")
	}

	return usage, nil
}

func (s *MemoryService) GetUsageByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	if conversationID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "conversation ID cannot be empty")
	}

	usage, err := s.memoryUsageRepo.GetByConversation(ctx, conversationID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get memory usage")
	}

	return usage, nil
}

func (s *MemoryService) GetUsageByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error) {
	if memoryID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "memory ID cannot be empty")
	}

	usage, err := s.memoryUsageRepo.GetByMemory(ctx, memoryID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get memory usage")
	}

	return usage, nil
}

// CalculateImportance calculates the importance score for a memory based on:
// - Base importance (explicit importance set by user or system)
// - Recency decay (how old the memory is)
// - Access frequency (how often it's been used)
// - Average similarity (how relevant it typically is)
func (s *MemoryService) CalculateImportance(ctx context.Context, memoryID string) (float32, error) {
	memory, err := s.GetByID(ctx, memoryID)
	if err != nil {
		return 0, err
	}

	// Get usage statistics
	stats, err := s.memoryUsageRepo.GetUsageStats(ctx, memoryID)
	if err != nil {
		return 0, domain.NewDomainError(err, "failed to get usage statistics")
	}

	// Base importance (0.0 - 1.0)
	baseImportance := memory.Importance

	// Recency factor (0.0 - 1.0): newer memories get higher scores
	// Decay over 90 days
	daysSinceCreation := time.Since(memory.CreatedAt).Hours() / 24
	recencyFactor := float32(1.0)
	if daysSinceCreation > 0 {
		// Exponential decay: 0.5 after 30 days, 0.25 after 60 days
		recencyFactor = float32(math.Exp(-daysSinceCreation / 45.0))
	}

	// Access frequency factor (0.0 - 1.0): more frequently used memories get higher scores
	// Normalized to 0-1 range (saturates at 20 uses)
	accessFactor := float32(0.0)
	if stats.TotalUsageCount > 0 {
		accessFactor = float32(math.Min(float64(stats.TotalUsageCount)/20.0, 1.0))
	}

	// Recent access boost (0.0 - 0.5): memories used recently get a boost
	recentAccessBoost := float32(0.0)
	if stats.LastUsedAt != nil {
		daysSinceLastUse := time.Since(*stats.LastUsedAt).Hours() / 24
		if daysSinceLastUse < 7 {
			// Linear decay over 7 days
			recentAccessBoost = float32(0.5 * (1.0 - daysSinceLastUse/7.0))
		}
	}

	// Average similarity factor (0.0 - 0.5): memories that are typically more relevant get a boost
	similarityBoost := float32(0.0)
	if stats.AverageSimilarity > 0.7 {
		similarityBoost = (stats.AverageSimilarity - 0.7) * 0.5 / 0.3 // Scale 0.7-1.0 to 0-0.5
	}

	// Combine factors with weights:
	// - Base importance: 40%
	// - Recency: 20%
	// - Access frequency: 20%
	// - Recent access boost: 10%
	// - Similarity boost: 10%
	importance := (baseImportance * 0.4) +
		(recencyFactor * 0.2) +
		(accessFactor * 0.2) +
		recentAccessBoost +
		similarityBoost

	// Ensure the result is within 0-1 range
	if importance > 1.0 {
		importance = 1.0
	}
	if importance < 0.0 {
		importance = 0.0
	}

	return importance, nil
}

// UpdateImportanceScore calculates and updates the importance score for a memory
func (s *MemoryService) UpdateImportanceScore(ctx context.Context, memoryID string) (*models.Memory, error) {
	importance, err := s.CalculateImportance(ctx, memoryID)
	if err != nil {
		return nil, err
	}

	memory, err := s.GetByID(ctx, memoryID)
	if err != nil {
		return nil, err
	}

	memory.SetImportance(importance)
	if err := s.Update(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

// SearchWithScores performs a semantic search and returns memories with their similarity scores
func (s *MemoryService) SearchWithScores(ctx context.Context, query string, threshold float32, limit int) ([]*ports.MemorySearchResult, error) {
	if query == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "search query cannot be empty")
	}

	if limit <= 0 {
		limit = 5
	}

	if threshold < 0 || threshold > 1 {
		threshold = 0.7
	}

	result, err := s.embedding.Embed(ctx, query)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate query embeddings")
	}

	searchResults, err := s.memoryRepo.SearchMemories(ctx, ports.MemorySearchOptions{
		Embedding:     result.Embedding,
		Limit:         limit,
		Threshold:     &threshold,
		IncludeScores: true,
	})
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMemorySearchFailed, "memory search failed")
	}

	// Return the search results directly (already in MemorySearchResult format)
	return searchResults, nil
}

// SearchWithDynamicImportance performs a semantic search and re-ranks results based on calculated importance
func (s *MemoryService) SearchWithDynamicImportance(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	if query == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "search query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	// Get more results than needed to allow for re-ranking
	searchLimit := limit * 3
	if searchLimit > 100 {
		searchLimit = 100
	}

	result, err := s.embedding.Embed(ctx, query)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrEmbeddingsFailed, "failed to generate query embeddings")
	}

	searchResults, err := s.memoryRepo.SearchMemories(ctx, ports.MemorySearchOptions{
		Embedding:     result.Embedding,
		Limit:         searchLimit,
		Threshold:     nil,
		IncludeScores: true,
	})
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMemorySearchFailed, "memory search failed")
	}

	// Calculate dynamic importance for each memory and combine with similarity score
	type scoredMemory struct {
		memory     *models.Memory
		finalScore float32
	}

	scoredMemories := make([]scoredMemory, 0, len(searchResults))
	for _, searchResult := range searchResults {
		dynamicImportance, err := s.CalculateImportance(ctx, searchResult.Memory.ID)
		if err != nil {
			// If we can't calculate importance, use the stored importance
			dynamicImportance = searchResult.Memory.Importance
		}

		// Combine similarity (70%) and dynamic importance (30%)
		finalScore := (searchResult.Similarity * 0.7) + (dynamicImportance * 0.3)

		scoredMemories = append(scoredMemories, scoredMemory{
			memory:     searchResult.Memory,
			finalScore: finalScore,
		})
	}

	// Sort by final score (descending)
	sort.Slice(scoredMemories, func(i, j int) bool {
		return scoredMemories[i].finalScore > scoredMemories[j].finalScore
	})

	// Take top N results
	if len(scoredMemories) > limit {
		scoredMemories = scoredMemories[:limit]
	}

	// Extract memories
	memories := make([]*models.Memory, len(scoredMemories))
	for i, sm := range scoredMemories {
		memories[i] = sm.memory
	}

	return memories, nil
}
