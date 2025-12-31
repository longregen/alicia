package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
	"github.com/XiaoConstantine/dspy-go/pkg/optimizers"
	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// OptimizationConfig configures the optimization service
type OptimizationConfig struct {
	// MaxIterations limits the number of optimization iterations
	MaxIterations int

	// MinibatchSize for GEPA reflection
	MinibatchSize int

	// SkipPerfectScore stops early if perfect score is reached
	SkipPerfectScore bool

	// ParetoArchiveSize is the maximum number of elite solutions to maintain
	ParetoArchiveSize int

	// DimensionWeights for multi-objective optimization
	DimensionWeights prompt.DimensionWeights
}

// DefaultOptimizationConfig returns sensible defaults
func DefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		MaxIterations:     100,
		MinibatchSize:     5,
		SkipPerfectScore:  true,
		ParetoArchiveSize: 50,
		DimensionWeights:  prompt.DefaultWeights(),
	}
}

// OptimizationProgressEvent represents a progress update during optimization
type OptimizationProgressEvent struct {
	Type            string             `json:"type"`
	RunID           string             `json:"run_id"`
	Iteration       int                `json:"iteration"`
	MaxIterations   int                `json:"max_iterations"`
	CurrentScore    float64            `json:"current_score"`
	BestScore       float64            `json:"best_score"`
	DimensionScores map[string]float64 `json:"dimension_scores,omitempty"`
	Status          string             `json:"status"`
	Message         string             `json:"message,omitempty"`
	Timestamp       string             `json:"timestamp"`
}

// OptimizationService manages prompt optimization using DSPy/GEPA
type OptimizationService struct {
	repo         ports.PromptOptimizationRepository
	llmService   ports.LLMService
	reflectionLM ports.LLMService // Strong model for GEPA reflection
	idGenerator  ports.IDGenerator
	config       OptimizationConfig

	// Progress channel infrastructure for push-based updates
	progressChannels map[string][]chan OptimizationProgressEvent
	progressMu       sync.RWMutex
}

// NewOptimizationService creates a new optimization service
func NewOptimizationService(
	repo ports.PromptOptimizationRepository,
	llmService ports.LLMService,
	idGenerator ports.IDGenerator,
	config OptimizationConfig,
) *OptimizationService {
	return &OptimizationService{
		repo:             repo,
		llmService:       llmService,
		idGenerator:      idGenerator,
		config:           config,
		progressChannels: make(map[string][]chan OptimizationProgressEvent),
	}
}

// WithReflectionLM sets a separate LLM for GEPA reflection (typically a stronger model)
func (s *OptimizationService) WithReflectionLM(reflectionLM ports.LLMService) *OptimizationService {
	s.reflectionLM = reflectionLM
	return s
}

// StartOptimizationRun creates a new optimization run
func (s *OptimizationService) StartOptimizationRun(
	ctx context.Context,
	name string,
	promptType string,
	baselinePrompt string,
) (*models.OptimizationRun, error) {
	if name == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "optimization run name cannot be empty")
	}

	if promptType == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "prompt type cannot be empty")
	}

	run := models.NewOptimizationRun(
		s.idGenerator.GenerateOptimizationRunID(),
		name,
		promptType,
		s.config.MaxIterations,
	)
	run.Config = map[string]any{
		"minibatch_size":  s.config.MinibatchSize,
		"skip_perfect":    s.config.SkipPerfectScore,
		"baseline_prompt": baselinePrompt,
	}

	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, domain.NewDomainError(err, "failed to create optimization run")
	}

	return run, nil
}

// GetOptimizationRun retrieves an optimization run by ID
func (s *OptimizationService) GetOptimizationRun(ctx context.Context, id string) (*models.OptimizationRun, error) {
	if id == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	run, err := s.repo.GetRun(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get optimization run")
	}

	return run, nil
}

// ListOptimizationRuns retrieves a list of optimization runs with filtering and pagination
func (s *OptimizationService) ListOptimizationRuns(ctx context.Context, opts ports.ListOptimizationRunsOptions) ([]*models.OptimizationRun, error) {
	runs, err := s.repo.ListRuns(ctx, opts)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list optimization runs")
	}

	return runs, nil
}

// AddCandidate adds a new prompt candidate to an optimization run
func (s *OptimizationService) AddCandidate(
	ctx context.Context,
	runID string,
	promptText string,
	iteration int,
) (*models.PromptCandidate, error) {
	if runID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	if promptText == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "prompt text cannot be empty")
	}

	candidate := models.NewPromptCandidate(
		s.idGenerator.GeneratePromptCandidateID(),
		runID,
		iteration,
		promptText,
	)

	if err := s.repo.SaveCandidate(ctx, runID, candidate); err != nil {
		return nil, domain.NewDomainError(err, "failed to save prompt candidate")
	}

	return candidate, nil
}

