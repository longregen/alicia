package handlers

import (
	"net/http"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
)

// FeedbackHandler handles feedback-related API endpoints
type FeedbackHandler struct {
	voteRepo            ports.VoteRepository
	optimizationService ports.OptimizationService
}

// NewFeedbackHandler creates a new feedback handler
func NewFeedbackHandler(
	voteRepo ports.VoteRepository,
	optimizationService ports.OptimizationService,
) *FeedbackHandler {
	return &FeedbackHandler{
		voteRepo:            voteRepo,
		optimizationService: optimizationService,
	}
}

// SubmitFeedbackRequest represents a request to submit feedback for optimization
type SubmitFeedbackRequest struct {
	TargetType    string `json:"target_type"`     // "message", "tool_use", "memory", "reasoning"
	TargetID      string `json:"target_id"`
	Vote          string `json:"vote"`            // "up", "down", "critical"
	QuickFeedback string `json:"quick_feedback"`  // Optional specific feedback
}

// DimensionWeightsResponse represents current dimension weights
type DimensionWeightsResponse struct {
	SuccessRate    float64 `json:"success_rate"`
	Quality        float64 `json:"quality"`
	Efficiency     float64 `json:"efficiency"`
	Robustness     float64 `json:"robustness"`
	Generalization float64 `json:"generalization"`
	Diversity      float64 `json:"diversity"`
	Innovation     float64 `json:"innovation"`
}

// SubmitFeedback handles POST /api/v1/feedback
// Converts frontend votes to dimension adjustments for GEPA
func (h *FeedbackHandler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[SubmitFeedbackRequest](r, w)
	if !ok {
		return
	}

	// Validate request
	if req.TargetType == "" || req.TargetID == "" || req.Vote == "" {
		respondError(w, "validation_error", "target_type, target_id, and vote are required", http.StatusBadRequest)
		return
	}

	// Convert vote to feedback type using the feedback mapping
	feedbackType := prompt.VoteToFeedback(req.Vote, req.QuickFeedback, req.TargetType)
	if feedbackType == "" {
		respondError(w, "validation_error", "Invalid vote or feedback combination", http.StatusBadRequest)
		return
	}

	// Apply feedback to dimension weights
	adjustment := prompt.MapFeedbackToDimensions(feedbackType)

	// Get current weights
	currentWeights := h.optimizationService.GetDimensionWeights()

	// Apply adjustment
	weights := prompt.DimensionWeightsFromMap(currentWeights)
	newWeights := prompt.ApplyAdjustment(weights, adjustment)

	// Update optimization service weights
	h.optimizationService.SetDimensionWeights(newWeights.ToMap())

	// Return updated weights and the adjustment that was applied
	response := map[string]any{
		"feedback_type": string(feedbackType),
		"adjustment": map[string]float64{
			"success_rate":    adjustment.SuccessRate,
			"quality":         adjustment.Quality,
			"efficiency":      adjustment.Efficiency,
			"robustness":      adjustment.Robustness,
			"generalization":  adjustment.Generalization,
			"diversity":       adjustment.Diversity,
			"innovation":      adjustment.Innovation,
		},
		"new_weights": newWeights.ToMap(),
	}

	respondJSON(w, response, http.StatusOK)
}

// GetDimensionWeights handles GET /api/v1/feedback/dimensions
// Returns current dimension weights for the optimization system
func (h *FeedbackHandler) GetDimensionWeights(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	weights := h.optimizationService.GetDimensionWeights()

	response := DimensionWeightsResponse{
		SuccessRate:    weights["successRate"],
		Quality:        weights["quality"],
		Efficiency:     weights["efficiency"],
		Robustness:     weights["robustness"],
		Generalization: weights["generalization"],
		Diversity:      weights["diversity"],
		Innovation:     weights["innovation"],
	}

	respondJSON(w, response, http.StatusOK)
}

// UpdateDimensionWeights handles PUT /api/v1/feedback/dimensions
// Allows manual adjustment of dimension weights
func (h *FeedbackHandler) UpdateDimensionWeights(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[DimensionWeightsResponse](r, w)
	if !ok {
		return
	}

	// Validate weights are in valid range
	weights := map[string]float64{
		"successRate":    req.SuccessRate,
		"quality":        req.Quality,
		"efficiency":     req.Efficiency,
		"robustness":     req.Robustness,
		"generalization": req.Generalization,
		"diversity":      req.Diversity,
		"innovation":     req.Innovation,
	}

	// Apply normalization through DimensionWeights
	dw := prompt.DimensionWeightsFromMap(weights)
	dw.Normalize()

	// Update optimization service
	h.optimizationService.SetDimensionWeights(dw.ToMap())

	response := DimensionWeightsResponse{
		SuccessRate:    dw.SuccessRate,
		Quality:        dw.Quality,
		Efficiency:     dw.Efficiency,
		Robustness:     dw.Robustness,
		Generalization: dw.Generalization,
		Diversity:      dw.Diversity,
		Innovation:     dw.Innovation,
	}

	respondJSON(w, response, http.StatusOK)
}
