package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateMemory inserts a new memory.
func (s *Store) CreateMemory(ctx context.Context, mem *domain.Memory) error {
	query := `
		INSERT INTO memories (id, content, embedding, importance, pinned, archived, source_msg_id, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := s.conn(ctx).Exec(ctx, query,
		mem.ID, mem.Content, mem.Embedding, mem.Importance,
		mem.Pinned, mem.Archived, mem.SourceMsgID, mem.Tags,
		mem.CreatedAt, mem.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create memory: %w", err)
	}
	return nil
}

// GetMemory retrieves a memory by ID.
func (s *Store) GetMemory(ctx context.Context, id string) (*domain.Memory, error) {
	query := `
		SELECT id, content, importance, pinned, archived, source_msg_id, tags, created_at, updated_at
		FROM memories
		WHERE id = $1 AND deleted_at IS NULL`

	mem := &domain.Memory{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&mem.ID, &mem.Content, &mem.Importance,
		&mem.Pinned, &mem.Archived, &mem.SourceMsgID, &mem.Tags,
		&mem.CreatedAt, &mem.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get memory: %w", err)
	}
	return mem, nil
}

// UpdateMemory updates a memory.
func (s *Store) UpdateMemory(ctx context.Context, mem *domain.Memory) error {
	query := `
		UPDATE memories
		SET content = $2, importance = $3, pinned = $4, archived = $5, tags = $6, updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL`

	mem.UpdatedAt = time.Now().UTC()
	_, err := s.conn(ctx).Exec(ctx, query,
		mem.ID, mem.Content, mem.Importance,
		mem.Pinned, mem.Archived, mem.Tags, mem.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

// DeleteMemory soft-deletes a memory with optional reason.
func (s *Store) DeleteMemory(ctx context.Context, id string, reason *string) error {
	query := `UPDATE memories SET deleted_at = $2, deleted_reason = $3 WHERE id = $1 AND deleted_at IS NULL`
	result, err := s.conn(ctx).Exec(ctx, query, id, time.Now().UTC(), reason)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListMemories returns all non-archived memories with total count.
func (s *Store) ListMemories(ctx context.Context, limit, offset int) ([]*domain.Memory, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM memories WHERE deleted_at IS NULL AND archived = false`
	var total int
	if err := s.conn(ctx).QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count memories: %w", err)
	}

	query := `
		SELECT id, content, importance, pinned, archived, source_msg_id, tags, created_at, updated_at
		FROM memories
		WHERE deleted_at IS NULL AND archived = false
		ORDER BY importance DESC, created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	mems, err := scanMemories(rows)
	if err != nil {
		return nil, 0, err
	}
	return mems, total, nil
}

// SearchMemories searches memories by embedding similarity.
func (s *Store) SearchMemories(ctx context.Context, embedding []float32, limit int, threshold float32) ([]*domain.Memory, error) {
	query := `
		SELECT id, content, importance, pinned, archived, source_msg_id, tags, created_at, updated_at
		FROM memories
		WHERE deleted_at IS NULL AND archived = false
		  AND embedding <=> $1 < $3
		ORDER BY embedding <=> $1
		LIMIT $2`

	rows, err := s.conn(ctx).Query(ctx, query, embedding, limit, 1-threshold)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// GetMemoriesByTags returns memories matching any of the given tags.
func (s *Store) GetMemoriesByTags(ctx context.Context, tags []string, limit int) ([]*domain.Memory, error) {
	query := `
		SELECT id, content, importance, pinned, archived, source_msg_id, tags, created_at, updated_at
		FROM memories
		WHERE deleted_at IS NULL AND archived = false AND tags && $1
		ORDER BY importance DESC
		LIMIT $2`

	rows, err := s.conn(ctx).Query(ctx, query, tags, limit)
	if err != nil {
		return nil, fmt.Errorf("get memories by tags: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

func scanMemories(rows pgx.Rows) ([]*domain.Memory, error) {
	var mems []*domain.Memory
	for rows.Next() {
		mem := &domain.Memory{}
		if err := rows.Scan(
			&mem.ID, &mem.Content, &mem.Importance,
			&mem.Pinned, &mem.Archived, &mem.SourceMsgID, &mem.Tags,
			&mem.CreatedAt, &mem.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		mems = append(mems, mem)
	}
	return mems, rows.Err()
}

// --- Memory Uses ---

// CreateMemoryUse records that a memory was retrieved for a message.
func (s *Store) CreateMemoryUse(ctx context.Context, use *domain.MemoryUse) error {
	query := `
		INSERT INTO memory_uses (id, memory_id, message_id, conversation_id, similarity, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.conn(ctx).Exec(ctx, query,
		use.ID, use.MemoryID, use.MessageID,
		use.ConversationID, use.Similarity, use.CreatedAt)
	if err != nil {
		return fmt.Errorf("create memory use: %w", err)
	}
	return nil
}

// GetMemoryUse retrieves a memory use by ID.
func (s *Store) GetMemoryUse(ctx context.Context, id string) (*domain.MemoryUse, error) {
	query := `
		SELECT id, memory_id, message_id, conversation_id, similarity, created_at
		FROM memory_uses
		WHERE id = $1`

	use := &domain.MemoryUse{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&use.ID, &use.MemoryID, &use.MessageID,
		&use.ConversationID, &use.Similarity, &use.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get memory use: %w", err)
	}
	return use, nil
}

// GetMemoryUsesByMessage returns memory uses for a message.
func (s *Store) GetMemoryUsesByMessage(ctx context.Context, messageID string) ([]*domain.MemoryUse, error) {
	query := `
		SELECT id, memory_id, message_id, conversation_id, similarity, created_at
		FROM memory_uses
		WHERE message_id = $1
		ORDER BY similarity DESC`

	rows, err := s.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("get memory uses: %w", err)
	}
	defer rows.Close()

	var uses []*domain.MemoryUse
	for rows.Next() {
		u := &domain.MemoryUse{}
		if err := rows.Scan(&u.ID, &u.MemoryID, &u.MessageID, &u.ConversationID, &u.Similarity, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory use: %w", err)
		}
		uses = append(uses, u)
	}
	return uses, rows.Err()
}

// GetMemoryUsesByConversation returns memory uses for a conversation.
func (s *Store) GetMemoryUsesByConversation(ctx context.Context, conversationID string) ([]*domain.MemoryUse, error) {
	query := `
		SELECT id, memory_id, message_id, conversation_id, similarity, created_at
		FROM memory_uses
		WHERE conversation_id = $1
		ORDER BY created_at DESC`

	rows, err := s.conn(ctx).Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get memory uses: %w", err)
	}
	defer rows.Close()

	var uses []*domain.MemoryUse
	for rows.Next() {
		u := &domain.MemoryUse{}
		if err := rows.Scan(&u.ID, &u.MemoryID, &u.MessageID, &u.ConversationID, &u.Similarity, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory use: %w", err)
		}
		uses = append(uses, u)
	}
	return uses, rows.Err()
}
