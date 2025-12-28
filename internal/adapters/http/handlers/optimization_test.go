package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock OptimizationService
type mockOptimizationService struct {
	startErr         error
	getErr           error
	listErr          error
	getCandidatesErr error
	getBestErr       error
	getProgramErr    error
	getEvalsErr      error

	run        *models.OptimizationRun
	runs       []*models.OptimizationRun
	candidates []*models.PromptCandidate
	candidate  *models.PromptCandidate
	program    *ports.OptimizedProgram
	evals      []*models.PromptEvaluation
}

func (m *mockOptimizationService) StartOptimizationRun(ctx context.Context, name, promptType, baselinePrompt string) (*models.OptimizationRun, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	return m.run, nil
}

func (m *mockOptimizationService) GetOptimizationRun(ctx context.Context, runID string) (*models.OptimizationRun, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.run, nil
}

func (m *mockOptimizationService) ListOptimizationRuns(ctx context.Context, opts ports.ListOptimizationRunsOptions) ([]*models.OptimizationRun, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.runs, nil
}

func (m *mockOptimizationService) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	if m.getCandidatesErr != nil {
		return nil, m.getCandidatesErr
	}
	return m.candidates, nil
}

func (m *mockOptimizationService) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	if m.getBestErr != nil {
		return nil, m.getBestErr
	}
	return m.candidate, nil
}

func (m *mockOptimizationService) GetOptimizedProgram(ctx context.Context, runID string) (*ports.OptimizedProgram, error) {
	if m.getProgramErr != nil {
		return nil, m.getProgramErr
	}
	return m.program, nil
}

func (m *mockOptimizationService) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	if m.getEvalsErr != nil {
		return nil, m.getEvalsErr
	}
	return m.evals, nil
}

func (m *mockOptimizationService) AddCandidate(ctx context.Context, runID, promptText string, iteration int) (*models.PromptCandidate, error) {
	return m.candidate, nil
}

func (m *mockOptimizationService) SetDimensionWeights(weights map[string]float64) {
}

func (m *mockOptimizationService) GetDimensionWeights() map[string]float64 {
	return map[string]float64{}
}

func (m *mockOptimizationService) CompleteRun(ctx context.Context, runID string, bestScore float64) error {
	return nil
}

func (m *mockOptimizationService) FailRun(ctx context.Context, runID string, reason string) error {
	return nil
}

func (m *mockOptimizationService) UpdateProgress(ctx context.Context, runID string, iteration int, currentScore float64) error {
	return nil
}

func (m *mockOptimizationService) RecordEvaluation(ctx context.Context, candidateID, runID, input, output string, score float64, success bool, latencyMs int64) (*models.PromptEvaluation, error) {
	return &models.PromptEvaluation{}, nil
}

// Tests for OptimizationHandler.CreateOptimization

func TestOptimizationHandler_CreateOptimization_Success(t *testing.T) {
	now := time.Now()
	run := &models.OptimizationRun{
		ID:            "aor_test123",
		Name:          "Test Optimization",
		PromptType:    "conversation",
		Status:        models.OptimizationStatusRunning,
		BestScore:     0.0,
		Iterations:    0,
		MaxIterations: 10,
		Config:        map[string]any{"key": "value"},
		CreatedAt:     now,
	}

	mockService := &mockOptimizationService{run: run}
	handler := NewOptimizationHandler(mockService)

	body := `{"name": "Test Optimization", "prompt_type": "conversation", "baseline_prompt": "Test prompt"}`
	req := httptest.NewRequest("POST", "/api/v1/optimizations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateOptimization(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var response OptimizationRunResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != "aor_test123" {
		t.Errorf("expected id 'aor_test123', got %v", response.ID)
	}

	if response.Name != "Test Optimization" {
		t.Errorf("expected name 'Test Optimization', got %v", response.Name)
	}
}

