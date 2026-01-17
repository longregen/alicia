package services

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock repositories for training set builder tests

type mockVoteRepository struct {
	toolUseVotes          []*ports.VoteWithToolContext
	memoryUsageVotes      []*ports.VoteWithMemoryContext
	memoryExtractionVotes []*ports.VoteWithExtractionContext
	targetTypeCounts      map[string]int
}

func newMockVoteRepository() *mockVoteRepository {
	return &mockVoteRepository{
		toolUseVotes:          []*ports.VoteWithToolContext{},
		memoryUsageVotes:      []*ports.VoteWithMemoryContext{},
		memoryExtractionVotes: []*ports.VoteWithExtractionContext{},
		targetTypeCounts:      make(map[string]int),
	}
}

func (m *mockVoteRepository) GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithToolContext, error) {
	if len(m.toolUseVotes) <= limit {
		return m.toolUseVotes, nil
	}
	return m.toolUseVotes[:limit], nil
}

func (m *mockVoteRepository) GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	if len(m.memoryUsageVotes) <= limit {
		return m.memoryUsageVotes, nil
	}
	return m.memoryUsageVotes[:limit], nil
}

func (m *mockVoteRepository) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	if len(m.memoryUsageVotes) <= limit {
		return m.memoryUsageVotes, nil
	}
	return m.memoryUsageVotes[:limit], nil
}

func (m *mockVoteRepository) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithExtractionContext, error) {
	if len(m.memoryExtractionVotes) <= limit {
		return m.memoryExtractionVotes, nil
	}
	return m.memoryExtractionVotes[:limit], nil
}

func (m *mockVoteRepository) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	if count, ok := m.targetTypeCounts[targetType]; ok {
		return count, nil
	}
	return 0, nil
}

// Implement other methods to satisfy interface
func (m *mockVoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	return nil
}

func (m *mockVoteRepository) Delete(ctx context.Context, targetType, targetID string) error {
	return nil
}

func (m *mockVoteRepository) GetByTarget(ctx context.Context, targetType, targetID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	return nil, nil
}

func (m *mockVoteRepository) GetAggregates(ctx context.Context, targetType, targetID string) (*models.VoteAggregates, error) {
	return nil, nil
}

type mockTrainingExampleRepository struct {
	examples   map[string]*models.TrainingExample
	taskCounts map[string]int
}

func newMockTrainingExampleRepository() *mockTrainingExampleRepository {
	return &mockTrainingExampleRepository{
		examples:   make(map[string]*models.TrainingExample),
		taskCounts: make(map[string]int),
	}
}

func (m *mockTrainingExampleRepository) Create(ctx context.Context, example *models.TrainingExample) error {
	m.examples[example.ID] = example
	m.taskCounts[example.TaskType]++
	return nil
}

func (m *mockTrainingExampleRepository) GetByID(ctx context.Context, id string) (*models.TrainingExample, error) {
	if example, ok := m.examples[id]; ok {
		return example, nil
	}
	return nil, errNotFound
}

func (m *mockTrainingExampleRepository) ListByTaskType(ctx context.Context, taskType string, limit, offset int) ([]*models.TrainingExample, error) {
	result := []*models.TrainingExample{}
	for _, ex := range m.examples {
		if ex.TaskType == taskType {
			result = append(result, ex)
		}
	}
	return result, nil
}

func (m *mockTrainingExampleRepository) CountByTaskType(ctx context.Context, taskType string) (int, error) {
	if count, ok := m.taskCounts[taskType]; ok {
		return count, nil
	}
	return 0, nil
}

func (m *mockTrainingExampleRepository) CountPositiveByTaskType(ctx context.Context, taskType string) (int, error) {
	count := 0
	for _, ex := range m.examples {
		if ex.TaskType == taskType && ex.IsPositive {
			count++
		}
	}
	return count, nil
}

func (m *mockTrainingExampleRepository) Delete(ctx context.Context, id string) error {
	if _, ok := m.examples[id]; !ok {
		return errNotFound
	}
	delete(m.examples, id)
	return nil
}

func (m *mockTrainingExampleRepository) DeleteByVoteID(ctx context.Context, voteID string) error {
	for id, ex := range m.examples {
		if ex.VoteID != nil && *ex.VoteID == voteID {
			delete(m.examples, id)
		}
	}
	return nil
}

type mockToolRepository struct {
	tools []*models.Tool
}

