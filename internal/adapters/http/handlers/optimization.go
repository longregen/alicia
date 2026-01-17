package handlers

import (
	"net/http"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/ports"
)

// OptimizationHandler handles prompt optimization API endpoints
type OptimizationHandler struct {
	optService             ports.OptimizationServiceFull
	runOptimizationUseCase ports.RunOptimizationUseCase
}

// NewOptimizationHandler creates a new optimization handler
func NewOptimizationHandler(optService ports.OptimizationServiceFull, runOptimizationUseCase ports.RunOptimizationUseCase) *OptimizationHandler {
	return &OptimizationHandler{
		optService:             optService,
		runOptimizationUseCase: runOptimizationUseCase,
	}
}

// CreateOptimizationRequest represents a request to start an optimization run
type CreateOptimizationRequest struct {
	Name           string `json:"name"`
	PromptType     string `json:"prompt_type"`     // "conversation", "tool_selection", "memory_extraction"
	BaselinePrompt string `json:"baseline_prompt"` // Initial prompt to optimize
}

// OptimizationRunResponse represents an optimization run in API responses
type OptimizationRunResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	PromptType    string         `json:"prompt_type"`
	Status        string         `json:"status"`
	BestScore     float64        `json:"best_score"`
	Iterations    int            `json:"iterations"`
	MaxIterations int            `json:"max_iterations"`
	Config        map[string]any `json:"config,omitempty"`
	CreatedAt     string         `json:"created_at"`
	CompletedAt   *string        `json:"completed_at,omitempty"`
}

// CandidateResponse represents a prompt candidate in API responses
type CandidateResponse struct {
	ID         string  `json:"id"`
	RunID      string  `json:"run_id"`
	Iteration  int     `json:"iteration"`
	PromptText string  `json:"prompt_text"`
	AvgScore   float64 `json:"avg_score"`
	CreatedAt  string  `json:"created_at"`
}

// EvaluationResponse represents an evaluation result in API responses
type EvaluationResponse struct {
	ID          string  `json:"id"`
	CandidateID string  `json:"candidate_id"`
	Input       string  `json:"input"`
	Output      string  `json:"output"`
	Score       float64 `json:"score"`
	Success     bool    `json:"success"`
	LatencyMs   int64   `json:"latency_ms"`
	CreatedAt   string  `json:"created_at"`
}

