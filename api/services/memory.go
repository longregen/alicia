package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
)

// EmbeddingService generates embeddings for text.
type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// MemoryService handles memory operations.
type MemoryService struct {
	store    *store.Store
	embedder EmbeddingService
}

// NewMemoryService creates a new memory service.
func NewMemoryService(s *store.Store, embedder EmbeddingService) *MemoryService {
	return &MemoryService{store: s, embedder: embedder}
}

// CreateMemory creates a new memory with optional embedding.
func (svc *MemoryService) CreateMemory(ctx context.Context, content string, sourceMsgID *string) (*domain.Memory, error) {
	mem := &domain.Memory{
		ID:          store.NewMemoryID(),
		Content:     content,
		Importance:  0.5,
		SourceMsgID: sourceMsgID,
		Tags:        []string{},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Generate embedding if embedder is available
	if svc.embedder != nil {
		embedding, err := svc.embedder.Embed(ctx, content)
		if err != nil {
			slog.Warn("embedding generation failed for memory", "memory_id", mem.ID, "error", err)
			// Continue without embedding - memory can still be created
		} else {
			mem.Embedding = embedding
		}
	}

	if err := svc.store.CreateMemory(ctx, mem); err != nil {
		return nil, err
	}
	return mem, nil
}

// GetMemory retrieves a memory by ID.
func (svc *MemoryService) GetMemory(ctx context.Context, id string) (*domain.Memory, error) {
	return svc.store.GetMemory(ctx, id)
}

// UpdateMemory updates a memory.
func (svc *MemoryService) UpdateMemory(ctx context.Context, mem *domain.Memory) error {
	return svc.store.UpdateMemory(ctx, mem)
}

// DeleteMemory soft-deletes a memory with optional reason.
func (svc *MemoryService) DeleteMemory(ctx context.Context, id string, reason *string) error {
	return svc.store.DeleteMemory(ctx, id, reason)
}

// ListMemories returns all non-archived memories with total count.
func (svc *MemoryService) ListMemories(ctx context.Context, limit, offset int) ([]*domain.Memory, int, error) {
	return svc.store.ListMemories(ctx, limit, offset)
}

// SearchMemories searches memories by semantic similarity.
func (svc *MemoryService) SearchMemories(ctx context.Context, query string, limit int) ([]*domain.Memory, error) {
	if svc.embedder == nil {
		// Fall back to listing if no embedder
		mems, _, err := svc.store.ListMemories(ctx, limit, 0)
		return mems, err
	}

	embedding, err := svc.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	return svc.store.SearchMemories(ctx, embedding, limit, 0.5) // 0.5 threshold
}

// GetMemoriesByTags returns memories matching tags.
func (svc *MemoryService) GetMemoriesByTags(ctx context.Context, tags []string, limit int) ([]*domain.Memory, error) {
	return svc.store.GetMemoriesByTags(ctx, tags, limit)
}

// TrackUsage records that a memory was retrieved for a message.
func (svc *MemoryService) TrackUsage(ctx context.Context, memoryID, convID, msgID string, similarity float32) (*domain.MemoryUse, error) {
	use := &domain.MemoryUse{
		ID:             store.NewMemoryUseID(),
		MemoryID:       memoryID,
		MessageID:      msgID,
		ConversationID: convID,
		Similarity:     similarity,
		CreatedAt:      time.Now().UTC(),
	}

	if err := svc.store.CreateMemoryUse(ctx, use); err != nil {
		return nil, err
	}
	return use, nil
}

// GetMemoryUse retrieves a memory use by ID.
func (svc *MemoryService) GetMemoryUse(ctx context.Context, id string) (*domain.MemoryUse, error) {
	return svc.store.GetMemoryUse(ctx, id)
}

// GetUsesByMessage returns memory uses for a message.
func (svc *MemoryService) GetUsesByMessage(ctx context.Context, messageID string) ([]*domain.MemoryUse, error) {
	return svc.store.GetMemoryUsesByMessage(ctx, messageID)
}

// GetUsesByConversation returns memory uses for a conversation.
func (svc *MemoryService) GetUsesByConversation(ctx context.Context, conversationID string) ([]*domain.MemoryUse, error) {
	return svc.store.GetMemoryUsesByConversation(ctx, conversationID)
}

// PinMemory pins or unpins a memory.
func (svc *MemoryService) PinMemory(ctx context.Context, id string, pinned bool) error {
	mem, err := svc.store.GetMemory(ctx, id)
	if err != nil {
		return err
	}
	mem.Pinned = pinned
	return svc.store.UpdateMemory(ctx, mem)
}

// ArchiveMemory archives a memory.
func (svc *MemoryService) ArchiveMemory(ctx context.Context, id string) error {
	mem, err := svc.store.GetMemory(ctx, id)
	if err != nil {
		return err
	}
	mem.Archived = true
	return svc.store.UpdateMemory(ctx, mem)
}

// SetImportance sets a memory's importance score.
func (svc *MemoryService) SetImportance(ctx context.Context, id string, importance float32) error {
	mem, err := svc.store.GetMemory(ctx, id)
	if err != nil {
		return err
	}
	mem.Importance = importance
	return svc.store.UpdateMemory(ctx, mem)
}

// AddTag adds a tag to a memory.
func (svc *MemoryService) AddTag(ctx context.Context, id, tag string) error {
	mem, err := svc.store.GetMemory(ctx, id)
	if err != nil {
		return err
	}
	for _, t := range mem.Tags {
		if t == tag {
			return nil // Already has tag
		}
	}
	mem.Tags = append(mem.Tags, tag)
	return svc.store.UpdateMemory(ctx, mem)
}

// RemoveTag removes a tag from a memory.
func (svc *MemoryService) RemoveTag(ctx context.Context, id, tag string) error {
	mem, err := svc.store.GetMemory(ctx, id)
	if err != nil {
		return err
	}
	tags := make([]string, 0, len(mem.Tags))
	for _, t := range mem.Tags {
		if t != tag {
			tags = append(tags, t)
		}
	}
	mem.Tags = tags
	return svc.store.UpdateMemory(ctx, mem)
}