func newMockToolRepository() *mockToolRepository {
	return &mockToolRepository{
		tools: []*models.Tool{},
	}
}

func (m *mockToolRepository) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	return m.tools, nil
}

func (m *mockToolRepository) Create(ctx context.Context, tool *models.Tool) error {
	return nil
}

func (m *mockToolRepository) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolRepository) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	return nil, nil
}

func (m *mockToolRepository) Update(ctx context.Context, tool *models.Tool) error {
	return nil
}

func (m *mockToolRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockToolRepository) List(ctx context.Context) ([]*models.Tool, error) {
	return m.tools, nil
}

func (m *mockToolRepository) ListAll(ctx context.Context) ([]*models.Tool, error) {
	return m.tools, nil
}

func (m *mockToolRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	return nil
}

type mockMemoryRepository struct{}

func newMockMemoryRepository() *mockMemoryRepository {
	return &mockMemoryRepository{}
}

func (m *mockMemoryRepository) Create(ctx context.Context, memory *models.Memory) error {
	return nil
}

func (m *mockMemoryRepository) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryRepository) List(ctx context.Context, limit, offset int) ([]*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryRepository) Update(ctx context.Context, memory *models.Memory) error {
	return nil
}

func (m *mockMemoryRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryRepository) Search(ctx context.Context, query string, limit int) ([]*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryRepository) RecordUsage(ctx context.Context, usage *models.MemoryUsage) error {
	return nil
}

func (m *mockMemoryRepository) GetUsagesByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryRepository) GetUsagesByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error) {
	return nil, nil
}

func (m *mockMemoryRepository) SearchMemories(ctx context.Context, opts ports.MemorySearchOptions) ([]*ports.MemorySearchResult, error) {
	return nil, nil
}

func (m *mockMemoryRepository) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	return nil, nil
}

func (m *mockMemoryRepository) Pin(ctx context.Context, id string, pinned bool) error {
	return nil
}

func (m *mockMemoryRepository) Archive(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryRepository) DeleteByConversationID(ctx context.Context, conversationID string) error {
	return nil
}

// Tests

func TestToolUseVoteToExample_Upvote(t *testing.T) {
	voteRepo := newMockVoteRepository()
	trainingRepo := newMockTrainingExampleRepository()
	toolRepo := newMockToolRepository()
	memoryRepo := newMockMemoryRepository()
	idGen := &mockIDGenerator{}
	config := DefaultTrainingSetBuilderConfig()

	svc := NewTrainingSetBuilderService(voteRepo, trainingRepo, toolRepo, memoryRepo, idGen, config)

	// Create upvote context
	voteWithContext := &ports.VoteWithToolContext{
		Vote: &models.Vote{
			ID:         "vote_1",
			TargetType: models.VoteTargetToolUse,
			TargetID:   "tu_1",
			Value:      models.VoteValueUp,
		},
		ToolUse: &models.ToolUse{
			ID:       "tu_1",
			ToolName: "search",
			Arguments: map[string]any{
				"query": "test query",
			},
		},
		UserMessage: "Please search for test query",
	}

	tools := []*models.Tool{
		{Name: "search", Description: "Search tool"},
	}

	example := svc.toolUseVoteToExample(voteWithContext, tools)

	// Verify inputs
	if userMsg, ok := example.Inputs["user_message"].(string); !ok || userMsg != "Please search for test query" {
		t.Errorf("expected user_message 'Please search for test query', got %v", example.Inputs["user_message"])
	}

	// Verify outputs
	if tool, ok := example.Outputs["selected_tool"].(string); !ok || tool != "search" {
		t.Errorf("expected selected_tool 'search', got %v", example.Outputs["selected_tool"])
	}

	// Upvotes should NOT have vote feedback in outputs
	if _, ok := example.Outputs["_vote_feedback"]; ok {
		t.Error("upvote should not have _vote_feedback in outputs")
	}
}

