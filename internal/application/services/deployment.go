package services

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// DeploymentService manages deployment of optimized prompts to production
type DeploymentService struct {
	repo        ports.PromptOptimizationRepository
	idGenerator ports.IDGenerator
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	repo ports.PromptOptimizationRepository,
	idGenerator ports.IDGenerator,
) *DeploymentService {
	return &DeploymentService{
		repo:        repo,
		idGenerator: idGenerator,
	}
}

// DeploymentStatus represents the deployment status of an optimized prompt
type DeploymentStatus struct {
	ID          string
	RunID       string
	PromptType  string
	IsActive    bool
	Prompt      string
	Score       float64
	Dimensions  map[string]float64
	DeployedAt  string
	DeployedBy  string
}

// DeployOptimizedPrompt deploys an optimized prompt to production
func (s *DeploymentService) DeployOptimizedPrompt(
	ctx context.Context,
	runID string,
	userID string,
) (*DeploymentStatus, error) {
	if runID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	// Get the optimization run
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get optimization run")
	}

	// Verify run is completed
	if run.Status != models.OptimizationStatusCompleted {
		return nil, domain.NewDomainError(
			domain.ErrInvalidState,
			fmt.Sprintf("optimization run is not completed (status: %s)", run.Status),
		)
	}

	// Get the best candidate
	candidate, err := s.repo.GetBestCandidate(ctx, runID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get best candidate")
	}

	// Create deployment record
	deployment := &DeploymentStatus{
		ID:         s.idGenerator.GenerateOptimizationRunID(), // Reuse ID generator
		RunID:      runID,
		PromptType: run.PromptType,
		IsActive:   true,
		Prompt:     candidate.PromptText,
		Score:      run.BestScore,
		Dimensions: run.BestDimScores,
		DeployedBy: userID,
	}

	// In a real implementation, this would:
	// 1. Store deployment record in database
	// 2. Update active prompt configuration
	// 3. Invalidate caches
	// 4. Notify services of new prompt
	//
	// For now, we store it in the run's metadata
	if run.Meta == nil {
		run.Meta = make(map[string]any)
	}

	run.Meta["deployed"] = true
	run.Meta["deployed_by"] = userID
	run.Meta["active_candidate_id"] = candidate.ID

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return nil, domain.NewDomainError(err, "failed to update deployment status")
	}

	return deployment, nil
}

// GetActiveDeployment returns the currently active deployment for a prompt type
func (s *DeploymentService) GetActiveDeployment(
	ctx context.Context,
	promptType string,
) (*DeploymentStatus, error) {
	if promptType == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "prompt type cannot be empty")
	}

	// List all completed runs for this prompt type
	runs, err := s.repo.ListRuns(ctx, ports.ListOptimizationRunsOptions{
		Status: models.OptimizationStatusCompleted,
		Limit:  100,
	})
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list optimization runs")
	}

	// Find the most recently deployed run for this prompt type
	var activeRun *models.OptimizationRun
	for _, run := range runs {
		if run.PromptType == promptType {
			if deployed, ok := run.Meta["deployed"].(bool); ok && deployed {
				if activeRun == nil || run.UpdatedAt.After(activeRun.UpdatedAt) {
					activeRun = run
				}
			}
		}
	}

	if activeRun == nil {
		return nil, domain.NewDomainError(domain.ErrNotFound, "no active deployment found for prompt type")
	}

	// Get the active candidate
	candidateID, ok := activeRun.Meta["active_candidate_id"].(string)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "active candidate ID not found")
	}

	candidates, err := s.repo.GetCandidates(ctx, activeRun.ID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get candidates")
	}

	var activeCandidate *models.PromptCandidate
	for _, c := range candidates {
		if c.ID == candidateID {
			activeCandidate = c
			break
		}
	}

	if activeCandidate == nil {
		return nil, domain.NewDomainError(domain.ErrNotFound, "active candidate not found")
	}

	deployedBy := ""
	if by, ok := activeRun.Meta["deployed_by"].(string); ok {
		deployedBy = by
	}

	deployment := &DeploymentStatus{
		ID:         candidateID,
		RunID:      activeRun.ID,
		PromptType: activeRun.PromptType,
		IsActive:   true,
		Prompt:     activeCandidate.PromptText,
		Score:      activeRun.BestScore,
		Dimensions: activeRun.BestDimScores,
		DeployedBy: deployedBy,
	}

	if activeRun.CompletedAt != nil {
		deployment.DeployedAt = activeRun.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	return deployment, nil
}

// RollbackDeployment deactivates the current deployment
func (s *DeploymentService) RollbackDeployment(
	ctx context.Context,
	runID string,
	userID string,
) error {
	if runID == "" {
		return domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return domain.NewDomainError(err, "failed to get optimization run")
	}

	if run.Meta == nil {
		run.Meta = make(map[string]any)
	}

	run.Meta["deployed"] = false
	run.Meta["rollback_by"] = userID

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return domain.NewDomainError(err, "failed to update deployment status")
	}

	return nil
}

// ListDeploymentHistory returns the deployment history for a prompt type
func (s *DeploymentService) ListDeploymentHistory(
	ctx context.Context,
	promptType string,
	limit int,
) ([]*DeploymentStatus, error) {
	if promptType == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "prompt type cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	runs, err := s.repo.ListRuns(ctx, ports.ListOptimizationRunsOptions{
		Status: models.OptimizationStatusCompleted,
		Limit:  limit,
	})
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list optimization runs")
	}

	var history []*DeploymentStatus
	for _, run := range runs {
		if run.PromptType != promptType {
			continue
		}

		deployed, _ := run.Meta["deployed"].(bool)
		if !deployed {
			continue
		}

		candidateID, ok := run.Meta["active_candidate_id"].(string)
		if !ok {
			continue
		}

		candidates, err := s.repo.GetCandidates(ctx, run.ID)
		if err != nil {
			continue
		}

		var candidate *models.PromptCandidate
		for _, c := range candidates {
			if c.ID == candidateID {
				candidate = c
				break
			}
		}

		if candidate == nil {
			continue
		}

		deployedBy := ""
		if by, ok := run.Meta["deployed_by"].(string); ok {
			deployedBy = by
		}

		deployment := &DeploymentStatus{
			ID:         candidateID,
			RunID:      run.ID,
			PromptType: run.PromptType,
			IsActive:   deployed,
			Prompt:     candidate.PromptText,
			Score:      run.BestScore,
			Dimensions: run.BestDimScores,
			DeployedBy: deployedBy,
		}

		if run.CompletedAt != nil {
			deployment.DeployedAt = run.CompletedAt.Format("2006-01-02T15:04:05Z")
		}

		history = append(history, deployment)
	}

	return history, nil
}
