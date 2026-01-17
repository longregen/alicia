package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

func TestDeploymentHandler_DeployPrompt(t *testing.T) {
	mockRepo := &MockPromptOptimizationRepository{
		runs: map[string]*models.OptimizationRun{
			"run_completed": {
				ID:         "run_completed",
				Name:       "Test Run",
				Status:     models.OptimizationStatusCompleted,
				PromptType: "conversation",
				BestScore:  0.85,
				BestDimScores: map[string]float64{
					"successRate": 0.90,
					"quality":     0.85,
				},
			},
			"run_running": {
				ID:         "run_running",
				Name:       "Running Run",
				Status:     models.OptimizationStatusRunning,
				PromptType: "conversation",
			},
		},
		candidates: map[string][]*models.PromptCandidate{
			"run_completed": {
				{
					ID:         "candidate_best",
					RunID:      "run_completed",
					PromptText: "You are an excellent assistant.",
					Score:      0.85,
				},
			},
		},
	}

	mockIDGen := &MockIDGenerator{}
	deploymentService := services.NewDeploymentService(mockRepo, mockIDGen)
	handler := NewDeploymentHandler(deploymentService)

	tests := []struct {
		name           string
		request        DeployPromptRequest
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "deploy completed run",
			request: DeployPromptRequest{
				RunID: "run_completed",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp DeploymentStatusResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				if resp.RunID != "run_completed" {
					t.Errorf("expected RunID to be run_completed, got %s", resp.RunID)
				}

				if resp.Score != 0.85 {
					t.Errorf("expected Score to be 0.85, got %f", resp.Score)
				}

				if !resp.IsActive {
					t.Error("expected deployment to be active")
				}
			},
		},
		{
			name: "deploy running run should fail",
			request: DeployPromptRequest{
				RunID: "run_running",
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
		{
			name:           "missing run_id",
			request:        DeployPromptRequest{},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = setTestUserID(req, "test-user")

			rec := httptest.NewRecorder()
			handler.DeployPrompt(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkResponse != nil && rec.Code == http.StatusCreated {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

func TestDeploymentHandler_GetActiveDeployment(t *testing.T) {
	mockRepo := &MockPromptOptimizationRepository{
		runs: map[string]*models.OptimizationRun{
			"run_deployed": {
				ID:         "run_deployed",
				Name:       "Deployed Run",
				Status:     models.OptimizationStatusCompleted,
				PromptType: "conversation",
				BestScore:  0.90,
				Meta: map[string]any{
					"deployed":            true,
					"deployed_by":         "user1",
					"active_candidate_id": "candidate_active",
				},
			},
		},
		candidates: map[string][]*models.PromptCandidate{
			"run_deployed": {
				{
					ID:         "candidate_active",
					RunID:      "run_deployed",
					PromptText: "You are the best assistant.",
					Score:      0.90,
				},
			},
		},
	}

	mockIDGen := &MockIDGenerator{}
	deploymentService := services.NewDeploymentService(mockRepo, mockIDGen)
	handler := NewDeploymentHandler(deploymentService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/conversation/active", nil)
	req = setTestUserID(req, "test-user")
	req = setURLParam(req, "prompt_type", "conversation")

	rec := httptest.NewRecorder()
	handler.GetActiveDeployment(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp DeploymentStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.RunID != "run_deployed" {
		t.Errorf("expected RunID to be run_deployed, got %s", resp.RunID)
	}

	if resp.Prompt != "You are the best assistant." {
		t.Errorf("expected specific prompt, got %s", resp.Prompt)
	}
}

// Mock implementations

type MockPromptOptimizationRepository struct {
	runs       map[string]*models.OptimizationRun
	candidates map[string][]*models.PromptCandidate
}

func (m *MockPromptOptimizationRepository) CreateRun(ctx context.Context, run *models.OptimizationRun) error {
	if m.runs == nil {
		m.runs = make(map[string]*models.OptimizationRun)
	}
	m.runs[run.ID] = run
	return nil
}

func (m *MockPromptOptimizationRepository) GetRun(ctx context.Context, id string) (*models.OptimizationRun, error) {
	if run, ok := m.runs[id]; ok {
		return run, nil
	}
	return nil, nil
}

func (m *MockPromptOptimizationRepository) UpdateRun(ctx context.Context, run *models.OptimizationRun) error {
	m.runs[run.ID] = run
	return nil
}

func (m *MockPromptOptimizationRepository) ListRuns(ctx context.Context, opts ports.ListOptimizationRunsOptions) ([]*models.OptimizationRun, error) {
	var runs []*models.OptimizationRun
	for _, run := range m.runs {
		if opts.Status == "" || run.Status == opts.Status {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func (m *MockPromptOptimizationRepository) SaveCandidate(ctx context.Context, runID string, candidate *models.PromptCandidate) error {
	if m.candidates == nil {
		m.candidates = make(map[string][]*models.PromptCandidate)
	}
	m.candidates[runID] = append(m.candidates[runID], candidate)
	return nil
}

func (m *MockPromptOptimizationRepository) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	return m.candidates[runID], nil
}

func (m *MockPromptOptimizationRepository) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	candidates := m.candidates[runID]
	if len(candidates) == 0 {
		return nil, nil
	}
	return candidates[0], nil
}

func (m *MockPromptOptimizationRepository) SaveEvaluation(ctx context.Context, eval *models.PromptEvaluation) error {
	return nil
}

func (m *MockPromptOptimizationRepository) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	return nil, nil
}

type MockOptimizationService struct {
	weights map[string]float64
}

func (m *MockOptimizationService) StartOptimizationRun(ctx context.Context, name, promptType, baselinePrompt string) (*models.OptimizationRun, error) {
	return nil, nil
}

func (m *MockOptimizationService) GetOptimizationRun(ctx context.Context, id string) (*models.OptimizationRun, error) {
	return nil, nil
}

func (m *MockOptimizationService) ListOptimizationRuns(ctx context.Context, status string, limit, offset int) ([]*models.OptimizationRun, error) {
	return nil, nil
}

func (m *MockOptimizationService) CompleteRun(ctx context.Context, runID string, bestScore float64) error {
	return nil
}

func (m *MockOptimizationService) FailRun(ctx context.Context, runID string, reason string) error {
	return nil
}

func (m *MockOptimizationService) UpdateProgress(ctx context.Context, runID string, iteration int, currentScore float64) error {
	return nil
}

func (m *MockOptimizationService) AddCandidate(ctx context.Context, runID, promptText string, iteration int) (*models.PromptCandidate, error) {
	return nil, nil
}

func (m *MockOptimizationService) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	return nil, nil
}

func (m *MockOptimizationService) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	return nil, nil
}

func (m *MockOptimizationService) RecordEvaluation(ctx context.Context, candidateID, runID, input, output string, score float64, success bool, latencyMs int64) (*models.PromptEvaluation, error) {
	return nil, nil
}

func (m *MockOptimizationService) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	return nil, nil
}

func (m *MockOptimizationService) GetOptimizedProgram(ctx context.Context, runID string) (*ports.OptimizedProgram, error) {
	return nil, nil
}

func (m *MockOptimizationService) GetDimensionWeights() map[string]float64 {
	if m.weights == nil {
		return map[string]float64{
			"successRate":    0.25,
			"quality":        0.20,
			"efficiency":     0.15,
			"robustness":     0.15,
			"generalization": 0.10,
			"diversity":      0.10,
			"innovation":     0.05,
		}
	}
	return m.weights
}

func (m *MockOptimizationService) SetDimensionWeights(weights map[string]float64) {
	m.weights = weights
}

func (m *MockOptimizationService) UpdateRunProgress(ctx context.Context, runID string, iteration int, bestScore float64, dimScores map[string]float64) error {
	return nil
}

func (m *MockOptimizationService) SaveCandidate(ctx context.Context, runID string, candidate *models.PromptCandidate) error {
	return nil
}

func (m *MockOptimizationService) SaveEvaluation(ctx context.Context, candidateID string, eval *models.PromptEvaluation) error {
	return nil
}