func TestToolUseVoteToExample_Downvote(t *testing.T) {
	voteRepo := newMockVoteRepository()
	trainingRepo := newMockTrainingExampleRepository()
	toolRepo := newMockToolRepository()
	memoryRepo := newMockMemoryRepository()
	idGen := &mockIDGenerator{}
	config := DefaultTrainingSetBuilderConfig()

	svc := NewTrainingSetBuilderService(voteRepo, trainingRepo, toolRepo, memoryRepo, idGen, config)

	// Create downvote context
	voteWithContext := &ports.VoteWithToolContext{
		Vote: &models.Vote{
			ID:            "vote_2",
			TargetType:    models.VoteTargetToolUse,
			TargetID:      "tu_2",
			Value:         models.VoteValueDown,
			QuickFeedback: "wrong_tool",
		},
		ToolUse: &models.ToolUse{
			ID:       "tu_2",
			ToolName: "calculator",
			Arguments: map[string]any{
				"expression": "2+2",
			},
		},
		UserMessage: "What is the weather?",
	}

	tools := []*models.Tool{
		{Name: "calculator", Description: "Math tool"},
		{Name: "weather", Description: "Weather tool"},
	}

	example := svc.toolUseVoteToExample(voteWithContext, tools)

	// Verify outputs contain vote metadata
	if _, ok := example.Outputs["_vote_feedback"]; !ok {
		t.Error("downvote should have _vote_feedback in outputs")
	}

	if voteValue, ok := example.Outputs["_vote_value"].(int); !ok || voteValue != models.VoteValueDown {
		t.Errorf("expected _vote_value %d, got %v", models.VoteValueDown, example.Outputs["_vote_value"])
	}

	if quickFeedback, ok := example.Outputs["_quick_feedback"].(string); !ok || quickFeedback != "wrong_tool" {
		t.Errorf("expected _quick_feedback 'wrong_tool', got %v", example.Outputs["_quick_feedback"])
	}
}

func TestBuildDiagnosticFeedback(t *testing.T) {
	tests := []struct {
		name          string
		quickFeedback string
		toolName      string
		args          map[string]any
		wantContains  string
	}{
		{
			name:          "wrong_tool feedback",
			quickFeedback: "wrong_tool",
			toolName:      "calculator",
			args:          nil,
			wantContains:  "calculator",
		},
		{
			name:          "unnecessary feedback",
			quickFeedback: "unnecessary",
			toolName:      "",
			args:          nil,
			wantContains:  "tool was used when none was needed",
		},
		{
			name:          "wrong_params feedback",
			quickFeedback: "wrong_params",
			toolName:      "search",
			args:          map[string]any{"query": "test"},
			wantContains:  "arguments",
		},
		{
			name:          "missing_context feedback",
			quickFeedback: "missing_context",
			toolName:      "",
			args:          nil,
			wantContains:  "conversation history",
		},
		{
			name:          "wrong_context memory feedback",
			quickFeedback: "wrong_context",
			toolName:      "",
			args:          nil,
			wantContains:  "wasn't relevant",
		},
		{
			name:          "too_generic memory feedback",
			quickFeedback: "too_generic",
			toolName:      "",
			args:          nil,
			wantContains:  "too generic",
		},
		{
			name:          "outdated memory feedback",
			quickFeedback: "outdated",
			toolName:      "",
			args:          nil,
			wantContains:  "outdated",
		},
		{
			name:          "incorrect memory feedback",
			quickFeedback: "incorrect",
			toolName:      "",
			args:          nil,
			wantContains:  "incorrect information",
		},
		{
			name:          "default feedback",
			quickFeedback: "unknown",
			toolName:      "",
			args:          nil,
			wantContains:  "marked as incorrect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedback := BuildDiagnosticFeedback(tt.quickFeedback, tt.toolName, tt.args)
			if len(feedback) == 0 {
				t.Error("expected non-empty feedback")
			}
			// Simple substring check
			if tt.wantContains != "" {
				found := false
				for i := 0; i <= len(feedback)-len(tt.wantContains); i++ {
					if feedback[i:i+len(tt.wantContains)] == tt.wantContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected feedback to contain '%s', got '%s'", tt.wantContains, feedback)
				}
			}
		})
	}
}

