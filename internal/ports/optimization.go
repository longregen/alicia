package ports

import (
	"context"

	"github.com/longregen/alicia/internal/domain/models"
)

// OptimizationProgressEvent represents a progress update during optimization.
// This is the canonical event type for pub/sub progress notifications.
type OptimizationProgressEvent struct {
	Type            string             `json:"type"` // "started", "progress", "completed", "failed"
	RunID           string             `json:"run_id"`
	Iteration       int                `json:"iteration"`
	MaxIterations   int                `json:"max_iterations"`
	CurrentScore    float64            `json:"current_score"`
	BestScore       float64            `json:"best_score"`
	DimensionScores map[string]float64 `json:"dimension_scores,omitempty"`
	Status          string             `json:"status"` // running, completed, failed
	Message         string             `json:"message,omitempty"`
	Timestamp       string             `json:"timestamp"`
}

// OptimizationConfig configures the optimization process
type OptimizationConfig struct {
	// MaxIterations limits the number of optimization iterations
	MaxIterations int `json:"max_iterations"`

	// MinibatchSize for GEPA reflection
	MinibatchSize int `json:"minibatch_size"`

	// SkipPerfectScore stops early if perfect score is reached
	SkipPerfectScore bool `json:"skip_perfect_score"`

	// ParetoArchiveSize is the maximum number of elite solutions to maintain
	ParetoArchiveSize int `json:"pareto_archive_size"`

	// DimensionWeights for multi-objective optimization (keys: successRate, quality, efficiency, robustness, generalization, diversity, innovation)
	DimensionWeights map[string]float64 `json:"dimension_weights,omitempty"`
}

// OptimizationService defines the interface for prompt optimization CRUD operations.
// This interface handles data persistence and retrieval for optimization runs.
type OptimizationService interface {
	// Run management
	StartOptimizationRun(ctx context.Context, name, promptType, baselinePrompt string) (*models.OptimizationRun, error)
	GetOptimizationRun(ctx context.Context, runID string) (*models.OptimizationRun, error)
	ListOptimizationRuns(ctx context.Context, status string, limit, offset int) ([]*models.OptimizationRun, error)
	UpdateRunProgress(ctx context.Context, runID string, iteration int, bestScore float64, dimScores map[string]float64) error
	CompleteRun(ctx context.Context, runID string, bestScore float64) error
	FailRun(ctx context.Context, runID string, errorMsg string) error

	// Candidate management
	SaveCandidate(ctx context.Context, runID string, candidate *models.PromptCandidate) error

	// Evaluation management
	SaveEvaluation(ctx context.Context, candidateID string, eval *models.PromptEvaluation) error
}

// OptimizationServiceFull extends OptimizationService with additional query and management methods.
// This is the full interface implemented by the application service.
type OptimizationServiceFull interface {
	OptimizationService

	// Extended run management
	UpdateProgress(ctx context.Context, runID string, iteration int, currentScore float64) error

	// Candidate management
	AddCandidate(ctx context.Context, runID, promptText string, iteration int) (*models.PromptCandidate, error)
	GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error)
	GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error)

	// Evaluation management
	RecordEvaluation(ctx context.Context, candidateID, runID, input, output string, score float64, success bool, latencyMs int64) (*models.PromptEvaluation, error)
	GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error)

	// Optimized program retrieval
	GetOptimizedProgram(ctx context.Context, runID string) (*OptimizedProgram, error)

	// Dimension weight management - uses map for decoupling from prompt package
	SetDimensionWeights(weights map[string]float64)
	GetDimensionWeights() map[string]float64
}

// OptimizedProgram represents the result of an optimization run
type OptimizedProgram struct {
	RunID       string
	BestPrompt  string
	BestScore   float64
	Iterations  int
	CompletedAt string
	Elites      []EliteSolution // Pareto-optimal solutions
}

// EliteSolution represents an elite solution from the Pareto archive
type EliteSolution struct {
	ID          string
	Label       string
	Description string
	BestFor     string
	Scores      EliteDimensionScores
}

// EliteDimensionScores holds per-dimension performance metrics for an elite
type EliteDimensionScores struct {
	SuccessRate    float64
	Quality        float64
	Efficiency     float64
	Robustness     float64
	Generalization float64
	Diversity      float64
	Innovation     float64
}

// OptimizationProgressPublisher defines the interface for pub/sub progress notifications.
// Implementations can use WebSocket, SSE, or other transport mechanisms.
type OptimizationProgressPublisher interface {
	// Subscribe creates a subscription for progress events for a specific run
	// Returns a channel that will receive OptimizationProgressEvent updates
	Subscribe(runID string) <-chan OptimizationProgressEvent

	// Unsubscribe removes a subscription for a specific run
	// The channel should be the same one returned by Subscribe
	Unsubscribe(runID string, ch <-chan OptimizationProgressEvent)

	// PublishProgress broadcasts a progress event to all subscribers of the run
	PublishProgress(event OptimizationProgressEvent)

	// Close closes all channels for a run (called when optimization completes)
	Close(runID string)
}

// RunOptimizationInput contains the parameters for starting an optimization run
type RunOptimizationInput struct {
	// Name is a human-readable identifier for the optimization run
	Name string `json:"name"`

	// PromptType identifies the type of prompt being optimized (e.g., "tool_selection", "memory_selection")
	PromptType string `json:"prompt_type"`

	// BaselinePrompt is the initial prompt to optimize from
	BaselinePrompt string `json:"baseline_prompt"`

	// Config contains optimization parameters (iterations, minibatch size, etc.)
	Config *OptimizationConfig `json:"config,omitempty"`
}

// RunOptimizationOutput contains the result of starting an optimization run
type RunOptimizationOutput struct {
	// Run is the created optimization run
	Run *models.OptimizationRun `json:"run"`

	// ProgressChannel provides real-time progress updates
	// Consumers should read from this channel until it closes
	ProgressChannel <-chan OptimizationProgressEvent `json:"-"`
}

// RunOptimizationUseCase defines the interface for executing prompt optimization.
// This is the main entry point for running GEPA/DSPy optimization.
type RunOptimizationUseCase interface {
	// Execute starts an optimization run and returns immediately with a progress channel.
	// The optimization runs asynchronously; progress is reported via the channel.
	Execute(ctx context.Context, input *RunOptimizationInput) (*RunOptimizationOutput, error)

	// GetProgress returns a channel for receiving progress updates for an existing run.
	// Returns nil if the run doesn't exist or has already completed.
	GetProgress(runID string) <-chan OptimizationProgressEvent
}

// OptimizationProgressBroadcaster defines the interface for broadcasting optimization progress.
// This is used for WebSocket broadcast to connected clients.
type OptimizationProgressBroadcaster interface {
	// BroadcastOptimizationProgress broadcasts optimization progress to all subscribed clients
	// The runID is used to identify the optimization run being tracked
	BroadcastOptimizationProgress(runID string, progress OptimizationProgressUpdate)
}

// OptimizationProgressUpdate represents an optimization progress update for broadcasting
type OptimizationProgressUpdate struct {
	RunID           string
	Status          string // running, completed, failed
	Iteration       int
	MaxIterations   int
	CurrentScore    float64
	BestScore       float64
	DimensionScores map[string]float64
	Message         string
	Timestamp       int64
}