func TestOptimizationHandler_CreateOptimization_MissingName(t *testing.T) {
	mockService := &mockOptimizationService{}
	handler := NewOptimizationHandler(mockService)

	body := `{"prompt_type": "conversation"}`
	req := httptest.NewRequest("POST", "/api/v1/optimizations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateOptimization(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestOptimizationHandler_CreateOptimization_InvalidPromptType(t *testing.T) {
	mockService := &mockOptimizationService{}
	handler := NewOptimizationHandler(mockService)

	body := `{"name": "Test", "prompt_type": "invalid_type"}`
	req := httptest.NewRequest("POST", "/api/v1/optimizations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateOptimization(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestOptimizationHandler_CreateOptimization_ServiceError(t *testing.T) {
	mockService := &mockOptimizationService{startErr: errors.New("service error")}
	handler := NewOptimizationHandler(mockService)

	body := `{"name": "Test", "prompt_type": "conversation"}`
	req := httptest.NewRequest("POST", "/api/v1/optimizations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.CreateOptimization(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// Tests for OptimizationHandler.GetOptimization

func TestOptimizationHandler_GetOptimization_Success(t *testing.T) {
	now := time.Now()
	run := &models.OptimizationRun{
		ID:            "aor_test123",
		Name:          "Test Optimization",
		PromptType:    "conversation",
		Status:        models.OptimizationStatusCompleted,
		BestScore:     0.95,
		Iterations:    5,
		MaxIterations: 10,
		CreatedAt:     now,
		CompletedAt:   &now,
	}

	mockService := &mockOptimizationService{run: run}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/aor_test123", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "aor_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetOptimization(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response OptimizationRunResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != "aor_test123" {
		t.Errorf("expected id 'aor_test123', got %v", response.ID)
	}

	if response.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestOptimizationHandler_GetOptimization_NotFound(t *testing.T) {
	mockService := &mockOptimizationService{getErr: errors.New("not found")}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/nonexistent", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetOptimization(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// Tests for OptimizationHandler.ListOptimizations

func TestOptimizationHandler_ListOptimizations_Success(t *testing.T) {
	now := time.Now()
	runs := []*models.OptimizationRun{
		{
			ID:         "aor_1",
			Name:       "Run 1",
			PromptType: "conversation",
			Status:     models.OptimizationStatusRunning,
			CreatedAt:  now,
		},
		{
			ID:         "aor_2",
			Name:       "Run 2",
			PromptType: "tool_selection",
			Status:     models.OptimizationStatusCompleted,
			CreatedAt:  now,
		},
	}

	mockService := &mockOptimizationService{runs: runs}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.ListOptimizations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response []OptimizationRunResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 runs, got %d", len(response))
	}
}

func TestOptimizationHandler_ListOptimizations_WithFilters(t *testing.T) {
	mockService := &mockOptimizationService{runs: []*models.OptimizationRun{}}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations?status=completed&limit=10&offset=5", nil)
	req = addUserContext(req, "test-user")

	rr := httptest.NewRecorder()
	handler.ListOptimizations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// Tests for OptimizationHandler.GetCandidates

func TestOptimizationHandler_GetCandidates_Success(t *testing.T) {
	now := time.Now()
	candidates := []*models.PromptCandidate{
		{
			ID:         "apc_1",
			RunID:      "aor_test123",
			Iteration:  1,
			PromptText: "Test prompt 1",
			Score:      0.8,
			CreatedAt:  now,
		},
		{
			ID:         "apc_2",
			RunID:      "aor_test123",
			Iteration:  2,
			PromptText: "Test prompt 2",
			Score:      0.9,
			CreatedAt:  now,
		},
	}

	mockService := &mockOptimizationService{candidates: candidates}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/aor_test123/candidates", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "aor_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetCandidates(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response []CandidateResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(response))
	}
}

// Tests for OptimizationHandler.GetBestCandidate

func TestOptimizationHandler_GetBestCandidate_Success(t *testing.T) {
	now := time.Now()
	candidate := &models.PromptCandidate{
		ID:         "apc_best",
		RunID:      "aor_test123",
		Iteration:  3,
		PromptText: "Best prompt",
		Score:      0.95,
		CreatedAt:  now,
	}

	mockService := &mockOptimizationService{candidate: candidate}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/aor_test123/best", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "aor_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetBestCandidate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response CandidateResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != "apc_best" {
		t.Errorf("expected id 'apc_best', got %v", response.ID)
	}
}

// Tests for OptimizationHandler.GetOptimizedProgram

func TestOptimizationHandler_GetOptimizedProgram_Success(t *testing.T) {
	program := &ports.OptimizedProgram{
		RunID:       "aor_test123",
		BestPrompt:  "Optimized prompt",
		BestScore:   0.95,
		Iterations:  5,
		CompletedAt: time.Now().Format("2006-01-02T15:04:05Z"),
	}

	mockService := &mockOptimizationService{program: program}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/aor_test123/program", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "aor_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetOptimizedProgram(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["run_id"] != "aor_test123" {
		t.Errorf("expected run_id 'aor_test123', got %v", response["run_id"])
	}
}

// Tests for OptimizationHandler.GetEvaluations

func TestOptimizationHandler_GetEvaluations_Success(t *testing.T) {
	now := time.Now()
	evals := []*models.PromptEvaluation{
		{
			ID:          "ape_1",
			CandidateID: "apc_test123",
			Input:       "Test input 1",
			Output:      "Test output 1",
			Score:       0.85,
			Success:     true,
			LatencyMs:   100,
			CreatedAt:   now,
		},
		{
			ID:          "ape_2",
			CandidateID: "apc_test123",
			Input:       "Test input 2",
			Output:      "Test output 2",
			Score:       0.90,
			Success:     true,
			LatencyMs:   120,
			CreatedAt:   now,
		},
	}

	mockService := &mockOptimizationService{evals: evals}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/candidates/apc_test123/evaluations", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "apc_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetEvaluations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response []EvaluationResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 evaluations, got %d", len(response))
	}
}

func TestOptimizationHandler_GetEvaluations_ServiceError(t *testing.T) {
	mockService := &mockOptimizationService{getEvalsErr: errors.New("service error")}
	handler := NewOptimizationHandler(mockService)

	req := httptest.NewRequest("GET", "/api/v1/optimizations/candidates/apc_test123/evaluations", nil)
	req = addUserContext(req, "test-user")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "apc_test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetEvaluations(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