func TestGetOrBuildToolSelectionDataset_FallbackToSynthetic(t *testing.T) {
	voteRepo := newMockVoteRepository()
	trainingRepo := newMockTrainingExampleRepository()
	toolRepo := newMockToolRepository()
	memoryRepo := newMockMemoryRepository()
	idGen := &mockIDGenerator{}
	config := DefaultTrainingSetBuilderConfig()

	// Set vote count below threshold (MinVotesForReal = 15)
	voteRepo.targetTypeCounts[models.VoteTargetToolUse] = 5

	svc := NewTrainingSetBuilderService(voteRepo, trainingRepo, toolRepo, memoryRepo, idGen, config)

	train, val, err := svc.GetOrBuildToolSelectionDataset(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return synthetic dataset when below threshold
	if len(train) == 0 && len(val) == 0 {
		t.Error("expected non-empty synthetic dataset")
	}
}

func TestGetOrBuildToolSelectionDataset_UsesVotes(t *testing.T) {
	voteRepo := newMockVoteRepository()
	trainingRepo := newMockTrainingExampleRepository()
	toolRepo := newMockToolRepository()
	memoryRepo := newMockMemoryRepository()
	idGen := &mockIDGenerator{}
	config := DefaultTrainingSetBuilderConfig()

	// Set vote count above threshold (MinVotesForReal = 15)
	voteRepo.targetTypeCounts[models.VoteTargetToolUse] = 20

	// Add mock vote data
	voteRepo.toolUseVotes = []*ports.VoteWithToolContext{
		{
			Vote: &models.Vote{
				ID:         "vote_1",
				TargetType: models.VoteTargetToolUse,
				TargetID:   "tu_1",
				Value:      models.VoteValueUp,
			},
			ToolUse: &models.ToolUse{
				ID:       "tu_1",
				ToolName: "search",
				Arguments: map[string]any{
					"query": "test",
				},
			},
			UserMessage: "Search for test",
		},
		{
			Vote: &models.Vote{
				ID:         "vote_2",
				TargetType: models.VoteTargetToolUse,
				TargetID:   "tu_2",
				Value:      models.VoteValueDown,
			},
			ToolUse: &models.ToolUse{
				ID:       "tu_2",
				ToolName: "calculator",
				Arguments: map[string]any{
					"expression": "2+2",
				},
			},
			UserMessage: "Calculate 2+2",
		},
	}

	svc := NewTrainingSetBuilderService(voteRepo, trainingRepo, toolRepo, memoryRepo, idGen, config)

	train, val, err := svc.GetOrBuildToolSelectionDataset(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return vote-based dataset when above threshold
	totalExamples := len(train) + len(val)
	if totalExamples != 2 {
		t.Errorf("expected 2 total examples from votes, got %d", totalExamples)
	}

	// With TrainValSplit of 0.8 and 2 examples, expect 1 train, 1 val
	if len(train) < 1 {
		t.Error("expected at least 1 training example")
	}
}

func TestGetTrainingStats(t *testing.T) {
	voteRepo := newMockVoteRepository()
	trainingRepo := newMockTrainingExampleRepository()
	toolRepo := newMockToolRepository()
	memoryRepo := newMockMemoryRepository()
	idGen := &mockIDGenerator{}
	config := DefaultTrainingSetBuilderConfig()

	// Set up mock data
	voteRepo.targetTypeCounts[models.VoteTargetToolUse] = 10
	voteRepo.targetTypeCounts[models.VoteTargetMemoryUsage] = 5
	voteRepo.targetTypeCounts[models.VoteTargetMemoryExtraction] = 3
	trainingRepo.taskCounts[models.TaskTypeToolSelection] = 20
	trainingRepo.taskCounts[models.TaskTypeMemorySelection] = 15
	trainingRepo.taskCounts[models.TaskTypeMemoryExtraction] = 8

	svc := NewTrainingSetBuilderService(voteRepo, trainingRepo, toolRepo, memoryRepo, idGen, config)

	stats, err := svc.GetTrainingStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.ToolSelectionVotes != 10 {
		t.Errorf("expected 10 tool selection votes, got %d", stats.ToolSelectionVotes)
	}

	if stats.MemoryUsageVotes != 5 {
		t.Errorf("expected 5 memory usage votes, got %d", stats.MemoryUsageVotes)
	}

	if stats.MemoryExtractionVotes != 3 {
		t.Errorf("expected 3 memory extraction votes, got %d", stats.MemoryExtractionVotes)
	}

	if stats.ToolSelectionExamples != 20 {
		t.Errorf("expected 20 tool selection examples, got %d", stats.ToolSelectionExamples)
	}

	if stats.MemorySelectionExamples != 15 {
		t.Errorf("expected 15 memory selection examples, got %d", stats.MemorySelectionExamples)
	}

	if stats.MemoryExtractionExamples != 8 {
		t.Errorf("expected 8 memory extraction examples, got %d", stats.MemoryExtractionExamples)
	}

	if stats.MinVotesRequired != config.MinVotesForReal {
		t.Errorf("expected min votes required %d, got %d", config.MinVotesForReal, stats.MinVotesRequired)
	}
}