// RecordEvaluation records an evaluation result for a candidate
func (s *OptimizationService) RecordEvaluation(
	ctx context.Context,
	candidateID string,
	runID string,
	input string,
	output string,
	score float64,
	success bool,
	latencyMs int64,
) (*models.PromptEvaluation, error) {
	if candidateID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "candidate ID cannot be empty")
	}

	eval := models.NewPromptEvaluation(
		s.idGenerator.GeneratePromptEvaluationID(),
		candidateID,
		runID,
		input,
		output,
		score,
		success,
		latencyMs,
	)

	if err := s.repo.SaveEvaluation(ctx, eval); err != nil {
		return nil, domain.NewDomainError(err, "failed to save evaluation")
	}

	return eval, nil
}

// RecordEvaluationWithDimensions records an evaluation result with per-dimension scores
func (s *OptimizationService) RecordEvaluationWithDimensions(
	ctx context.Context,
	candidateID string,
	runID string,
	input string,
	output string,
	dimScores prompt.DimensionScores,
	success bool,
	latencyMs int64,
) (*models.PromptEvaluation, error) {
	if candidateID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "candidate ID cannot be empty")
	}

	// Calculate weighted score from dimensions
	score := dimScores.WeightedScore(s.config.DimensionWeights)

	eval := models.NewPromptEvaluation(
		s.idGenerator.GeneratePromptEvaluationID(),
		candidateID,
		runID,
		input,
		output,
		score,
		success,
		latencyMs,
	)

	// Set dimension scores
	eval.DimensionScores = map[string]float64{
		"successRate":    dimScores.SuccessRate,
		"quality":        dimScores.Quality,
		"efficiency":     dimScores.Efficiency,
		"robustness":     dimScores.Robustness,
		"generalization": dimScores.Generalization,
		"diversity":      dimScores.Diversity,
		"innovation":     dimScores.Innovation,
	}

	if err := s.repo.SaveEvaluation(ctx, eval); err != nil {
		return nil, domain.NewDomainError(err, "failed to save evaluation")
	}

	return eval, nil
}

// GetCandidates retrieves all candidates for a run
func (s *OptimizationService) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	if runID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	candidates, err := s.repo.GetCandidates(ctx, runID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get candidates")
	}

	return candidates, nil
}

// GetBestCandidate retrieves the best performing candidate for a run
func (s *OptimizationService) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	if runID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	candidate, err := s.repo.GetBestCandidate(ctx, runID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get best candidate")
	}

	return candidate, nil
}

// GetEvaluations retrieves all evaluations for a candidate
func (s *OptimizationService) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	if candidateID == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "candidate ID cannot be empty")
	}

	evals, err := s.repo.GetEvaluations(ctx, candidateID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get evaluations")
	}

	return evals, nil
}

// CompleteRun marks an optimization run as completed
func (s *OptimizationService) CompleteRun(ctx context.Context, runID string, bestScore float64) error {
	if runID == "" {
		return domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return domain.NewDomainError(err, "failed to get optimization run")
	}

	run.BestScore = bestScore
	run.MarkCompleted()

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return domain.NewDomainError(err, "failed to update optimization run")
	}

	return nil
}

// FailRun marks an optimization run as failed
func (s *OptimizationService) FailRun(ctx context.Context, runID string, reason string) error {
	if runID == "" {
		return domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return domain.NewDomainError(err, "failed to get optimization run")
	}

	// Store the failure reason in config
	if run.Config == nil {
		run.Config = make(map[string]any)
	}
	run.Config["failure_reason"] = reason
	run.MarkFailed()

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return domain.NewDomainError(err, "failed to update optimization run")
	}

	return nil
}

// UpdateProgress updates the progress of an optimization run
func (s *OptimizationService) UpdateProgress(ctx context.Context, runID string, iteration int, currentScore float64) error {
	if runID == "" {
		return domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return domain.NewDomainError(err, "failed to get optimization run")
	}

	run.Iterations = iteration
	if currentScore > run.BestScore {
		run.BestScore = currentScore
	}

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return domain.NewDomainError(err, "failed to update optimization run")
	}

	return nil
}

// UpdateProgressWithDimensions updates the progress of an optimization run with dimension scores
func (s *OptimizationService) UpdateProgressWithDimensions(
	ctx context.Context,
	runID string,
	iteration int,
	dimScores prompt.DimensionScores,
) error {
	if runID == "" {
		return domain.NewDomainError(domain.ErrEmptyContent, "run ID cannot be empty")
	}

	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return domain.NewDomainError(err, "failed to get optimization run")
	}

	run.Iterations = iteration

	// Calculate weighted score
	currentScore := dimScores.WeightedScore(s.config.DimensionWeights)
	if currentScore > run.BestScore {
		run.BestScore = currentScore
		run.BestDimScores = map[string]float64{
			"successRate":    dimScores.SuccessRate,
			"quality":        dimScores.Quality,
			"efficiency":     dimScores.Efficiency,
			"robustness":     dimScores.Robustness,
			"generalization": dimScores.Generalization,
			"diversity":      dimScores.Diversity,
			"innovation":     dimScores.Innovation,
		}
	}

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return domain.NewDomainError(err, "failed to update optimization run")
	}

	return nil
}

