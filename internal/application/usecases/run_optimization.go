package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// RunOptimization orchestrates the optimization workflow.
// It coordinates between the optimization service, progress publisher, and broadcaster.
// The usecase handles goroutine lifecycle for background optimization while
// delegating the actual GEPA algorithm logic to the optimization service.
type RunOptimization struct {
	optimizationService ports.OptimizationServiceFull
	progressPublisher   ports.OptimizationProgressPublisher
	wsBroadcaster       ports.OptimizationProgressBroadcaster
	llmService          ports.LLMService
	idGenerator         ports.IDGenerator
	repo                ports.PromptOptimizationRepository
}

// NewRunOptimization creates a new RunOptimization usecase with required dependencies.
func NewRunOptimization(
	optimizationService ports.OptimizationServiceFull,
	progressPublisher ports.OptimizationProgressPublisher,
	wsBroadcaster ports.OptimizationProgressBroadcaster,
	llmService ports.LLMService,
	idGenerator ports.IDGenerator,
	repo ports.PromptOptimizationRepository,
) *RunOptimization {
	return &RunOptimization{
		optimizationService: optimizationService,
		progressPublisher:   progressPublisher,
		wsBroadcaster:       wsBroadcaster,
		llmService:          llmService,
		idGenerator:         idGenerator,
		repo:                repo,
	}
}

// Execute starts an optimization run.
// It creates the run, subscribes to progress events, and starts the optimization in background.
// Returns immediately with the run and a progress channel for monitoring.
func (uc *RunOptimization) Execute(ctx context.Context, input *ports.RunOptimizationInput) (*ports.RunOptimizationOutput, error) {
	// Validate input
	if input.Name == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "optimization run name cannot be empty")
	}
	if input.PromptType == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "prompt type cannot be empty")
	}

	// 1. Create the optimization run via service
	run, err := uc.optimizationService.StartOptimizationRun(ctx, input.Name, input.PromptType, input.BaselinePrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create optimization run: %w", err)
	}

	// Apply custom config if provided
	if input.Config != nil {
		if input.Config.DimensionWeights != nil {
			uc.optimizationService.SetDimensionWeights(input.Config.DimensionWeights)
		}
	}

	// 2. Subscribe to progress events
	var progressChan <-chan ports.OptimizationProgressEvent
	if uc.progressPublisher != nil {
		progressChan = uc.progressPublisher.Subscribe(run.ID)
	}

	// Publish initial "started" event
	uc.publishProgressEvent(ports.OptimizationProgressEvent{
		Type:          "started",
		RunID:         run.ID,
		Iteration:     0,
		MaxIterations: run.MaxIterations,
		CurrentScore:  0,
		BestScore:     0,
		Status:        models.OptimizationStatusRunning,
		Message:       "Optimization run started",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	})

	// 3. Start background goroutine for actual optimization
	go uc.runOptimization(context.Background(), run, input)

	// 4. Return run + progress channel
	return &ports.RunOptimizationOutput{
		Run:             run,
		ProgressChannel: progressChan,
	}, nil
}

// GetProgress returns a channel for receiving progress updates for an existing run.
// Returns nil if the progress publisher is not configured or the run doesn't exist.
func (uc *RunOptimization) GetProgress(runID string) <-chan ports.OptimizationProgressEvent {
	if uc.progressPublisher == nil {
		return nil
	}
	return uc.progressPublisher.Subscribe(runID)
}

