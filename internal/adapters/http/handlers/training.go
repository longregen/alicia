package handlers

import (
	"net/http"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/application/services"
)

// TrainingHandler handles training and prompt version API endpoints
type TrainingHandler struct {
	trainingBuilder *services.TrainingSetBuilderService
	optimization    *services.OptimizationService
	promptVersion   *services.PromptVersionService
}

// NewTrainingHandler creates a new training handler
func NewTrainingHandler(
	trainingBuilder *services.TrainingSetBuilderService,
	optimization *services.OptimizationService,
	promptVersion *services.PromptVersionService,
) *TrainingHandler {
	return &TrainingHandler{
		trainingBuilder: trainingBuilder,
		optimization:    optimization,
		promptVersion:   promptVersion,
	}
}

// OptimizeFromVotesRequest represents a request to run optimization from votes
type OptimizeFromVotesRequest struct {
	TaskType string `json:"task_type"` // "tool_selection", "memory_selection", "memory_extraction"
}

// GetTrainingStats handles GET /api/v1/training/stats
func (h *TrainingHandler) GetTrainingStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	stats, err := h.trainingBuilder.GetTrainingStats(r.Context())
	if err != nil {
		respondError(w, "service_error", "Failed to get training stats", http.StatusInternalServerError)
		return
	}

	respondJSON(w, stats, http.StatusOK)
}

// RunOptimization handles POST /api/v1/training/optimize
func (h *TrainingHandler) RunOptimization(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[OptimizeFromVotesRequest](r, w)
	if !ok {
		return
	}

	// Validate task type
	validTypes := map[string]bool{
		"tool_selection":    true,
		"memory_selection":  true,
		"memory_extraction": true,
	}
	if !validTypes[req.TaskType] {
		respondError(w, "validation_error", "Invalid task type", http.StatusBadRequest)
		return
	}

	run, err := h.optimization.OptimizeFromVotes(r.Context(), req.TaskType, h.trainingBuilder)
	if err != nil {
		respondError(w, "service_error", "Failed to run optimization", http.StatusInternalServerError)
		return
	}

	respondJSON(w, run, http.StatusOK)
}

// ListPromptVersions handles GET /api/v1/prompts/versions?type=main
func (h *TrainingHandler) ListPromptVersions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	promptType := r.URL.Query().Get("type")
	if promptType == "" {
		promptType = "main"
	}

	versions, err := h.promptVersion.ListVersions(r.Context(), promptType, 20)
	if err != nil {
		respondError(w, "service_error", "Failed to list prompt versions", http.StatusInternalServerError)
		return
	}

	respondJSON(w, versions, http.StatusOK)
}

// ActivatePromptVersion handles POST /api/v1/prompts/versions/{id}/activate
func (h *TrainingHandler) ActivatePromptVersion(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	versionID, ok := validateURLParam(r, w, "id", "Prompt version ID")
	if !ok {
		return
	}

	if err := h.promptVersion.ActivateVersion(r.Context(), versionID); err != nil {
		respondError(w, "service_error", "Failed to activate prompt version", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