// ApplyFeedbackToWeights applies user feedback to adjust dimension weights
func (s *OptimizationService) ApplyFeedbackToWeights(feedbackType prompt.FeedbackType) {
	adjustment := prompt.MapFeedbackToDimensions(feedbackType)
	s.config.DimensionWeights = prompt.ApplyAdjustment(s.config.DimensionWeights, adjustment)
}

// GetDimensionWeights returns the current dimension weights as a map
func (s *OptimizationService) GetDimensionWeights() map[string]float64 {
	return s.config.DimensionWeights.ToMap()
}

// SetDimensionWeights sets the dimension weights from a map
func (s *OptimizationService) SetDimensionWeights(weights map[string]float64) {
	dw := prompt.DimensionWeightsFromMap(weights)
	dw.Normalize()
	s.config.DimensionWeights = dw
}

// SubscribeProgress subscribes to progress updates for a given run ID
// Returns a channel that will receive progress events
func (s *OptimizationService) SubscribeProgress(runID string) <-chan OptimizationProgressEvent {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()

	// Create a buffered channel to prevent blocking the publisher
	ch := make(chan OptimizationProgressEvent, 100)
	s.progressChannels[runID] = append(s.progressChannels[runID], ch)
	return ch
}

// UnsubscribeProgress removes a subscription for a given run ID
func (s *OptimizationService) UnsubscribeProgress(runID string, ch <-chan OptimizationProgressEvent) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()

	channels := s.progressChannels[runID]
	for i, subscriberCh := range channels {
		if subscriberCh == ch {
			// Remove this channel from the slice
			s.progressChannels[runID] = append(channels[:i], channels[i+1:]...)
			close(subscriberCh)
			break
		}
	}

	// Clean up the map entry if no more subscribers
	if len(s.progressChannels[runID]) == 0 {
		delete(s.progressChannels, runID)
	}
}

// publishProgress publishes a progress event to all subscribers of a run
func (s *OptimizationService) publishProgress(runID string, event OptimizationProgressEvent) {
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()

	channels := s.progressChannels[runID]
	for _, ch := range channels {
		// Non-blocking send to prevent slow subscribers from blocking the optimization
		select {
		case ch <- event:
		default:
			// Channel buffer is full, skip this update
		}
	}
}

// OptimizeFromVotes runs GEPA optimization using vote-derived training data
func (s *OptimizationService) OptimizeFromVotes(
	ctx context.Context,
	taskType string,
	trainingBuilder *TrainingSetBuilderService,
) (*models.OptimizationRun, error) {
	var sig prompt.Signature
	var metric prompt.Metric
	var trainset, valset []prompt.Example
	var err error

	switch taskType {
	case models.TaskTypeToolSelection:
		sig = baselines.ToolSelectionSignature
		metric = baselines.NewToolSelectionMetric(nil)
		trainset, valset, err = trainingBuilder.GetOrBuildToolSelectionDataset(ctx)
	case models.TaskTypeMemorySelection:
		sig = baselines.MemorySelectionSignature
		metric = baselines.NewMemorySelectionMetric(nil)
		trainset, valset, err = trainingBuilder.GetOrBuildMemorySelectionDataset(ctx)
	case models.TaskTypeMemoryExtraction:
		sig = baselines.MemoryExtractionSignature
		metric = baselines.NewMemoryExtractionMetric(nil)
		trainset, valset, err = trainingBuilder.GetOrBuildMemoryExtractionDataset(ctx)
	default:
		return nil, domain.NewDomainError(domain.ErrInvalidState, fmt.Sprintf("unknown task type: %s", taskType))
	}

	if err != nil {
		return nil, domain.NewDomainError(err, fmt.Sprintf("failed to build dataset for task type %s", taskType))
	}

	return s.OptimizeSignature(ctx, sig, trainset, valset, metric)
}

