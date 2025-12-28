package services

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
)

// Mock optimization repository
type mockOptimizationRepo struct {
	runs       map[string]*models.OptimizationRun
	candidates map[string][]*models.PromptCandidate
	evals      map[string][]*models.PromptEvaluation
}

func newMockOptimizationRepo() *mockOptimizationRepo {
	return &mockOptimizationRepo{
		runs:       make(map[string]*models.OptimizationRun),
		candidates: make(map[string][]*models.PromptCandidate),
		evals:      make(map[string][]*models.PromptEvaluation),
	}
}

func (m *mockOptimizationRepo) CreateRun(ctx context.Context, run *models.OptimizationRun) error {
	m.runs[run.ID] = run
	return nil
}

func (m *mockOptimizationRepo) GetRun(ctx context.Context, id string) (*models.OptimizationRun, error) {
	if run, ok := m.runs[id]; ok {
		return run, nil
	}
	return nil, errNotFound
}

func (m *mockOptimizationRepo) UpdateRun(ctx context.Context, run *models.OptimizationRun) error {
	if _, ok := m.runs[run.ID]; !ok {
		return errNotFound
	}
	m.runs[run.ID] = run
	return nil
}

func (m *mockOptimizationRepo) ListRuns(ctx context.Context, opts ports.ListOptimizationRunsOptions) ([]*models.OptimizationRun, error) {
	runs := make([]*models.OptimizationRun, 0)
	for _, run := range m.runs {
		runs = append(runs, run)
	}
	return runs, nil
}

func (m *mockOptimizationRepo) SaveCandidate(ctx context.Context, runID string, candidate *models.PromptCandidate) error {
	m.candidates[runID] = append(m.candidates[runID], candidate)
	return nil
}

func (m *mockOptimizationRepo) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	if candidates, ok := m.candidates[runID]; ok {
		return candidates, nil
	}
	return []*models.PromptCandidate{}, nil
}

func (m *mockOptimizationRepo) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	candidates, ok := m.candidates[runID]
	if !ok || len(candidates) == 0 {
		return nil, errNotFound
	}
	return candidates[0], nil
}

func (m *mockOptimizationRepo) SaveEvaluation(ctx context.Context, eval *models.PromptEvaluation) error {
	m.evals[eval.CandidateID] = append(m.evals[eval.CandidateID], eval)
	return nil
}

func (m *mockOptimizationRepo) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	if evals, ok := m.evals[candidateID]; ok {
		return evals, nil
	}
	return []*models.PromptEvaluation{}, nil
}

// Mock LLM service for testing
type mockLLMService struct{}

func (m *mockLLMService) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	return &ports.LLMResponse{
		Content: "mock response",
	}, nil
}

func (m *mockLLMService) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	return &ports.LLMResponse{
		Content: "mock response",
	}, nil
}

func (m *mockLLMService) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	ch := make(chan ports.LLMStreamChunk, 1)
	ch <- ports.LLMStreamChunk{Content: "mock", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockLLMService) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	ch := make(chan ports.LLMStreamChunk, 1)
	ch <- ports.LLMStreamChunk{Content: "mock", Done: true}
	close(ch)
	return ch, nil
}

// Tests

func TestOptimizationService_StartOptimizationRun(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, err := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if run.ID != "aor_test" {
		t.Errorf("expected ID aor_test, got %s", run.ID)
	}

	if run.Name != "test-run" {
		t.Errorf("expected name test-run, got %s", run.Name)
	}

	if run.PromptType != "signature" {
		t.Errorf("expected prompt type signature, got %s", run.PromptType)
	}

	if run.Status != models.OptimizationStatusRunning {
		t.Errorf("expected status running, got %s", run.Status)
	}
}

