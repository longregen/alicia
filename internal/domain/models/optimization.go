package models

import (
	"time"
)

// OptimizationRun represents a DSPy/GEPA optimization run
type OptimizationRun struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Description      string             `json:"description,omitempty"`
	Status           string             `json:"status"` // "running", "completed", "failed"
	PromptType       string             `json:"prompt_type"`
	BaselineScore    float64            `json:"baseline_score,omitempty"`
	BestScore        float64            `json:"best_score,omitempty"`
	Iterations       int                `json:"iterations"`
	MaxIterations    int                `json:"max_iterations"`
	DimensionWeights map[string]float64 `json:"dimension_weights,omitempty"` // GEPA 7-dimension weights
	BestDimScores    map[string]float64 `json:"best_dim_scores,omitempty"`   // Best dimension scores achieved
	Config           map[string]any     `json:"config,omitempty"`
	Meta             map[string]any     `json:"meta,omitempty"`
	StartedAt        time.Time          `json:"started_at"`
	CompletedAt      *time.Time         `json:"completed_at,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// OptimizationRun status values
const (
	OptimizationStatusRunning   = "running"
	OptimizationStatusCompleted = "completed"
	OptimizationStatusFailed    = "failed"
)

func NewOptimizationRun(id, name, promptType string, maxIterations int) *OptimizationRun {
	now := time.Now().UTC()
	return &OptimizationRun{
		ID:            id,
		Name:          name,
		Status:        OptimizationStatusRunning,
		PromptType:    promptType,
		MaxIterations: maxIterations,
		Iterations:    0,
		DimensionWeights: map[string]float64{
			"successRate":    0.25,
			"quality":        0.20,
			"efficiency":     0.15,
			"robustness":     0.15,
			"generalization": 0.10,
			"diversity":      0.10,
			"innovation":     0.05,
		},
		BestDimScores: make(map[string]float64),
		Config:        make(map[string]any),
		Meta:          make(map[string]any),
		StartedAt:     now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (r *OptimizationRun) MarkCompleted() {
	now := time.Now().UTC()
	r.Status = OptimizationStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now
}

func (r *OptimizationRun) MarkFailed() {
	now := time.Now().UTC()
	r.Status = OptimizationStatusFailed
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// PromptCandidate represents a candidate prompt variation being tested
type PromptCandidate struct {
	ID               string             `json:"id"`
	RunID            string             `json:"run_id"`
	Iteration        int                `json:"iteration"`
	PromptText       string             `json:"prompt_text"`
	PromptVariables  map[string]any     `json:"prompt_variables,omitempty"`
	Score            float64            `json:"score"`
	DimensionScores  map[string]float64 `json:"dimension_scores,omitempty"` // Per-dimension scores
	EvaluationCount  int                `json:"evaluation_count"`
	SuccessCount     int                `json:"success_count"`
	FailureCount     int                `json:"failure_count"`
	AverageLatencyMs float64            `json:"average_latency_ms,omitempty"`
	Meta             map[string]any     `json:"meta,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

func NewPromptCandidate(id, runID string, iteration int, promptText string) *PromptCandidate {
	now := time.Now().UTC()
	return &PromptCandidate{
		ID:              id,
		RunID:           runID,
		Iteration:       iteration,
		PromptText:      promptText,
		PromptVariables: make(map[string]any),
		DimensionScores: make(map[string]float64),
		Meta:            make(map[string]any),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (c *PromptCandidate) RecordEvaluation(score float64, success bool, latencyMs int64) {
	c.EvaluationCount++
	if success {
		c.SuccessCount++
	} else {
		c.FailureCount++
	}

	// Update average score (running average)
	c.Score = ((c.Score * float64(c.EvaluationCount-1)) + score) / float64(c.EvaluationCount)

	// Update average latency
	if latencyMs > 0 {
		c.AverageLatencyMs = ((c.AverageLatencyMs * float64(c.EvaluationCount-1)) + float64(latencyMs)) / float64(c.EvaluationCount)
	}

	c.UpdatedAt = time.Now().UTC()
}

// PromptEvaluation represents a single evaluation of a prompt candidate
type PromptEvaluation struct {
	ID              string             `json:"id"`
	CandidateID     string             `json:"candidate_id"`
	RunID           string             `json:"run_id"`
	Input           string             `json:"input"`
	Output          string             `json:"output"`
	Expected        string             `json:"expected,omitempty"`
	Score           float64            `json:"score"`
	DimensionScores map[string]float64 `json:"dimension_scores,omitempty"` // Per-dimension scores
	Success         bool               `json:"success"`
	LatencyMs       int64              `json:"latency_ms"`
	Metrics         map[string]any     `json:"metrics,omitempty"`
	Error           string             `json:"error,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

func NewPromptEvaluation(id, candidateID, runID, input, output string, score float64, success bool, latencyMs int64) *PromptEvaluation {
	return &PromptEvaluation{
		ID:              id,
		CandidateID:     candidateID,
		RunID:           runID,
		Input:           input,
		Output:          output,
		Score:           score,
		DimensionScores: make(map[string]float64),
		Success:         success,
		LatencyMs:       latencyMs,
		Metrics:         make(map[string]any),
		CreatedAt:       time.Now().UTC(),
	}
}

// UpdateDimensionScore updates a specific dimension score and recalculates the aggregate score
func (c *PromptCandidate) UpdateDimensionScore(dimension string, score float64) {
	if c.DimensionScores == nil {
		c.DimensionScores = make(map[string]float64)
	}
	c.DimensionScores[dimension] = score
	c.UpdatedAt = time.Now().UTC()
}

// SetDimensionScores sets all dimension scores at once
func (c *PromptCandidate) SetDimensionScores(scores map[string]float64) {
	c.DimensionScores = scores
	c.UpdatedAt = time.Now().UTC()
}

// GetWeightedScore calculates the weighted score based on provided dimension weights
func (c *PromptCandidate) GetWeightedScore(weights map[string]float64) float64 {
	if len(c.DimensionScores) == 0 || len(weights) == 0 {
		return c.Score
	}

	var score float64
	for dim, weight := range weights {
		if dimScore, ok := c.DimensionScores[dim]; ok {
			score += dimScore * weight
		}
	}
	return score
}