// OptimizeSignature runs GEPA optimization on a signature
// This is the main entry point for automatic prompt optimization
func (s *OptimizationService) OptimizeSignature(
	ctx context.Context,
	sig prompt.Signature,
	trainset []prompt.Example,
	valset []prompt.Example,
	metric prompt.Metric,
) (*models.OptimizationRun, error) {
	if len(trainset) == 0 {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "training set cannot be empty")
	}

	// Create optimization run
	run, err := s.StartOptimizationRun(
		ctx,
		sig.Name,
		"signature",
		"", // baseline will be set by GEPA
	)
	if err != nil {
		return nil, err
	}

	// Store dimension weights in the run
	run.DimensionWeights = map[string]float64{
		"successRate":    s.config.DimensionWeights.SuccessRate,
		"quality":        s.config.DimensionWeights.Quality,
		"efficiency":     s.config.DimensionWeights.Efficiency,
		"robustness":     s.config.DimensionWeights.Robustness,
		"generalization": s.config.DimensionWeights.Generalization,
		"diversity":      s.config.DimensionWeights.Diversity,
		"innovation":     s.config.DimensionWeights.Innovation,
	}
	_ = s.repo.UpdateRun(ctx, run)

	// Get the LLM adapter for dspy-go and register it as the default
	llmAdapter := prompt.NewLLMServiceAdapter(s.llmService)
	core.SetDefaultLLM(llmAdapter)

	// If we have a separate reflection LLM, set it as the teacher LLM for GEPA reflection
	if s.reflectionLM != nil {
		reflectionAdapter := prompt.NewLLMServiceAdapter(s.reflectionLM)
		core.GlobalConfig.TeacherLLM = reflectionAdapter
	}

	// Create the base module and wrap it in a Program for GEPA
	baseModule := prompt.NewAliciaPredict(sig)
	program := baseModule.ToProgram(sig.Name)

	// Create dataset adapter for dspy-go
	dataset := prompt.NewDatasetAdapter(trainset)

	// Create metric adapter for dspy-go
	metricAdapter := prompt.NewMetricAdapter(metric)
	coreMetric := metricAdapter.ToCoreMetric()

	// Run optimization in a goroutine (it can be long-running)
	go func() {
		optimizeCtx := context.Background() // Use background context for long-running operation

		defer func() {
			if r := recover(); r != nil {
				reason := fmt.Sprintf("optimization panicked: %v", r)
				_ = s.FailRun(optimizeCtx, run.ID, reason)

				// Publish failure event to subscribers
				s.publishProgress(run.ID, OptimizationProgressEvent{
					Type:      "failed",
					RunID:     run.ID,
					Status:    string(models.OptimizationStatusFailed),
					Message:   reason,
					Timestamp: time.Now().Format(time.RFC3339),
				})
			}
		}()

		// Configure GEPA with Alicia-specific settings
		gepaConfig := &optimizers.GEPAConfig{
			// Map MaxIterations to GEPA generations
			// GEPA uses population-based evolution, so fewer generations are needed
			MaxGenerations:       s.config.MaxIterations / 10, // ~10 iterations per generation
			PopulationSize:       20,
			MutationRate:         0.3,
			CrossoverRate:        0.7,
			ElitismRate:          0.1,
			ReflectionFreq:       2,
			ReflectionDepth:      3,
			SelfCritiqueTemp:     0.7,
			TournamentSize:       3,
			SelectionStrategy:    "adaptive_pareto", // Multi-objective Pareto selection
			ConvergenceThreshold: 0.01,
			StagnationLimit:      3,
			EvaluationBatchSize:  s.config.MinibatchSize,
			ConcurrencyLevel:     3,
			Temperature:          0.8,
			MaxTokens:            500,
		}

		// Ensure at least 1 generation
		if gepaConfig.MaxGenerations < 1 {
			gepaConfig.MaxGenerations = 1
		}

		// Create the GEPA optimizer
		gepaOptimizer, err := optimizers.NewGEPA(gepaConfig)
		if err != nil {
			reason := fmt.Sprintf("failed to create GEPA optimizer: %v", err)
			_ = s.FailRun(optimizeCtx, run.ID, reason)

			// Publish failure event to subscribers
			s.publishProgress(run.ID, OptimizationProgressEvent{
				Type:      "failed",
				RunID:     run.ID,
				Status:    string(models.OptimizationStatusFailed),
				Message:   reason,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			return
		}

		// Run GEPA optimization
		optimizedProgram, err := gepaOptimizer.Compile(optimizeCtx, program, dataset, coreMetric)
		if err != nil {
			reason := fmt.Sprintf("GEPA optimization failed: %v", err)
			_ = s.FailRun(optimizeCtx, run.ID, reason)

			// Publish failure event to subscribers
			s.publishProgress(run.ID, OptimizationProgressEvent{
				Type:      "failed",
				RunID:     run.ID,
				Status:    string(models.OptimizationStatusFailed),
				Message:   reason,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			return
		}

		// Get the GEPA optimization state to extract results
		gepaState := gepaOptimizer.GetOptimizationState()

		// Extract best candidate from GEPA state
		bestPrompt := ""
		bestScore := 0.0
		bestDimScores := prompt.DimensionScores{}

		if gepaState != nil && gepaState.BestCandidate != nil {
			bestPrompt = gepaState.BestCandidate.Instruction
			bestScore = gepaState.BestCandidate.Fitness

			// Extract GEPA's multi-objective fitness for the best candidate
			if moFitness, ok := gepaState.MultiObjectiveFitnessMap[gepaState.BestCandidate.ID]; ok && moFitness != nil {
				bestDimScores = prompt.DimensionScores{
					SuccessRate:    moFitness.SuccessRate,
					Quality:        moFitness.OutputQuality,
					Efficiency:     moFitness.Efficiency,
					Robustness:     moFitness.Robustness,
					Generalization: moFitness.Generalization,
					Diversity:      moFitness.Diversity,
					Innovation:     moFitness.Innovation,
				}
			} else {
				// Fallback to basic fitness if multi-objective data unavailable
				bestDimScores = prompt.DimensionScores{
					SuccessRate:    bestScore,
					Quality:        bestScore * 0.9,
					Efficiency:     0.8,
					Robustness:     bestScore * 0.85,
					Generalization: bestScore * 0.8,
					Diversity:      0.5,
					Innovation:     0.3,
				}
			}
		}

		// Store Pareto archive solutions as candidates
		paretoArchive := gepaState.GetParetoArchive()
		for i, candidate := range paretoArchive {
			savedCandidate, err := s.AddCandidate(
				optimizeCtx,
				run.ID,
				candidate.Instruction,
				candidate.Generation,
			)
			if err != nil {
				continue
			}

			// Record evaluation for this candidate
			// Extract multi-objective fitness from archive fitness map
			candidateDimScores := prompt.DimensionScores{}
			if moFitness, ok := gepaState.ArchiveFitnessMap[candidate.ID]; ok && moFitness != nil {
				candidateDimScores = prompt.DimensionScores{
					SuccessRate:    moFitness.SuccessRate,
					Quality:        moFitness.OutputQuality,
					Efficiency:     moFitness.Efficiency,
					Robustness:     moFitness.Robustness,
					Generalization: moFitness.Generalization,
					Diversity:      moFitness.Diversity,
					Innovation:     moFitness.Innovation,
				}
			} else {
				// Fallback to basic fitness
				candidateDimScores = prompt.DimensionScores{
					SuccessRate:    candidate.Fitness,
					Quality:        candidate.Fitness * 0.9,
					Efficiency:     0.8,
					Robustness:     candidate.Fitness * 0.85,
					Generalization: candidate.Fitness * 0.8,
					Diversity:      0.5,
					Innovation:     0.3,
				}
			}

			_, _ = s.RecordEvaluationWithDimensions(
				optimizeCtx,
				savedCandidate.ID,
				run.ID,
				fmt.Sprintf("pareto_%d", i),
				"",
				candidateDimScores,
				candidate.Fitness > 0.5,
				0,
			)

			// Update progress periodically
			if i%5 == 0 {
				_ = s.UpdateProgressWithDimensions(optimizeCtx, run.ID, i, candidateDimScores)

				// Publish progress event to subscribers
				s.publishProgress(run.ID, OptimizationProgressEvent{
					Type:          "progress",
					RunID:         run.ID,
					Iteration:     i,
					MaxIterations: run.MaxIterations,
					CurrentScore:  candidate.Fitness,
					BestScore:     bestScore,
					DimensionScores: map[string]float64{
						"successRate":    candidateDimScores.SuccessRate,
						"quality":        candidateDimScores.Quality,
						"efficiency":     candidateDimScores.Efficiency,
						"robustness":     candidateDimScores.Robustness,
						"generalization": candidateDimScores.Generalization,
						"diversity":      candidateDimScores.Diversity,
						"innovation":     candidateDimScores.Innovation,
					},
					Status:    string(models.OptimizationStatusRunning),
					Timestamp: time.Now().Format(time.RFC3339),
				})
			}
		}

		// Validation phase on optimized program
		if len(valset) > 0 && optimizedProgram.Forward != nil {
			valDimScores := prompt.DimensionScores{}
			successCount := 0

			for _, example := range valset {
				outputs, err := optimizedProgram.Execute(optimizeCtx, prompt.ConvertToInterfaceMap(example.Inputs))
				if err != nil {
					continue
				}

				predExample := prompt.Example{
					Inputs:  example.Inputs,
					Outputs: prompt.ConvertFromInterfaceMap(outputs),
				}

				scoreResult, err := metric.Score(optimizeCtx, example, predExample, nil)
				if err != nil {
					continue
				}

				exampleDimScores := calculateDimensionScores(scoreResult, example, predExample)
				valDimScores.SuccessRate += exampleDimScores.SuccessRate
				valDimScores.Quality += exampleDimScores.Quality
				valDimScores.Efficiency += exampleDimScores.Efficiency
				valDimScores.Robustness += exampleDimScores.Robustness
				valDimScores.Generalization += exampleDimScores.Generalization
				valDimScores.Diversity += exampleDimScores.Diversity
				valDimScores.Innovation += exampleDimScores.Innovation
				successCount++
			}

			if successCount > 0 {
				n := float64(successCount)
				valDimScores.SuccessRate /= n
				valDimScores.Quality /= n
				valDimScores.Efficiency /= n
				valDimScores.Robustness /= n
				valDimScores.Generalization /= n
				valDimScores.Diversity /= n
				valDimScores.Innovation /= n

				valScore := valDimScores.WeightedScore(s.config.DimensionWeights)

				// Use validation score as final best score if worse
				if valScore < bestScore {
					bestScore = valScore
					bestDimScores = valDimScores
				}
			}
		}

		// Mark run as completed with best dimension scores
		run, _ := s.repo.GetRun(optimizeCtx, run.ID)
		if run != nil {
			run.BestScore = bestScore
			run.BestDimScores = map[string]float64{
				"successRate":    bestDimScores.SuccessRate,
				"quality":        bestDimScores.Quality,
				"efficiency":     bestDimScores.Efficiency,
				"robustness":     bestDimScores.Robustness,
				"generalization": bestDimScores.Generalization,
				"diversity":      bestDimScores.Diversity,
				"innovation":     bestDimScores.Innovation,
			}
			run.MarkCompleted()
			_ = s.repo.UpdateRun(optimizeCtx, run)

			// Publish completion event to subscribers
			s.publishProgress(run.ID, OptimizationProgressEvent{
				Type:          "completed",
				RunID:         run.ID,
				Iteration:     run.Iterations,
				MaxIterations: run.MaxIterations,
				CurrentScore:  bestScore,
				BestScore:     bestScore,
				DimensionScores: map[string]float64{
					"successRate":    bestDimScores.SuccessRate,
					"quality":        bestDimScores.Quality,
					"efficiency":     bestDimScores.Efficiency,
					"robustness":     bestDimScores.Robustness,
					"generalization": bestDimScores.Generalization,
					"diversity":      bestDimScores.Diversity,
					"innovation":     bestDimScores.Innovation,
				},
				Status:    string(models.OptimizationStatusCompleted),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}

		// Suppress unused variable warning for optimizedProgram
		_ = bestPrompt
	}()

	return run, nil
}

// calculateDimensionScores derives dimension scores from a metric result
func calculateDimensionScores(scoreResult prompt.ScoreWithFeedback, expected, predicted prompt.Example) prompt.DimensionScores {
	// Base the dimension scores on the metric result
	baseScore := scoreResult.Score

	// Success rate is directly from the score
	successRate := baseScore

	// Quality is based on score and any quality indicators in the result
	quality := baseScore

	// Efficiency is a baseline (can be enhanced with latency tracking)
	efficiency := 0.8 // Default reasonable efficiency

	// Robustness starts at a baseline
	robustness := baseScore * 0.9

	// Generalization is based on how well it handles varied inputs
	generalization := baseScore * 0.85

	// Diversity and innovation are harder to measure automatically
	// These could be enhanced with output analysis
	diversity := 0.5
	innovation := 0.3

	return prompt.DimensionScores{
		SuccessRate:    successRate,
		Quality:        quality,
		Efficiency:     efficiency,
		Robustness:     robustness,
		Generalization: generalization,
		Diversity:      diversity,
		Innovation:     innovation,
	}
}

// OptimizeSignatureWithMemory runs GEPA optimization with memory-augmented few-shot learning
// This variant uses the MemoryService to retrieve relevant examples and enrich the training set
func (s *OptimizationService) OptimizeSignatureWithMemory(
	ctx context.Context,
	sig prompt.Signature,
	trainset []prompt.Example,
	valset []prompt.Example,
	metric prompt.Metric,
	memoryService ports.MemoryService,
	memoryOptions ...prompt.MemoryAwareOption,
) (*models.OptimizationRun, error) {
	if len(trainset) == 0 {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "training set cannot be empty")
	}

	if memoryService == nil {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "memory service is required for memory-aware optimization")
	}

	// Create optimization run
	run, err := s.StartOptimizationRun(
		ctx,
		sig.Name+" (memory-aware)",
		"signature_memory_aware",
		"", // baseline will be set by GEPA
	)
	if err != nil {
		return nil, err
	}

	// Store dimension weights and memory config in the run
	run.DimensionWeights = map[string]float64{
		"successRate":    s.config.DimensionWeights.SuccessRate,
		"quality":        s.config.DimensionWeights.Quality,
		"efficiency":     s.config.DimensionWeights.Efficiency,
		"robustness":     s.config.DimensionWeights.Robustness,
		"generalization": s.config.DimensionWeights.Generalization,
		"diversity":      s.config.DimensionWeights.Diversity,
		"innovation":     s.config.DimensionWeights.Innovation,
	}
	if run.Config == nil {
		run.Config = make(map[string]any)
	}
	run.Config["memory_aware"] = true
	_ = s.repo.UpdateRun(ctx, run)

	// Get the LLM adapter for dspy-go and register it as the default
	llmAdapter := prompt.NewLLMServiceAdapter(s.llmService)
	core.SetDefaultLLM(llmAdapter)

	// If we have a separate reflection LLM, set it as the teacher LLM for GEPA reflection
	if s.reflectionLM != nil {
		reflectionAdapter := prompt.NewLLMServiceAdapter(s.reflectionLM)
		core.GlobalConfig.TeacherLLM = reflectionAdapter
	}

	// Create memory-aware module instead of base AliciaPredict
	memoryModule := prompt.NewMemoryAwareModule(sig, memoryService, memoryOptions...)
	program := memoryModule.ToProgram(sig.Name)

	// Create dataset adapter for dspy-go
	dataset := prompt.NewDatasetAdapter(trainset)

	// Create metric adapter for dspy-go
	metricAdapter := prompt.NewMetricAdapter(metric)
	coreMetric := metricAdapter.ToCoreMetric()

	// Run optimization in a goroutine (it can be long-running)
	go func() {
		optimizeCtx := context.Background() // Use background context for long-running operation

		defer func() {
			if r := recover(); r != nil {
				reason := fmt.Sprintf("optimization panicked: %v", r)
				_ = s.FailRun(optimizeCtx, run.ID, reason)

				// Publish failure event to subscribers
				s.publishProgress(run.ID, OptimizationProgressEvent{
					Type:      "failed",
					RunID:     run.ID,
					Status:    string(models.OptimizationStatusFailed),
					Message:   reason,
					Timestamp: time.Now().Format(time.RFC3339),
				})
			}
		}()

		// Configure GEPA with Alicia-specific settings
		gepaConfig := &optimizers.GEPAConfig{
			MaxGenerations:       s.config.MaxIterations / 10,
			PopulationSize:       20,
			MutationRate:         0.3,
			CrossoverRate:        0.7,
			ElitismRate:          0.1,
			ReflectionFreq:       2,
			ReflectionDepth:      3,
			SelfCritiqueTemp:     0.7,
			TournamentSize:       3,
			SelectionStrategy:    "adaptive_pareto",
			ConvergenceThreshold: 0.01,
			StagnationLimit:      3,
			EvaluationBatchSize:  s.config.MinibatchSize,
			ConcurrencyLevel:     3,
			Temperature:          0.8,
			MaxTokens:            500,
		}

		// Ensure at least 1 generation
		if gepaConfig.MaxGenerations < 1 {
			gepaConfig.MaxGenerations = 1
		}

		// Create the GEPA optimizer
		gepaOptimizer, err := optimizers.NewGEPA(gepaConfig)
		if err != nil {
			_ = s.FailRun(optimizeCtx, run.ID, fmt.Sprintf("failed to create GEPA optimizer: %v", err))
			return
		}

		// Run GEPA optimization with memory-aware module
		optimizedProgram, err := gepaOptimizer.Compile(optimizeCtx, program, dataset, coreMetric)
		if err != nil {
			_ = s.FailRun(optimizeCtx, run.ID, fmt.Sprintf("GEPA optimization failed: %v", err))
			return
		}

		// Get the GEPA optimization state to extract results
		gepaState := gepaOptimizer.GetOptimizationState()

		// Extract best candidate from GEPA state
		bestPrompt := ""
		bestScore := 0.0
		bestDimScores := prompt.DimensionScores{}

		if gepaState != nil && gepaState.BestCandidate != nil {
			bestPrompt = gepaState.BestCandidate.Instruction
			bestScore = gepaState.BestCandidate.Fitness

			// Extract GEPA's multi-objective fitness for the best candidate
			if moFitness, ok := gepaState.MultiObjectiveFitnessMap[gepaState.BestCandidate.ID]; ok && moFitness != nil {
				bestDimScores = prompt.DimensionScores{
					SuccessRate:    moFitness.SuccessRate,
					Quality:        moFitness.OutputQuality,
					Efficiency:     moFitness.Efficiency,
					Robustness:     moFitness.Robustness,
					Generalization: moFitness.Generalization,
					Diversity:      moFitness.Diversity,
					Innovation:     moFitness.Innovation,
				}
			} else {
				// Fallback to basic fitness if multi-objective data unavailable
				bestDimScores = prompt.DimensionScores{
					SuccessRate:    bestScore,
					Quality:        bestScore * 0.9,
					Efficiency:     0.8,
					Robustness:     bestScore * 0.85,
					Generalization: bestScore * 0.8,
					Diversity:      0.5,
					Innovation:     0.3,
				}
			}
		}

		// Store Pareto archive solutions as candidates
		paretoArchive := gepaState.GetParetoArchive()
		for i, candidate := range paretoArchive {
			savedCandidate, err := s.AddCandidate(
				optimizeCtx,
				run.ID,
				candidate.Instruction,
				candidate.Generation,
			)
			if err != nil {
				continue
			}

			// Record evaluation for this candidate
			// Extract multi-objective fitness from archive fitness map
			candidateDimScores := prompt.DimensionScores{}
			if moFitness, ok := gepaState.ArchiveFitnessMap[candidate.ID]; ok && moFitness != nil {
				candidateDimScores = prompt.DimensionScores{
					SuccessRate:    moFitness.SuccessRate,
					Quality:        moFitness.OutputQuality,
					Efficiency:     moFitness.Efficiency,
					Robustness:     moFitness.Robustness,
					Generalization: moFitness.Generalization,
					Diversity:      moFitness.Diversity,
					Innovation:     moFitness.Innovation,
				}
			} else {
				// Fallback to basic fitness
				candidateDimScores = prompt.DimensionScores{
					SuccessRate:    candidate.Fitness,
					Quality:        candidate.Fitness * 0.9,
					Efficiency:     0.8,
					Robustness:     candidate.Fitness * 0.85,
					Generalization: candidate.Fitness * 0.8,
					Diversity:      0.5,
					Innovation:     0.3,
				}
			}

			_, _ = s.RecordEvaluationWithDimensions(
				optimizeCtx,
				savedCandidate.ID,
				run.ID,
				fmt.Sprintf("pareto_%d", i),
				"",
				candidateDimScores,
				candidate.Fitness > 0.5,
				0,
			)

			// Update progress periodically
			if i%5 == 0 {
				_ = s.UpdateProgressWithDimensions(optimizeCtx, run.ID, i, candidateDimScores)

				// Publish progress event to subscribers
				s.publishProgress(run.ID, OptimizationProgressEvent{
					Type:          "progress",
					RunID:         run.ID,
					Iteration:     i,
					MaxIterations: run.MaxIterations,
					CurrentScore:  candidate.Fitness,
					BestScore:     bestScore,
					DimensionScores: map[string]float64{
						"successRate":    candidateDimScores.SuccessRate,
						"quality":        candidateDimScores.Quality,
						"efficiency":     candidateDimScores.Efficiency,
						"robustness":     candidateDimScores.Robustness,
						"generalization": candidateDimScores.Generalization,
						"diversity":      candidateDimScores.Diversity,
						"innovation":     candidateDimScores.Innovation,
					},
					Status:    string(models.OptimizationStatusRunning),
					Timestamp: time.Now().Format(time.RFC3339),
				})
			}
		}

		// Validation phase on optimized program
		if len(valset) > 0 && optimizedProgram.Forward != nil {
			valDimScores := prompt.DimensionScores{}
			successCount := 0

			for _, example := range valset {
				outputs, err := optimizedProgram.Execute(optimizeCtx, prompt.ConvertToInterfaceMap(example.Inputs))
				if err != nil {
					continue
				}

				predExample := prompt.Example{
					Inputs:  example.Inputs,
					Outputs: prompt.ConvertFromInterfaceMap(outputs),
				}

				scoreResult, err := metric.Score(optimizeCtx, example, predExample, nil)
				if err != nil {
					continue
				}

				exampleDimScores := calculateDimensionScores(scoreResult, example, predExample)
				valDimScores.SuccessRate += exampleDimScores.SuccessRate
				valDimScores.Quality += exampleDimScores.Quality
				valDimScores.Efficiency += exampleDimScores.Efficiency
				valDimScores.Robustness += exampleDimScores.Robustness
				valDimScores.Generalization += exampleDimScores.Generalization
				valDimScores.Diversity += exampleDimScores.Diversity
				valDimScores.Innovation += exampleDimScores.Innovation
				successCount++
			}

			if successCount > 0 {
				n := float64(successCount)
				valDimScores.SuccessRate /= n
				valDimScores.Quality /= n
				valDimScores.Efficiency /= n
				valDimScores.Robustness /= n
				valDimScores.Generalization /= n
				valDimScores.Diversity /= n
				valDimScores.Innovation /= n

				valScore := valDimScores.WeightedScore(s.config.DimensionWeights)

				// Use validation score as final best score if worse
				if valScore < bestScore {
					bestScore = valScore
					bestDimScores = valDimScores
				}
			}
		}

		// Mark run as completed with best dimension scores
		run, _ := s.repo.GetRun(optimizeCtx, run.ID)
		if run != nil {
			run.BestScore = bestScore
			run.BestDimScores = map[string]float64{
				"successRate":    bestDimScores.SuccessRate,
				"quality":        bestDimScores.Quality,
				"efficiency":     bestDimScores.Efficiency,
				"robustness":     bestDimScores.Robustness,
				"generalization": bestDimScores.Generalization,
				"diversity":      bestDimScores.Diversity,
				"innovation":     bestDimScores.Innovation,
			}
			run.MarkCompleted()
			_ = s.repo.UpdateRun(optimizeCtx, run)
		}

		// Suppress unused variable warning
		_ = bestPrompt
	}()

	return run, nil
}

// GetOptimizedProgram retrieves the optimized program for a completed run
func (s *OptimizationService) GetOptimizedProgram(ctx context.Context, runID string) (*ports.OptimizedProgram, error) {
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get optimization run")
	}

	if run.Status != models.OptimizationStatusCompleted {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "optimization run is not completed")
	}

	bestCandidate, err := s.repo.GetBestCandidate(ctx, runID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get best candidate")
	}

	program := &ports.OptimizedProgram{
		RunID:      runID,
		BestPrompt: bestCandidate.PromptText,
		BestScore:  run.BestScore,
		Iterations: run.Iterations,
	}

	if run.CompletedAt != nil {
		program.CompletedAt = run.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	return program, nil
}
