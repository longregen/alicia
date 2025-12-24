//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// Fixtures provides common test data setup
type Fixtures struct {
	db *TestDB
}

// NewFixtures creates a new fixtures helper
func NewFixtures(db *TestDB) *Fixtures {
	return &Fixtures{db: db}
}

// CreateConversation inserts a conversation directly into the database
func (f *Fixtures) CreateConversation(ctx context.Context, t *testing.T, id, title string) *models.Conversation {
	t.Helper()

	now := time.Now()
	query := `
		INSERT INTO alicia_conversations (id, title, status, preferences, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, title, "active", `{"enable_memory": true}`, now, now)
	if err != nil {
		t.Fatalf("failed to create conversation fixture: %v", err)
	}

	return &models.Conversation{
		ID:     id,
		Title:  title,
		Status: models.ConversationStatusActive,
		Preferences: &models.ConversationPreferences{
			EnableMemory: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateMessage inserts a message directly into the database
func (f *Fixtures) CreateMessage(ctx context.Context, t *testing.T, id, conversationID string, role models.MessageRole, content string, sequence int) *models.Message {
	t.Helper()

	now := time.Now()
	query := `
		INSERT INTO alicia_messages (id, conversation_id, sequence_number, message_role, contents, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, conversationID, sequence, string(role), content, now, now)
	if err != nil {
		t.Fatalf("failed to create message fixture: %v", err)
	}

	return &models.Message{
		ID:             id,
		ConversationID: conversationID,
		SequenceNumber: sequence,
		Role:           role,
		Contents:       content,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateMemory inserts a memory directly into the database
func (f *Fixtures) CreateMemory(ctx context.Context, t *testing.T, id, content string) *models.Memory {
	t.Helper()

	now := time.Now()
	query := `
		INSERT INTO alicia_memory (id, content, importance, confidence, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, content, 0.5, 1.0, now, now)
	if err != nil {
		t.Fatalf("failed to create memory fixture: %v", err)
	}

	return &models.Memory{
		ID:         id,
		Content:    content,
		Importance: 0.5,
		Confidence: 1.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateMemoryWithEmbedding inserts a memory with embeddings
func (f *Fixtures) CreateMemoryWithEmbedding(ctx context.Context, t *testing.T, id, content string, embeddings []float32) *models.Memory {
	t.Helper()

	now := time.Now()

	// Convert embeddings to pgvector format
	query := `
		INSERT INTO alicia_memory (id, content, embeddings, importance, confidence, created_at, updated_at)
		VALUES ($1, $2, $3::vector, $4, $5, $6, $7)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, content, embeddings, 0.5, 1.0, now, now)
	if err != nil {
		t.Fatalf("failed to create memory with embedding fixture: %v", err)
	}

	return &models.Memory{
		ID:         id,
		Content:    content,
		Embeddings: embeddings,
		Importance: 0.5,
		Confidence: 1.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateMemoryUsage inserts a memory usage record
func (f *Fixtures) CreateMemoryUsage(ctx context.Context, t *testing.T, id, conversationID, messageID, memoryID string) *models.MemoryUsage {
	t.Helper()

	now := time.Now()
	query := `
		INSERT INTO alicia_memory_used (id, conversation_id, message_id, memory_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, conversationID, messageID, memoryID, now, now)
	if err != nil {
		t.Fatalf("failed to create memory usage fixture: %v", err)
	}

	return &models.MemoryUsage{
		ID:             id,
		ConversationID: conversationID,
		MessageID:      messageID,
		MemoryID:       memoryID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateTool inserts a tool directly into the database
func (f *Fixtures) CreateTool(ctx context.Context, t *testing.T, id, name, description string, schema map[string]any) *models.Tool {
	t.Helper()

	now := time.Now()
	query := `
		INSERT INTO alicia_tools (id, name, description, schema, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, name, description, schema, true, now, now)
	if err != nil {
		t.Fatalf("failed to create tool fixture: %v", err)
	}

	return &models.Tool{
		ID:          id,
		Name:        name,
		Description: description,
		Schema:      schema,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CreateToolUse inserts a tool use record
func (f *Fixtures) CreateToolUse(ctx context.Context, t *testing.T, id, messageID, toolName string, args map[string]any, sequence int) *models.ToolUse {
	t.Helper()

	now := time.Now()
	query := `
		INSERT INTO alicia_tool_uses (id, message_id, tool_name, tool_arguments, status, sequence_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := f.db.Pool.Exec(ctx, query, id, messageID, toolName, args, "pending", sequence, now, now)
	if err != nil {
		t.Fatalf("failed to create tool use fixture: %v", err)
	}

	return &models.ToolUse{
		ID:             id,
		MessageID:      messageID,
		ToolName:       toolName,
		ToolArguments:  args,
		Status:         models.ToolStatusPending,
		SequenceNumber: sequence,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// GenerateEmbedding creates a dummy embedding vector
func (f *Fixtures) GenerateEmbedding(dimensions int) []float32 {
	embeddings := make([]float32, dimensions)
	for i := range embeddings {
		embeddings[i] = float32(i) / float32(dimensions)
	}
	return embeddings
}
