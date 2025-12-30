package handlers

import (
	"net/http"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/application/services"
)

// DeploymentHandler handles prompt deployment API endpoints
type DeploymentHandler struct {
	deploymentService *services.DeploymentService
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(deploymentService *services.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{
		deploymentService: deploymentService,
	}
}

// DeployPromptRequest represents a request to deploy an optimized prompt
type DeployPromptRequest struct {
	RunID string `json:"run_id"`
}

// DeploymentStatusResponse represents a deployment status in API responses
type DeploymentStatusResponse struct {
	ID         string             `json:"id"`
	RunID      string             `json:"run_id"`
	PromptType string             `json:"prompt_type"`
	IsActive   bool               `json:"is_active"`
	Prompt     string             `json:"prompt"`
	Score      float64            `json:"score"`
	Dimensions map[string]float64 `json:"dimensions,omitempty"`
	DeployedAt string             `json:"deployed_at,omitempty"`
	DeployedBy string             `json:"deployed_by,omitempty"`
}

// DeployPrompt handles POST /api/v1/deployments
// Deploys an optimized prompt to production
func (h *DeploymentHandler) DeployPrompt(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[DeployPromptRequest](r, w)
	if !ok {
		return
	}

	if req.RunID == "" {
		respondError(w, "validation_error", "run_id is required", http.StatusBadRequest)
		return
	}

	deployment, err := h.deploymentService.DeployOptimizedPrompt(r.Context(), req.RunID, userID)
	if err != nil {
		respondError(w, "service_error", err.Error(), http.StatusInternalServerError)
		return
	}

	response := &DeploymentStatusResponse{
		ID:         deployment.ID,
		RunID:      deployment.RunID,
		PromptType: deployment.PromptType,
		IsActive:   deployment.IsActive,
		Prompt:     deployment.Prompt,
		Score:      deployment.Score,
		Dimensions: deployment.Dimensions,
		DeployedAt: deployment.DeployedAt,
		DeployedBy: deployment.DeployedBy,
	}

	respondJSON(w, response, http.StatusCreated)
}

// GetActiveDeployment handles GET /api/v1/deployments/{prompt_type}/active
// Returns the currently active deployment for a prompt type
func (h *DeploymentHandler) GetActiveDeployment(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	promptType, ok := validateURLParam(r, w, "prompt_type", "Prompt type")
	if !ok {
		return
	}

	deployment, err := h.deploymentService.GetActiveDeployment(r.Context(), promptType)
	if err != nil {
		respondError(w, "not_found", "No active deployment found", http.StatusNotFound)
		return
	}

	response := &DeploymentStatusResponse{
		ID:         deployment.ID,
		RunID:      deployment.RunID,
		PromptType: deployment.PromptType,
		IsActive:   deployment.IsActive,
		Prompt:     deployment.Prompt,
		Score:      deployment.Score,
		Dimensions: deployment.Dimensions,
		DeployedAt: deployment.DeployedAt,
		DeployedBy: deployment.DeployedBy,
	}

	respondJSON(w, response, http.StatusOK)
}

// RollbackDeployment handles DELETE /api/v1/deployments/{run_id}
// Deactivates a deployment (rollback)
func (h *DeploymentHandler) RollbackDeployment(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	runID, ok := validateURLParam(r, w, "run_id", "Run ID")
	if !ok {
		return
	}

	err := h.deploymentService.RollbackDeployment(r.Context(), runID, userID)
	if err != nil {
		respondError(w, "service_error", err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]string{
		"status":  "success",
		"message": "Deployment rolled back",
	}, http.StatusOK)
}

// ListDeploymentHistory handles GET /api/v1/deployments/{prompt_type}/history
// Returns deployment history for a prompt type
func (h *DeploymentHandler) ListDeploymentHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	promptType, ok := validateURLParam(r, w, "prompt_type", "Prompt type")
	if !ok {
		return
	}

	limit := parseIntQuery(r, "limit", 10)

	history, err := h.deploymentService.ListDeploymentHistory(r.Context(), promptType, limit)
	if err != nil {
		respondError(w, "service_error", "Failed to list deployment history", http.StatusInternalServerError)
		return
	}

	responses := make([]DeploymentStatusResponse, len(history))
	for i, deployment := range history {
		responses[i] = DeploymentStatusResponse{
			ID:         deployment.ID,
			RunID:      deployment.RunID,
			PromptType: deployment.PromptType,
			IsActive:   deployment.IsActive,
			Prompt:     deployment.Prompt,
			Score:      deployment.Score,
			Dimensions: deployment.Dimensions,
			DeployedAt: deployment.DeployedAt,
			DeployedBy: deployment.DeployedBy,
		}
	}

	respondJSON(w, responses, http.StatusOK)
}