// CreateOptimization handles POST /api/v1/optimizations
func (h *OptimizationHandler) CreateOptimization(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[CreateOptimizationRequest](r, w)
	if !ok {
		return
	}

	if req.Name == "" {
		respondError(w, "validation_error", "Name is required", http.StatusBadRequest)
		return
	}

	if req.PromptType == "" {
		respondError(w, "validation_error", "Prompt type is required", http.StatusBadRequest)
		return
	}

	// Validate prompt type
	validTypes := map[string]bool{
		"conversation":       true,
		"tool_selection":     true,
		"memory_extraction":  true,
		"tool_description":   true,
		"tool_result_format": true,
	}
	if !validTypes[req.PromptType] {
		respondError(w, "validation_error", "Invalid prompt type", http.StatusBadRequest)
		return
	}

	// Use the RunOptimization usecase to start the optimization run
	// This creates the run AND starts the background optimization process
	output, err := h.runOptimizationUseCase.Execute(
		r.Context(),
		&ports.RunOptimizationInput{
			Name:           req.Name,
			PromptType:     req.PromptType,
			BaselinePrompt: req.BaselinePrompt,
		},
	)
	if err != nil {
		respondError(w, "service_error", "Failed to start optimization run", http.StatusInternalServerError)
		return
	}

	run := output.Run
	response := &OptimizationRunResponse{
		ID:            run.ID,
		Name:          run.Name,
		PromptType:    run.PromptType,
		Status:        string(run.Status),
		BestScore:     run.BestScore,
		Iterations:    run.Iterations,
		MaxIterations: run.MaxIterations,
		Config:        run.Config,
		CreatedAt:     run.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	respondJSON(w, response, http.StatusCreated)
}

// GetOptimization handles GET /api/v1/optimizations/{id}
func (h *OptimizationHandler) GetOptimization(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	runID, ok := validateURLParam(r, w, "id", "Optimization run ID")
	if !ok {
		return
	}

	run, err := h.optService.GetOptimizationRun(r.Context(), runID)
	if err != nil {
		respondError(w, "not_found", "Optimization run not found", http.StatusNotFound)
		return
	}

	response := &OptimizationRunResponse{
		ID:            run.ID,
		Name:          run.Name,
		PromptType:    run.PromptType,
		Status:        string(run.Status),
		BestScore:     run.BestScore,
		Iterations:    run.Iterations,
		MaxIterations: run.MaxIterations,
		Config:        run.Config,
		CreatedAt:     run.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if run.CompletedAt != nil {
		completedAt := run.CompletedAt.Format("2006-01-02T15:04:05Z")
		response.CompletedAt = &completedAt
	}

	respondJSON(w, response, http.StatusOK)
}

// ListOptimizations handles GET /api/v1/optimizations
func (h *OptimizationHandler) ListOptimizations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)

	runs, err := h.optService.ListOptimizationRuns(r.Context(), status, limit, offset)
	if err != nil {
		respondError(w, "service_error", "Failed to list optimization runs", http.StatusInternalServerError)
		return
	}

	responses := make([]OptimizationRunResponse, len(runs))
	for i, run := range runs {
		responses[i] = OptimizationRunResponse{
			ID:            run.ID,
			Name:          run.Name,
			PromptType:    run.PromptType,
			Status:        string(run.Status),
			BestScore:     run.BestScore,
			Iterations:    run.Iterations,
			MaxIterations: run.MaxIterations,
			Config:        run.Config,
			CreatedAt:     run.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if run.CompletedAt != nil {
			completedAt := run.CompletedAt.Format("2006-01-02T15:04:05Z")
			responses[i].CompletedAt = &completedAt
		}
	}

	respondJSON(w, responses, http.StatusOK)
}

// GetCandidates handles GET /api/v1/optimizations/{id}/candidates
func (h *OptimizationHandler) GetCandidates(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	runID, ok := validateURLParam(r, w, "id", "Optimization run ID")
	if !ok {
		return
	}

	candidates, err := h.optService.GetCandidates(r.Context(), runID)
	if err != nil {
		respondError(w, "service_error", "Failed to get candidates", http.StatusInternalServerError)
		return
	}

	responses := make([]CandidateResponse, len(candidates))
	for i, c := range candidates {
		responses[i] = CandidateResponse{
			ID:         c.ID,
			RunID:      c.RunID,
			Iteration:  c.Iteration,
			PromptText: c.PromptText,
			AvgScore:   c.Score,
			CreatedAt:  c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	respondJSON(w, responses, http.StatusOK)
}

// GetBestCandidate handles GET /api/v1/optimizations/{id}/best
func (h *OptimizationHandler) GetBestCandidate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	runID, ok := validateURLParam(r, w, "id", "Optimization run ID")
	if !ok {
		return
	}

	candidate, err := h.optService.GetBestCandidate(r.Context(), runID)
	if err != nil {
		respondError(w, "not_found", "Best candidate not found", http.StatusNotFound)
		return
	}

	response := &CandidateResponse{
		ID:         candidate.ID,
		RunID:      candidate.RunID,
		Iteration:  candidate.Iteration,
		PromptText: candidate.PromptText,
		AvgScore:   candidate.Score,
		CreatedAt:  candidate.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	respondJSON(w, response, http.StatusOK)
}

// GetOptimizedProgram handles GET /api/v1/optimizations/{id}/program
func (h *OptimizationHandler) GetOptimizedProgram(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	runID, ok := validateURLParam(r, w, "id", "Optimization run ID")
	if !ok {
		return
	}

	program, err := h.optService.GetOptimizedProgram(r.Context(), runID)
	if err != nil {
		respondError(w, "not_found", "Optimized program not found or run not completed", http.StatusNotFound)
		return
	}

	response := map[string]any{
		"run_id":       program.RunID,
		"best_prompt":  program.BestPrompt,
		"best_score":   program.BestScore,
		"iterations":   program.Iterations,
		"completed_at": program.CompletedAt,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetEvaluations handles GET /api/v1/optimizations/candidates/{id}/evaluations
func (h *OptimizationHandler) GetEvaluations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	candidateID, ok := validateURLParam(r, w, "id", "Candidate ID")
	if !ok {
		return
	}

	evals, err := h.optService.GetEvaluations(r.Context(), candidateID)
	if err != nil {
		respondError(w, "service_error", "Failed to get evaluations", http.StatusInternalServerError)
		return
	}

	responses := make([]EvaluationResponse, len(evals))
	for i, e := range evals {
		responses[i] = EvaluationResponse{
			ID:          e.ID,
			CandidateID: e.CandidateID,
			Input:       e.Input,
			Output:      e.Output,
			Score:       e.Score,
			Success:     e.Success,
			LatencyMs:   e.LatencyMs,
			CreatedAt:   e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	respondJSON(w, responses, http.StatusOK)
}