func TestOptimizationService_StartOptimizationRun_EmptyName(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	_, err := svc.StartOptimizationRun(context.Background(), "", "signature", "baseline")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestOptimizationService_StartOptimizationRun_EmptyPromptType(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	_, err := svc.StartOptimizationRun(context.Background(), "test-run", "", "baseline")
	if err == nil {
		t.Fatal("expected error for empty prompt type, got nil")
	}
}

func TestOptimizationService_GetOptimizationRun(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	created, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	retrieved, err := svc.GetOptimizationRun(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestOptimizationService_GetOptimizationRun_NotFound(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	_, err := svc.GetOptimizationRun(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent run, got nil")
	}
}

func TestOptimizationService_AddCandidate(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	candidate, err := svc.AddCandidate(context.Background(), run.ID, "test prompt text", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if candidate.ID != "apc_test" {
		t.Errorf("expected ID apc_test, got %s", candidate.ID)
	}

	if candidate.RunID != run.ID {
		t.Errorf("expected run ID %s, got %s", run.ID, candidate.RunID)
	}

	if candidate.PromptText != "test prompt text" {
		t.Errorf("expected prompt text 'test prompt text', got %s", candidate.PromptText)
	}

	if candidate.Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", candidate.Iteration)
	}
}

func TestOptimizationService_AddCandidate_EmptyRunID(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	_, err := svc.AddCandidate(context.Background(), "", "prompt text", 1)
	if err == nil {
		t.Fatal("expected error for empty run ID, got nil")
	}
}

func TestOptimizationService_AddCandidate_EmptyPromptText(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	_, err := svc.AddCandidate(context.Background(), run.ID, "", 1)
	if err == nil {
		t.Fatal("expected error for empty prompt text, got nil")
	}
}

func TestOptimizationService_RecordEvaluation(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")
	candidate, _ := svc.AddCandidate(context.Background(), run.ID, "prompt", 1)

	eval, err := svc.RecordEvaluation(context.Background(), candidate.ID, run.ID, "input", "output", 0.85, true, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eval.CandidateID != candidate.ID {
		t.Errorf("expected candidate ID %s, got %s", candidate.ID, eval.CandidateID)
	}

	if eval.Score != 0.85 {
		t.Errorf("expected score 0.85, got %f", eval.Score)
	}

	if !eval.Success {
		t.Error("expected success to be true")
	}

	if eval.LatencyMs != 100 {
		t.Errorf("expected latency 100, got %d", eval.LatencyMs)
	}
}

func TestOptimizationService_RecordEvaluationWithDimensions(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")
	candidate, _ := svc.AddCandidate(context.Background(), run.ID, "prompt", 1)

	dimScores := prompt.DimensionScores{
		SuccessRate:    0.9,
		Quality:        0.85,
		Efficiency:     0.8,
		Robustness:     0.75,
		Generalization: 0.7,
		Diversity:      0.6,
		Innovation:     0.5,
	}

	eval, err := svc.RecordEvaluationWithDimensions(context.Background(), candidate.ID, run.ID, "input", "output", dimScores, true, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eval.DimensionScores == nil {
		t.Fatal("expected dimension scores to be set")
	}

	if eval.DimensionScores["successRate"] != 0.9 {
		t.Errorf("expected success rate 0.9, got %f", eval.DimensionScores["successRate"])
	}

	if eval.DimensionScores["quality"] != 0.85 {
		t.Errorf("expected quality 0.85, got %f", eval.DimensionScores["quality"])
	}
}

func TestOptimizationService_GetCandidates(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")
	svc.AddCandidate(context.Background(), run.ID, "prompt 1", 1)
	svc.AddCandidate(context.Background(), run.ID, "prompt 2", 2)

	candidates, err := svc.GetCandidates(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(candidates))
	}
}

func TestOptimizationService_CompleteRun(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	err := svc.CompleteRun(context.Background(), run.ID, 0.95)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := svc.GetOptimizationRun(context.Background(), run.ID)
	if updated.Status != models.OptimizationStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}

	if updated.BestScore != 0.95 {
		t.Errorf("expected best score 0.95, got %f", updated.BestScore)
	}
}

func TestOptimizationService_FailRun(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	err := svc.FailRun(context.Background(), run.ID, "test failure reason")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := svc.GetOptimizationRun(context.Background(), run.ID)
	if updated.Status != models.OptimizationStatusFailed {
		t.Errorf("expected status failed, got %s", updated.Status)
	}

	if updated.Config["failure_reason"] != "test failure reason" {
		t.Errorf("expected failure reason in config, got %v", updated.Config["failure_reason"])
	}
}

func TestOptimizationService_UpdateProgress(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	err := svc.UpdateProgress(context.Background(), run.ID, 5, 0.75)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := svc.GetOptimizationRun(context.Background(), run.ID)
	if updated.Iterations != 5 {
		t.Errorf("expected iterations 5, got %d", updated.Iterations)
	}

	if updated.BestScore != 0.75 {
		t.Errorf("expected best score 0.75, got %f", updated.BestScore)
	}
}

func TestOptimizationService_UpdateProgressWithDimensions(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	dimScores := prompt.DimensionScores{
		SuccessRate:    0.9,
		Quality:        0.85,
		Efficiency:     0.8,
		Robustness:     0.75,
		Generalization: 0.7,
		Diversity:      0.6,
		Innovation:     0.5,
	}

	err := svc.UpdateProgressWithDimensions(context.Background(), run.ID, 5, dimScores)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := svc.GetOptimizationRun(context.Background(), run.ID)
	if updated.Iterations != 5 {
		t.Errorf("expected iterations 5, got %d", updated.Iterations)
	}

	if updated.BestDimScores == nil {
		t.Fatal("expected best dimension scores to be set")
	}
}

func TestOptimizationService_GetDimensionWeights(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	weights := svc.GetDimensionWeights()
	if weights == nil {
		t.Fatal("expected weights to be returned")
	}

	if _, ok := weights["successRate"]; !ok {
		t.Error("expected successRate in weights")
	}
}

func TestOptimizationService_SetDimensionWeights(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	newWeights := map[string]float64{
		"successRate":    0.5,
		"quality":        0.3,
		"efficiency":     0.1,
		"robustness":     0.05,
		"generalization": 0.03,
		"diversity":      0.01,
		"innovation":     0.01,
	}

	svc.SetDimensionWeights(newWeights)

	retrievedWeights := svc.GetDimensionWeights()
	if retrievedWeights["successRate"] < 0.4 || retrievedWeights["successRate"] > 0.6 {
		t.Errorf("expected successRate around 0.5, got %f", retrievedWeights["successRate"])
	}
}

func TestOptimizationService_GetOptimizedProgram(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")
	svc.AddCandidate(context.Background(), run.ID, "optimized prompt", 1)
	svc.CompleteRun(context.Background(), run.ID, 0.95)

	program, err := svc.GetOptimizedProgram(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if program.RunID != run.ID {
		t.Errorf("expected run ID %s, got %s", run.ID, program.RunID)
	}

	if program.BestScore != 0.95 {
		t.Errorf("expected best score 0.95, got %f", program.BestScore)
	}
}

func TestOptimizationService_GetOptimizedProgram_NotCompleted(t *testing.T) {
	repo := newMockOptimizationRepo()
	llm := &mockLLMService{}
	idGen := &mockIDGenerator{}
	config := DefaultOptimizationConfig()

	svc := NewOptimizationService(repo, llm, idGen, config)

	run, _ := svc.StartOptimizationRun(context.Background(), "test-run", "signature", "baseline")

	_, err := svc.GetOptimizedProgram(context.Background(), run.ID)
	if err == nil {
		t.Fatal("expected error for non-completed run, got nil")
	}
}
