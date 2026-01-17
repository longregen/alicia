package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// Integration tests for TrainingExampleRepository
// These require a real PostgreSQL instance with the test database

func TestTrainingExampleRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewTrainingExampleRepository(pool, idGen)

	example := &models.TrainingExample{
		ID:         idGen.GenerateTrainingExampleID(),
		TaskType:   models.TaskTypeToolSelection,
		IsPositive: true,
		Inputs: map[string]any{
			"user_message":    "Search for Python tutorials",
			"available_tools": `[{"name":"search"}]`,
		},
		Outputs: map[string]any{
			"selected_tool": "search",
			"arguments": map[string]any{
				"query": "Python tutorials",
			},
		},
		Source:    "vote",
		CreatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), example)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify we can retrieve it
	retrieved, err := repo.GetByID(context.Background(), example.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != example.ID {
		t.Errorf("expected ID %s, got %s", example.ID, retrieved.ID)
	}

	if retrieved.TaskType != example.TaskType {
		t.Errorf("expected task type %s, got %s", example.TaskType, retrieved.TaskType)
	}

	if retrieved.IsPositive != example.IsPositive {
		t.Errorf("expected is_positive %v, got %v", example.IsPositive, retrieved.IsPositive)
	}

	if retrieved.Source != example.Source {
		t.Errorf("expected source %s, got %s", example.Source, retrieved.Source)
	}

	if retrieved.Inputs == nil {
		t.Fatal("expected inputs to be not nil")
	}

	if userMsg, ok := retrieved.Inputs["user_message"].(string); !ok || userMsg != "Search for Python tutorials" {
		t.Errorf("expected user_message 'Search for Python tutorials', got %v", retrieved.Inputs["user_message"])
	}

	if retrieved.Outputs == nil {
		t.Fatal("expected outputs to be not nil")
	}

	if tool, ok := retrieved.Outputs["selected_tool"].(string); !ok || tool != "search" {
		t.Errorf("expected selected_tool 'search', got %v", retrieved.Outputs["selected_tool"])
	}
}

func TestTrainingExampleRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewTrainingExampleRepository(pool, idGen)

	// Create an example
	example := &models.TrainingExample{
		ID:         idGen.GenerateTrainingExampleID(),
		TaskType:   models.TaskTypeMemorySelection,
		IsPositive: false,
		Inputs: map[string]any{
			"user_message": "What did I tell you about my schedule?",
		},
		Outputs: map[string]any{
			"selected_memory_id": "mem_123",
		},
		VoteMetadata: &models.VoteMetadata{
			QuickFeedback: "wrong_context",
			VoteValue:     models.VoteValueDown,
		},
		Source:    "vote",
		CreatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), example)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Retrieve it
	retrieved, err := repo.GetByID(context.Background(), example.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.VoteMetadata == nil {
		t.Fatal("expected vote metadata to be not nil")
	}

	if retrieved.VoteMetadata.QuickFeedback != "wrong_context" {
		t.Errorf("expected quick feedback 'wrong_context', got %s", retrieved.VoteMetadata.QuickFeedback)
	}

	if retrieved.VoteMetadata.VoteValue != models.VoteValueDown {
		t.Errorf("expected vote value %d, got %d", models.VoteValueDown, retrieved.VoteMetadata.VoteValue)
	}
}