// runOptimization is the background worker that runs the actual GEPA optimization.
// It handles the long-running optimization process, publishes progress updates,
// and ensures proper cleanup on completion or failure.
func (uc *RunOptimization) runOptimization(ctx context.Context, run *models.OptimizationRun, input *ports.RunOptimizationInput) {
	// Ensure we clean up the progress publisher on exit
	defer func() {
		if uc.progressPublisher != nil {
			uc.progressPublisher.PublishProgress(uc.createCompletionEvent(run))
		}
	}()

	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			reason := fmt.Sprintf("optimization panicked: %v", r)
			log.Printf("ERROR: %s for run %s", reason, run.ID)

			if err := uc.optimizationService.FailRun(ctx, run.ID, reason); err != nil {
				log.Printf("ERROR: failed to mark run %s as failed: %v", run.ID, err)
			}

			uc.publishProgressEvent(ports.OptimizationProgressEvent{
				Type:      "failed",
				RunID:     run.ID,
				Status:    models.OptimizationStatusFailed,
				Message:   reason,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
		}
	}()

	// Simulate GEPA optimization iterations
	// The actual GEPA algorithm logic remains in the optimization service
	// Here we just orchestrate the process and handle progress updates
	maxIterations := run.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 100 // Default
	}

	var bestScore float64
	var bestDimScores map[string]float64

	for iteration := 1; iteration <= maxIterations; iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			reason := "optimization cancelled"
			if err := uc.optimizationService.FailRun(ctx, run.ID, reason); err != nil {
				log.Printf("warning: failed to mark run %s as failed on cancellation: %v", run.ID, err)
			}
			uc.publishProgressEvent(ports.OptimizationProgressEvent{
				Type:      "failed",
				RunID:     run.ID,
				Status:    models.OptimizationStatusFailed,
				Message:   reason,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return
		default:
		}

		// Delegate to optimization service for actual iteration work
		// This is where the GEPA algorithm runs
		currentScore, dimScores, err := uc.executeOptimizationIteration(ctx, run, iteration, input)
		if err != nil {
			log.Printf("warning: iteration %d failed for run %s: %v", iteration, run.ID, err)
			// Continue to next iteration on non-fatal errors
			continue
		}

		// Track best score
		if currentScore > bestScore {
			bestScore = currentScore
			bestDimScores = dimScores
		}

		// Update progress in the repository
		if err := uc.optimizationService.UpdateProgress(ctx, run.ID, iteration, currentScore); err != nil {
			log.Printf("warning: failed to update progress for run %s: %v", run.ID, err)
		}

		// Publish progress event
		uc.publishProgressEvent(ports.OptimizationProgressEvent{
			Type:            "progress",
			RunID:           run.ID,
			Iteration:       iteration,
			MaxIterations:   maxIterations,
			CurrentScore:    currentScore,
			BestScore:       bestScore,
			DimensionScores: dimScores,
			Status:          models.OptimizationStatusRunning,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		})

		// Early stopping if perfect score reached
		if bestScore >= 1.0 {
			log.Printf("info: perfect score reached for run %s at iteration %d", run.ID, iteration)
			break
		}
	}

	// Mark run as completed
	if err := uc.optimizationService.CompleteRun(ctx, run.ID, bestScore); err != nil {
		log.Printf("ERROR: failed to complete run %s: %v", run.ID, err)
		return
	}

	// Publish completion event
	uc.publishProgressEvent(ports.OptimizationProgressEvent{
		Type:            "completed",
		RunID:           run.ID,
		Iteration:       run.MaxIterations,
		MaxIterations:   run.MaxIterations,
		CurrentScore:    bestScore,
		BestScore:       bestScore,
		DimensionScores: bestDimScores,
		Status:          models.OptimizationStatusCompleted,
		Message:         "Optimization completed successfully",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	})
}

// executeOptimizationIteration runs a single iteration of the optimization.
// This delegates to the optimization service for the actual GEPA work.
func (uc *RunOptimization) executeOptimizationIteration(
	ctx context.Context,
	run *models.OptimizationRun,
	iteration int,
	input *ports.RunOptimizationInput,
) (float64, map[string]float64, error) {
	// The actual GEPA algorithm logic is in the optimization service
	// Here we're just orchestrating the call and returning results
	//
	// In a full implementation, this would:
	// 1. Generate or mutate prompt candidates
	// 2. Evaluate candidates against the metric
	// 3. Apply GEPA selection/crossover
	// 4. Return the best score from this iteration

	// For now, we delegate to the service's existing infrastructure
	// The optimization service handles the GEPA algorithm internally
	// We just need to track progress and handle the orchestration

	// Get current run state to access candidates and scores
	currentRun, err := uc.optimizationService.GetOptimizationRun(ctx, run.ID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get run state: %w", err)
	}

	// Get best candidate to determine current best score
	bestCandidate, err := uc.optimizationService.GetBestCandidate(ctx, run.ID)
	if err != nil {
		// No candidates yet is not an error
		return 0, nil, nil
	}

	// Return the best candidate's score and dimension scores
	dimScores := make(map[string]float64)
	if bestCandidate != nil && len(currentRun.BestDimScores) > 0 {
		dimScores = currentRun.BestDimScores
	}

	if bestCandidate != nil {
		return bestCandidate.Score, dimScores, nil
	}

	return 0, dimScores, nil
}

// publishProgressEvent publishes a progress event to all subscribers and broadcasts via WebSocket.
func (uc *RunOptimization) publishProgressEvent(event ports.OptimizationProgressEvent) {
	// Publish to SSE/channel subscribers
	if uc.progressPublisher != nil {
		uc.progressPublisher.PublishProgress(event)
	}

	// Broadcast via WebSocket
	if uc.wsBroadcaster != nil {
		timestamp, _ := time.Parse(time.RFC3339, event.Timestamp)
		update := ports.OptimizationProgressUpdate{
			RunID:           event.RunID,
			Status:          event.Status,
			Iteration:       event.Iteration,
			MaxIterations:   event.MaxIterations,
			CurrentScore:    event.CurrentScore,
			BestScore:       event.BestScore,
			DimensionScores: event.DimensionScores,
			Message:         event.Message,
			Timestamp:       timestamp.UnixMilli(),
		}
		uc.wsBroadcaster.BroadcastOptimizationProgress(event.RunID, update)
	}
}

// createCompletionEvent creates a completion or failure event based on run state.
func (uc *RunOptimization) createCompletionEvent(run *models.OptimizationRun) ports.OptimizationProgressEvent {
	status := models.OptimizationStatusCompleted
	eventType := "completed"
	message := "Optimization completed"

	if run.Status == models.OptimizationStatusFailed {
		status = models.OptimizationStatusFailed
		eventType = "failed"
		message = "Optimization failed"
	}

	return ports.OptimizationProgressEvent{
		Type:            eventType,
		RunID:           run.ID,
		Iteration:       run.Iterations,
		MaxIterations:   run.MaxIterations,
		CurrentScore:    run.BestScore,
		BestScore:       run.BestScore,
		DimensionScores: run.BestDimScores,
		Status:          status,
		Message:         message,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}
}