func TestTrainingExampleRepository_ListByTaskType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewTrainingExampleRepository(pool, idGen)

	// Create multiple examples of different types
	for i := 0; i < 3; i++ {
		example := &models.TrainingExample{
			ID:         idGen.GenerateTrainingExampleID(),
			TaskType:   models.TaskTypeToolSelection,
			IsPositive: true,
			Inputs:     map[string]any{"test": "data"},
			Outputs:    map[string]any{"result": "output"},
			Source:     "synthetic",
			CreatedAt:  time.Now(),
		}
		if err := repo.Create(context.Background(), example); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		example := &models.TrainingExample{
			ID:         idGen.GenerateTrainingExampleID(),
			TaskType:   models.TaskTypeMemorySelection,
			IsPositive: true,
			Inputs:     map[string]any{"test": "data"},
			Outputs:    map[string]any{"result": "output"},
			Source:     "synthetic",
			CreatedAt:  time.Now(),
		}
		if err := repo.Create(context.Background(), example); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List tool selection examples
	examples, err := repo.ListByTaskType(context.Background(), models.TaskTypeToolSelection, 10, 0)
	if err != nil {
		t.Fatalf("ListByTaskType failed: %v", err)
	}

	if len(examples) != 3 {
		t.Errorf("expected 3 tool selection examples, got %d", len(examples))
	}

	// List memory selection examples
	examples, err = repo.ListByTaskType(context.Background(), models.TaskTypeMemorySelection, 10, 0)
	if err != nil {
		t.Fatalf("ListByTaskType failed: %v", err)
	}

	if len(examples) != 2 {
		t.Errorf("expected 2 memory selection examples, got %d", len(examples))
	}

	// Test pagination
	examples, err = repo.ListByTaskType(context.Background(), models.TaskTypeToolSelection, 2, 0)
	if err != nil {
		t.Fatalf("ListByTaskType with limit failed: %v", err)
	}

	if len(examples) != 2 {
		t.Errorf("expected 2 examples with limit=2, got %d", len(examples))
	}

	// Test offset
	examples, err = repo.ListByTaskType(context.Background(), models.TaskTypeToolSelection, 10, 2)
	if err != nil {
		t.Fatalf("ListByTaskType with offset failed: %v", err)
	}

	if len(examples) != 1 {
		t.Errorf("expected 1 example with offset=2, got %d", len(examples))
	}
}

func TestTrainingExampleRepository_CountByTaskType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewTrainingExampleRepository(pool, idGen)

	// Create examples
	for i := 0; i < 5; i++ {
		example := &models.TrainingExample{
			ID:         idGen.GenerateTrainingExampleID(),
			TaskType:   models.TaskTypeToolSelection,
			IsPositive: i%2 == 0, // Alternate positive/negative
			Inputs:     map[string]any{"test": "data"},
			Outputs:    map[string]any{"result": "output"},
			Source:     "vote",
			CreatedAt:  time.Now(),
		}
		if err := repo.Create(context.Background(), example); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Count total
	count, err := repo.CountByTaskType(context.Background(), models.TaskTypeToolSelection)
	if err != nil {
		t.Fatalf("CountByTaskType failed: %v", err)
	}

	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}

	// Count positive only
	positiveCount, err := repo.CountPositiveByTaskType(context.Background(), models.TaskTypeToolSelection)
	if err != nil {
		t.Fatalf("CountPositiveByTaskType failed: %v", err)
	}

	if positiveCount != 3 {
		t.Errorf("expected positive count 3, got %d", positiveCount)
	}
}

func TestTrainingExampleRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewTrainingExampleRepository(pool, idGen)

	// Create an example
	example := &models.TrainingExample{
		ID:         idGen.GenerateTrainingExampleID(),
		TaskType:   models.TaskTypeToolSelection,
		IsPositive: true,
		Inputs:     map[string]any{"test": "data"},
		Outputs:    map[string]any{"result": "output"},
		Source:     "synthetic",
		CreatedAt:  time.Now(),
	}

	err := repo.Create(context.Background(), example)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it exists
	_, err = repo.GetByID(context.Background(), example.ID)
	if err != nil {
		t.Fatalf("GetByID failed before delete: %v", err)
	}

	// Delete it
	err = repo.Delete(context.Background(), example.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone (soft deleted)
	_, err = repo.GetByID(context.Background(), example.ID)
	if err == nil {
		t.Error("expected error when getting deleted example, got nil")
	}

	// Count should exclude deleted
	count, err := repo.CountByTaskType(context.Background(), models.TaskTypeToolSelection)
	if err != nil {
		t.Fatalf("CountByTaskType failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count 0 after delete, got %d", count)
	}
}

func TestTrainingExampleRepository_DeleteByVoteID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewTrainingExampleRepository(pool, idGen)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)

	ctx := context.Background()

	// Create conversation and message for the vote
	conv := models.NewConversation("ac_vote_delete_test", "test-user", "Vote Delete Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	msg := &models.Message{
		ID:               "msg_vote_delete_test",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleUser,
		Contents:         "Test message",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, msg)
	if err != nil {
		t.Fatalf("Create message failed: %v", err)
	}

	// Create a vote to link to
	vote := &models.Vote{
		ID:         "vote_test123",
		MessageID:  msg.ID,
		TargetType: models.VoteTargetMessage,
		TargetID:   msg.ID,
		Value:      models.VoteValueUp,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = voteRepo.Create(ctx, vote)
	if err != nil {
		t.Fatalf("Create vote failed: %v", err)
	}

	voteID := vote.ID

	// Create multiple examples linked to same vote
	for i := 0; i < 3; i++ {
		example := &models.TrainingExample{
			ID:         idGen.GenerateTrainingExampleID(),
			TaskType:   models.TaskTypeToolSelection,
			VoteID:     &voteID,
			IsPositive: true,
			Inputs:     map[string]any{"test": "data"},
			Outputs:    map[string]any{"result": "output"},
			Source:     "vote",
			CreatedAt:  time.Now(),
		}
		if err := repo.Create(ctx, example); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Verify count before delete
	count, err := repo.CountByTaskType(context.Background(), models.TaskTypeToolSelection)
	if err != nil {
		t.Fatalf("CountByTaskType failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected count 3 before delete, got %d", count)
	}

	// Delete by vote ID
	err = repo.DeleteByVoteID(context.Background(), voteID)
	if err != nil {
		t.Fatalf("DeleteByVoteID failed: %v", err)
	}

	// Verify all are deleted
	count, err = repo.CountByTaskType(context.Background(), models.TaskTypeToolSelection)
	if err != nil {
		t.Fatalf("CountByTaskType failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count 0 after DeleteByVoteID, got %d", count)
	}
}

// newTestIDGenerator creates a test ID generator
func newTestIDGenerator() *testIDGenerator {
	return &testIDGenerator{
		counter: 0,
	}
}

type testIDGenerator struct {
	counter int
}

func (g *testIDGenerator) GenerateTrainingExampleID() string {
	g.counter++
	return fmt.Sprintf("gte_test_%d_%d", time.Now().UnixNano(), g.counter)
}

func (g *testIDGenerator) GenerateSystemPromptVersionID() string {
	g.counter++
	return fmt.Sprintf("spv_test_%d_%d", time.Now().UnixNano(), g.counter)
}

// Implement other required methods (not used in these tests but required by interface)
func (g *testIDGenerator) GenerateConversationID() string { return "ac_test" }
func (g *testIDGenerator) GenerateMessageID() string      { return "msg_test" }
func (g *testIDGenerator) GenerateSentenceID() string     { return "sent_test" }
func (g *testIDGenerator) GenerateAudioID() string        { return "audio_test" }
func (g *testIDGenerator) GenerateMemoryID() string       { return "mem_test" }
func (g *testIDGenerator) GenerateMemoryUsageID() string  { return "mu_test" }
func (g *testIDGenerator) GenerateToolID() string         { return "tool_test" }
func (g *testIDGenerator) GenerateToolUseID() string      { return "tu_test" }
func (g *testIDGenerator) GenerateReasoningStepID() string {
	return "rs_test"
}
func (g *testIDGenerator) GenerateCommentaryID() string       { return "comm_test" }
func (g *testIDGenerator) GenerateMetaID() string             { return "meta_test" }
func (g *testIDGenerator) GenerateMCPServerID() string        { return "amcp_test" }
func (g *testIDGenerator) GenerateVoteID() string             { return "av_test" }
func (g *testIDGenerator) GenerateNoteID() string             { return "an_test" }
func (g *testIDGenerator) GenerateSessionStatsID() string     { return "ass_test" }
func (g *testIDGenerator) GenerateOptimizationRunID() string  { return "aor_test" }
func (g *testIDGenerator) GeneratePromptCandidateID() string  { return "apc_test" }
func (g *testIDGenerator) GeneratePromptEvaluationID() string { return "ape_test" }
func (g *testIDGenerator) GenerateRequestID() string          { return "areq_test" }
