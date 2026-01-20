package usecases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// SolveInput contains the input parameters for Pareto search strategy evolution.
type SolveInput struct {
	// Query is the question to find the best answer for
	Query string

	// SeedStrategy is an optional custom seed strategy (if empty, uses default)
	SeedStrategy string

	// Config contains optional configuration overrides
	Config *models.PathSearchConfig
}

// SolveOutput contains the result of Pareto search strategy evolution.
type SolveOutput struct {
	// Result contains the best path found
	Result *models.PathSearchResult

	// ParetoFront contains all non-dominated paths (optional, for analysis)
	ParetoFront []*models.PathCandidate
}

// MaxToolLoopIterations is the maximum number of LLM-tool loop iterations.
// This prevents infinite loops when tools keep being called.
const MaxToolLoopIterations = 5

// SolveWithParetoSearchStrategyEvolution is the use case for finding the best answer
// for a single query using GEPA (Genetic-Pareto) path search:
//   - Explores multiple execution paths (branching attempts)
//   - Uses Pareto selection across 5 dimensions (quality, efficiency, cost, robustness, latency)
//   - Genetically mutates strategy/reflection TEXT via LLM
//   - Accumulates lessons to guide future attempts
//
// Unlike prompt optimization which optimizes across many queries, this finds
// the best answer for ONE specific query through evolved reasoning strategies.
//
// This use case is thread-safe: each Execute call creates its own execution context
// with a local config copy and Pareto archive.
type SolveWithParetoSearchStrategyEvolution struct {
	llmService    ports.LLMService
	reflectionLLM ports.LLMService // Optional stronger model for mutation/reflection
	idGenerator   ports.IDGenerator

	// GEPA components (shared, stateless)
	mutator   *prompt.PathMutator
	evaluator *baselines.PathEvaluator

	// Default configuration (not mutated during execution)
	config *SolveConfig

	// Optional tool execution support
	// When toolRunner is non-nil, executePathWithConfig runs a multi-turn agent loop
	// that actually executes tools and feeds results back to the LLM.
	// When nil, only single-turn execution capturing tool call intents is performed.
	toolRunner ports.ToolRunner
	tools        []*models.Tool // Available tools for LLM to call
}

// executionContext holds per-execution state to ensure thread-safety.
// Each call to Execute creates its own context with local copies of mutable state.
type executionContext struct {
	config        *SolveConfig
	paretoArchive *prompt.PathParetoArchive
}

// SolveConfig configures the Pareto search strategy evolution.
type SolveConfig struct {
	// MaxGenerations is the maximum number of evolutionary generations
	MaxGenerations int

	// BranchesPerGen is the number of parallel paths to explore per generation
	BranchesPerGen int

	// TargetScore is the early exit threshold (0-1) for answer quality
	TargetScore float64

	// MaxToolCalls limits the total number of tool calls across all paths (budget)
	MaxToolCalls int

	// MaxLLMCalls limits the total number of LLM calls across all paths (budget)
	MaxLLMCalls int

	// ParetoArchiveSize is the maximum number of candidates in the Pareto archive
	ParetoArchiveSize int

	// EnableCrossover enables crossover between Pareto-optimal paths
	EnableCrossover bool

	// ExecutionTimeoutMs is the timeout for each path execution in milliseconds
	ExecutionTimeoutMs int64

	// EnableParallelBranches enables parallel processing of branches within a generation
	EnableParallelBranches bool

	// MaxParallelBranches limits the number of concurrent branch executions
	// If 0, defaults to BranchesPerGen (or 5 if BranchesPerGen is larger)
	MaxParallelBranches int
}

// DefaultSolveConfig returns sensible defaults for path search configuration.
func DefaultSolveConfig() *SolveConfig {
	return &SolveConfig{
		MaxGenerations:         5,
		BranchesPerGen:         3,
		TargetScore:            0.85,
		MaxToolCalls:           100,
		MaxLLMCalls:            50,
		ParetoArchiveSize:      50,
		EnableCrossover:        true,
		ExecutionTimeoutMs:     30000,
		EnableParallelBranches: true,
		MaxParallelBranches:    0, // Will default to min(BranchesPerGen, 5)
	}
}

// NewSolveWithParetoSearchStrategyEvolution creates a new SolveWithParetoSearchStrategyEvolution use case
// with tool execution support.
// Tool execution is the default behavior - the multi-turn agent loop will:
//  1. Call the LLM with available tools
//  2. Execute any requested tools via the ToolRunner
//  3. Feed results back to the LLM
//  4. Repeat until the LLM provides a final answer (no more tool calls)
//
// Parameters:
//   - llmService: The main LLM service for agent execution
//   - reflectionLLM: Optional stronger LLM for mutation/reflection (can be nil)
//   - idGenerator: ID generator for creating unique candidate IDs
//   - config: Configuration for the search (can be nil, defaults to DefaultSolveConfig)
//   - toolRunner: Runner for executing tools (can be nil to disable tool execution and use single-turn mode)
//   - tools: Available tools for the LLM to call (can be nil/empty)
func NewSolveWithParetoSearchStrategyEvolution(
	llmService ports.LLMService,
	reflectionLLM ports.LLMService,
	idGenerator ports.IDGenerator,
	config *SolveConfig,
	toolRunner ports.ToolRunner,
	tools []*models.Tool,
) *SolveWithParetoSearchStrategyEvolution {
	if config == nil {
		config = DefaultSolveConfig()
	}

	// Create mutator with reflection LLM (falls back to main LLM if nil)
	mutator := prompt.NewPathMutator(llmService, reflectionLLM)

	// Create evaluator
	evaluator := baselines.NewPathEvaluator(llmService)

	return &SolveWithParetoSearchStrategyEvolution{
		llmService:    llmService,
		reflectionLLM: reflectionLLM,
		idGenerator:   idGenerator,
		mutator:       mutator,
		evaluator:     evaluator,
		config:        config,
		toolRunner:    toolRunner,
		tools:         tools,
	}
}

// NewSolveWithParetoSearchStrategyEvolutionWithTools is deprecated: use NewSolveWithParetoSearchStrategyEvolution instead.
// Tool execution is now the default behavior in NewSolveWithParetoSearchStrategyEvolution.
// This function is kept for backward compatibility but simply delegates to NewSolveWithParetoSearchStrategyEvolution.
//
// Deprecated: Use NewSolveWithParetoSearchStrategyEvolution with toolRunner and tools parameters.
func NewSolveWithParetoSearchStrategyEvolutionWithTools(
	llmService ports.LLMService,
	reflectionLLM ports.LLMService,
	idGenerator ports.IDGenerator,
	config *SolveConfig,
	toolRunner ports.ToolRunner,
	tools []*models.Tool,
) *SolveWithParetoSearchStrategyEvolution {
	return NewSolveWithParetoSearchStrategyEvolution(llmService, reflectionLLM, idGenerator, config, toolRunner, tools)
}

// SetToolRunner sets or updates the tool runner and available tools.
// This allows enabling/disabling tool execution after construction.
func (uc *SolveWithParetoSearchStrategyEvolution) SetToolRunner(runner ports.ToolRunner, tools []*models.Tool) {
	uc.toolRunner = runner
	uc.tools = tools
}

// newExecutionContext creates a per-execution context with local copies of mutable state.
// This ensures thread-safety by avoiding shared state mutation during Execute calls.
func (uc *SolveWithParetoSearchStrategyEvolution) newExecutionContext(inputConfig *models.PathSearchConfig) *executionContext {
	// Create a copy of the base config
	cfg := &SolveConfig{
		MaxGenerations:         uc.config.MaxGenerations,
		BranchesPerGen:         uc.config.BranchesPerGen,
		TargetScore:            uc.config.TargetScore,
		MaxToolCalls:           uc.config.MaxToolCalls,
		MaxLLMCalls:            uc.config.MaxLLMCalls,
		ParetoArchiveSize:      uc.config.ParetoArchiveSize,
		EnableCrossover:        uc.config.EnableCrossover,
		ExecutionTimeoutMs:     uc.config.ExecutionTimeoutMs,
		EnableParallelBranches: uc.config.EnableParallelBranches,
		MaxParallelBranches:    uc.config.MaxParallelBranches,
	}

	// Apply overrides from input config if provided
	if inputConfig != nil {
		if inputConfig.MaxGenerations > 0 {
			cfg.MaxGenerations = inputConfig.MaxGenerations
		}
		if inputConfig.BranchesPerGen > 0 {
			cfg.BranchesPerGen = inputConfig.BranchesPerGen
		}
		if inputConfig.TargetScore > 0 && inputConfig.TargetScore <= 1.0 {
			cfg.TargetScore = inputConfig.TargetScore
		}
	}

	// Create a fresh Pareto archive for this execution
	paretoArchive := prompt.NewPathParetoArchive(cfg.ParetoArchiveSize)

	return &executionContext{
		config:        cfg,
		paretoArchive: paretoArchive,
	}
}

// Execute runs the path search to find the best answer for the given query.
// This is the main entry point for the use case.
// Execute is thread-safe: it creates a per-execution context with its own config copy
// and Pareto archive, ensuring concurrent calls do not interfere with each other.
func (uc *SolveWithParetoSearchStrategyEvolution) Execute(ctx context.Context, input *SolveInput) (*SolveOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Create per-execution context with local config copy and fresh Pareto archive
	execCtx := uc.newExecutionContext(input.Config)

	// Generate a run ID for this search
	runID := uc.idGenerator.GenerateOptimizationRunID()

	// Create seed candidate
	var seed *models.PathCandidate
	if input.SeedStrategy != "" {
		candidateID := uc.idGenerator.GeneratePromptCandidateID()
		seed = models.NewPathCandidate(candidateID, runID, 0, nil, input.SeedStrategy, nil)
	} else {
		candidateID := uc.idGenerator.GeneratePromptCandidateID()
		seed = models.NewSeedCandidate(candidateID, runID)
	}

	// Execute and evaluate the seed candidate
	trace, err := uc.executePathWithConfig(ctx, input.Query, seed, execCtx.config)
	if err != nil {
		trace = &models.ExecutionTrace{
			Query:       input.Query,
			ToolCalls:   []models.ToolCallRecord{},
			FinalAnswer: "",
			DurationMs:  0,
		}
	}
	seed.SetTrace(trace)

	scores, feedback, err := uc.evaluateCandidate(ctx, input.Query, trace)
	if err != nil {
		scores = models.PathScores{}
		feedback = "Evaluation failed: " + err.Error()
	}
	seed.SetScores(scores)
	seed.SetFeedback(feedback)

	// Add seed to Pareto archive
	execCtx.paretoArchive.Add(seed)

	// Track budget usage
	toolCallsUsed := len(trace.ToolCalls)
	llmCallsUsed := 1 // Initial execution

	var bestPath *models.PathCandidate = seed

	// Main evolutionary loop
	for gen := 0; gen < execCtx.config.MaxGenerations; gen++ {
		// Check budget limits
		if toolCallsUsed >= execCtx.config.MaxToolCalls || llmCallsUsed >= execCtx.config.MaxLLMCalls {
			break
		}

		// Select candidates from Pareto front for this generation
		selected := execCtx.paretoArchive.SelectForMutation(execCtx.config.BranchesPerGen)
		if len(selected) == 0 {
			if bestPath != nil {
				selected = []*models.PathCandidate{bestPath}
			} else {
				break
			}
		}

		// Process each selected candidate (parallel or sequential)
		if execCtx.config.EnableParallelBranches && len(selected) > 1 {
			// Parallel branch execution
			result := uc.processBranchesParallel(ctx, input.Query, selected, feedback, toolCallsUsed, llmCallsUsed, bestPath, execCtx)

			// Update shared state from parallel results
			toolCallsUsed = result.toolCallsUsed
			llmCallsUsed = result.llmCallsUsed
			bestPath = result.bestPath

			// Check for early exit
			if result.earlyExit {
				return &SolveOutput{
					Result: &models.PathSearchResult{
						BestPath:   result.bestPath,
						Answer:     result.bestPath.Trace.FinalAnswer,
						Score:      result.bestPath.Scores.AnswerQuality,
						Iterations: gen + 1,
					},
					ParetoFront: execCtx.paretoArchive.GetParetoFront(),
				}, nil
			}
		} else {
			// Sequential branch execution (original behavior)
			for _, candidate := range selected {
				if toolCallsUsed >= execCtx.config.MaxToolCalls || llmCallsUsed >= execCtx.config.MaxLLMCalls {
					break
				}

				// Get cached feedback for this candidate (avoids redundant LLM re-evaluation)
				candidateFeedback := candidate.Feedback
				if candidateFeedback == "" {
					// Fallback: use the most recent feedback if no cached feedback exists
					candidateFeedback = feedback
				}

				// Mutate strategy based on feedback
				mutated, err := uc.mutateCandidate(ctx, candidate, candidateFeedback)
				llmCallsUsed++
				if err != nil {
					continue
				}

				// Execute the mutated path
				mutatedTrace, err := uc.executePathWithConfig(ctx, input.Query, mutated, execCtx.config)
				llmCallsUsed++
				if err != nil {
					continue
				}
				mutated.SetTrace(mutatedTrace)
				toolCallsUsed += len(mutatedTrace.ToolCalls)

				// Evaluate
				mutatedScores, mutatedFeedback, err := uc.evaluateCandidate(ctx, input.Query, mutatedTrace)
				llmCallsUsed++
				if err != nil {
					continue
				}
				mutated.SetScores(mutatedScores)
				mutated.SetFeedback(mutatedFeedback)

				// Update Pareto archive
				execCtx.paretoArchive.Add(mutated)

				// Track best by answer quality
				if mutatedScores.AnswerQuality > bestPath.Scores.AnswerQuality {
					bestPath = mutated
				}

				// Early exit if target reached
				if mutatedScores.AnswerQuality >= execCtx.config.TargetScore {
					return &SolveOutput{
						Result: &models.PathSearchResult{
							BestPath:   bestPath,
							Answer:     mutatedTrace.FinalAnswer,
							Score:      mutatedScores.AnswerQuality,
							Iterations: gen + 1,
						},
						ParetoFront: execCtx.paretoArchive.GetParetoFront(),
					}, nil
				}
			}
		}

		// Crossover between Pareto-optimal paths
		if execCtx.config.EnableCrossover && execCtx.paretoArchive.Size() >= 2 {
			parent1, parent2 := uc.selectDiversePairFromArchive(execCtx.paretoArchive)
			if parent1 != nil && parent2 != nil {
				child, err := uc.crossoverCandidates(ctx, parent1, parent2)
				llmCallsUsed++
				if err == nil && child != nil {
					childTrace, err := uc.executePathWithConfig(ctx, input.Query, child, execCtx.config)
					llmCallsUsed++
					if err == nil {
						child.SetTrace(childTrace)
						toolCallsUsed += len(childTrace.ToolCalls)
						childScores, childFeedback, _ := uc.evaluateCandidate(ctx, input.Query, childTrace)
						llmCallsUsed++
						child.SetScores(childScores)
						child.SetFeedback(childFeedback)
						execCtx.paretoArchive.Add(child)

						if childScores.AnswerQuality > bestPath.Scores.AnswerQuality {
							bestPath = child
						}
					}
				}
			}
		}
	}

	// Return best result found
	if bestPath == nil || bestPath.Trace == nil {
		return nil, fmt.Errorf("no valid path found after %d generations", execCtx.config.MaxGenerations)
	}

	return &SolveOutput{
		Result: &models.PathSearchResult{
			BestPath:   bestPath,
			Answer:     bestPath.Trace.FinalAnswer,
			Score:      bestPath.Scores.AnswerQuality,
			Iterations: execCtx.config.MaxGenerations,
		},
		ParetoFront: execCtx.paretoArchive.GetParetoFront(),
	}, nil
}

// parallelBranchResult holds the aggregated result from parallel branch execution.
type parallelBranchResult struct {
	toolCallsUsed int
	llmCallsUsed  int
	bestPath      *models.PathCandidate
	earlyExit     bool
}

// solverBranchResult holds the result of processing a single branch in the solver.
type solverBranchResult struct {
	mutated   *models.PathCandidate
	toolCalls int
	llmCalls  int
	success   bool
	targetMet bool
}

// processBranchesParallel processes multiple branches concurrently with proper synchronization.
// It respects MaxParallelBranches to limit concurrency and protects shared state with mutexes.
func (uc *SolveWithParetoSearchStrategyEvolution) processBranchesParallel(
	ctx context.Context,
	query string,
	candidates []*models.PathCandidate,
	fallbackFeedback string,
	initialToolCalls int,
	initialLLMCalls int,
	currentBest *models.PathCandidate,
	execCtx *executionContext,
) parallelBranchResult {
	result := parallelBranchResult{
		toolCallsUsed: initialToolCalls,
		llmCallsUsed:  initialLLMCalls,
		bestPath:      currentBest,
		earlyExit:     false,
	}

	// Determine parallelism limit
	maxParallel := execCtx.config.MaxParallelBranches
	if maxParallel <= 0 {
		// Default to min(BranchesPerGen, 5)
		maxParallel = execCtx.config.BranchesPerGen
		if maxParallel > 5 {
			maxParallel = 5
		}
	}

	// Semaphore channel to limit concurrent goroutines
	sem := make(chan struct{}, maxParallel)

	// Channel to collect results from goroutines
	resultsChan := make(chan solverBranchResult, len(candidates))

	// Mutex for protecting shared state during budget checks
	var mu sync.Mutex

	// WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Track if we should stop early (target reached or context canceled)
	var stopEarly bool

	for _, candidate := range candidates {
		// Check if we should stop before launching more goroutines
		mu.Lock()
		if stopEarly || result.toolCallsUsed >= execCtx.config.MaxToolCalls || result.llmCallsUsed >= execCtx.config.MaxLLMCalls {
			mu.Unlock()
			break
		}
		mu.Unlock()

		// Check context cancellation
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)

		// Capture candidate for goroutine
		candidate := candidate

		go func() {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			// Check if we should stop early (another goroutine may have set this)
			mu.Lock()
			if stopEarly {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Get feedback for this candidate
			candidateFeedback := candidate.Feedback
			if candidateFeedback == "" {
				candidateFeedback = fallbackFeedback
			}

			brResult := solverBranchResult{
				llmCalls: 0,
				success:  false,
			}

			// Mutate strategy based on feedback
			mutated, err := uc.mutateCandidate(ctx, candidate, candidateFeedback)
			brResult.llmCalls++
			if err != nil {
				resultsChan <- brResult
				return
			}

			// Execute the mutated path
			mutatedTrace, err := uc.executePathWithConfig(ctx, query, mutated, execCtx.config)
			brResult.llmCalls++
			if err != nil {
				resultsChan <- brResult
				return
			}
			mutated.SetTrace(mutatedTrace)
			brResult.toolCalls = len(mutatedTrace.ToolCalls)

			// Evaluate
			mutatedScores, mutatedFeedback, err := uc.evaluateCandidate(ctx, query, mutatedTrace)
			brResult.llmCalls++
			if err != nil {
				resultsChan <- brResult
				return
			}
			mutated.SetScores(mutatedScores)
			mutated.SetFeedback(mutatedFeedback)

			brResult.mutated = mutated
			brResult.success = true
			brResult.targetMet = mutatedScores.AnswerQuality >= execCtx.config.TargetScore

			resultsChan <- brResult
		}()
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results and update shared state
	for brResult := range resultsChan {
		mu.Lock()

		// Update budget counters
		result.llmCallsUsed += brResult.llmCalls
		result.toolCallsUsed += brResult.toolCalls

		if brResult.success && brResult.mutated != nil {
			// Update Pareto archive (thread-safe via mutex)
			execCtx.paretoArchive.Add(brResult.mutated)

			// Track best by answer quality
			if brResult.mutated.Scores.AnswerQuality > result.bestPath.Scores.AnswerQuality {
				result.bestPath = brResult.mutated
			}

			// Check for early exit
			if brResult.targetMet {
				result.earlyExit = true
				stopEarly = true
			}
		}

		mu.Unlock()
	}

	return result
}

// executePathWithConfig runs the agent with the candidate's strategy and captures the execution trace.
// It uses the provided config for timeout settings, ensuring thread-safety.
//
// When toolRunner is nil (default): Performs single-turn execution that captures the LLM's
// tool call intents but does NOT actually execute any tools. The trace records what tools
// the LLM wanted to call (name, arguments) but Result will be nil and Success is optimistically
// set to true.
//
// When toolRunner is set: Performs a multi-turn agent loop that:
//  1. Calls the LLM (with tools if available)
//  2. Executes any requested tools via the ToolExecutor
//  3. Feeds results back to the LLM
//  4. Repeats until the LLM provides a final answer (no more tool calls) or max iterations
//
// This allows the path search to evaluate either the LLM's reasoning strategy (single-turn)
// or the full execution including tool results (multi-turn).
func (uc *SolveWithParetoSearchStrategyEvolution) executePathWithConfig(ctx context.Context, query string, candidate *models.PathCandidate, cfg *SolveConfig) (*models.ExecutionTrace, error) {
	if candidate == nil {
		return nil, fmt.Errorf("candidate cannot be nil")
	}

	// Create a timeout context for this path execution
	timeout := time.Duration(cfg.ExecutionTimeoutMs) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Capture start time to measure execution duration
	startTime := time.Now()

	// Build the agent prompt with strategy and accumulated lessons
	agentPrompt := uc.buildAgentPrompt(candidate, query)

	// Create initial LLM messages
	messages := []ports.LLMMessage{
		{
			Role:    "system",
			Content: agentPrompt,
		},
		{
			Role:    "user",
			Content: query,
		},
	}

	// If no tool executor, fall back to single-turn execution
	if uc.toolRunner == nil {
		return uc.executePathSingleTurn(timeoutCtx, messages, agentPrompt, query, startTime, cfg)
	}

	// Multi-turn execution with tool loop
	return uc.executePathWithToolLoop(timeoutCtx, messages, agentPrompt, query, startTime, cfg)
}

// executePathSingleTurn performs single-turn execution without actual tool execution.
// This is the original behavior - captures tool call intents but doesn't execute them.
func (uc *SolveWithParetoSearchStrategyEvolution) executePathSingleTurn(ctx context.Context, messages []ports.LLMMessage, agentPrompt, query string, startTime time.Time, cfg *SolveConfig) (*models.ExecutionTrace, error) {
	var response *ports.LLMResponse
	var err error

	// Use ChatWithTools if tools are available, otherwise use Chat
	if len(uc.tools) > 0 {
		response, err = uc.llmService.ChatWithTools(ctx, messages, uc.tools)
	} else {
		response, err = uc.llmService.Chat(ctx, messages)
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("path execution timed out after %dms: %w", cfg.ExecutionTimeoutMs, err)
		}
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("path execution was canceled: %w", err)
		}
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	durationMs := time.Since(startTime).Milliseconds()

	// Parse any tool calls from response (not executed, just captured)
	toolCalls := uc.parseToolCalls(response)

	// Build execution trace
	trace := &models.ExecutionTrace{
		Query:          query,
		ToolCalls:      toolCalls,
		ReasoningSteps: []string{},
		FinalAnswer:    response.Content,
		TotalTokens:    uc.estimateTokens(agentPrompt, response.Content),
		DurationMs:     durationMs,
	}

	return trace, nil
}

// executePathWithToolLoop performs multi-turn execution with actual tool execution.
// Runs an agent loop: LLM -> tools -> LLM -> tools -> ... -> final answer
func (uc *SolveWithParetoSearchStrategyEvolution) executePathWithToolLoop(ctx context.Context, messages []ports.LLMMessage, agentPrompt, query string, startTime time.Time, cfg *SolveConfig) (*models.ExecutionTrace, error) {
	currentMessages := make([]ports.LLMMessage, len(messages))
	copy(currentMessages, messages)

	var allToolCalls []models.ToolCallRecord
	var finalAnswer string
	var totalTokens int

	for iteration := 0; iteration < MaxToolLoopIterations; iteration++ {
		var response *ports.LLMResponse
		var err error

		// Use ChatWithTools if tools are available
		if len(uc.tools) > 0 {
			response, err = uc.llmService.ChatWithTools(ctx, currentMessages, uc.tools)
		} else {
			response, err = uc.llmService.Chat(ctx, currentMessages)
		}

		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("path execution timed out after %dms on iteration %d: %w", cfg.ExecutionTimeoutMs, iteration, err)
			}
			if ctx.Err() == context.Canceled {
				return nil, fmt.Errorf("path execution was canceled on iteration %d: %w", iteration, err)
			}
			return nil, fmt.Errorf("LLM call failed on iteration %d: %w", iteration, err)
		}

		// Estimate tokens for this turn
		totalTokens += uc.estimateTokens("", response.Content)

		// Add assistant response to message history
		currentMessages = append(currentMessages, ports.LLMMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		// If no tool calls, we're done - this is the final answer
		if len(response.ToolCalls) == 0 {
			finalAnswer = response.Content
			break
		}

		// Execute each tool call and collect results
		for _, tc := range response.ToolCalls {
			toolRecord := models.ToolCallRecord{
				ToolName:  tc.Name,
				Arguments: tc.Arguments,
				Success:   false,
				Result:    nil,
				Error:     "",
			}

			// Execute the tool
			result, execErr := uc.toolRunner.RunTool(ctx, tc.Name, tc.Arguments)

			if execErr != nil {
				toolRecord.Success = false
				toolRecord.Error = execErr.Error()
				// Add error message to conversation
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing %s: %s", tc.Name, execErr.Error()),
				})
			} else {
				toolRecord.Success = true
				toolRecord.Result = result
				// Add tool result to conversation
				resultContent := fmt.Sprintf("%v", result)
				currentMessages = append(currentMessages, ports.LLMMessage{
					Role:    "tool",
					Content: resultContent,
				})
			}

			allToolCalls = append(allToolCalls, toolRecord)
		}

		// If this was the last iteration and we still have tool calls, use last response as answer
		if iteration == MaxToolLoopIterations-1 {
			finalAnswer = response.Content
			if finalAnswer == "" {
				finalAnswer = "Max tool execution iterations reached."
			}
		}
	}

	durationMs := time.Since(startTime).Milliseconds()

	// Build execution trace with all tool calls and results
	trace := &models.ExecutionTrace{
		Query:          query,
		ToolCalls:      allToolCalls,
		ReasoningSteps: []string{},
		FinalAnswer:    finalAnswer,
		TotalTokens:    totalTokens + uc.estimateTokens(agentPrompt, ""),
		DurationMs:     durationMs,
	}

	return trace, nil
}

// buildAgentPrompt constructs the full prompt for agent execution.
func (uc *SolveWithParetoSearchStrategyEvolution) buildAgentPrompt(candidate *models.PathCandidate, query string) string {
	var result string

	result = candidate.StrategyPrompt + "\n\n"

	if len(candidate.AccumulatedLessons) > 0 {
		result += "ACCUMULATED LESSONS FROM PREVIOUS ATTEMPTS:\n"
		for i, lesson := range candidate.AccumulatedLessons {
			result += fmt.Sprintf("%d. %s\n", i+1, lesson)
		}
		result += "\n"
	}

	result += "Think step by step. Use tools as needed to find the best answer.\n"
	result += fmt.Sprintf("\nQuery: %s", query)

	return result
}

// parseToolCalls extracts tool call information from LLM response.
//
// NOTE: This method only extracts the LLM's intent to call tools, not actual execution results.
// It parses the tool calls that the LLM requested (tool name and arguments) from the response,
// but does not execute them. The returned ToolCallRecord entries will have:
//   - ToolName and Arguments populated from the LLM's request
//   - Success optimistically set to true (no actual execution occurred)
//   - Result set to nil (no tool was actually invoked)
//   - Error left empty
//
// This is intentional for the current path search implementation, which evaluates the LLM's
// reasoning strategy without actually running tools. See executePath documentation for details.
func (uc *SolveWithParetoSearchStrategyEvolution) parseToolCalls(response *ports.LLMResponse) []models.ToolCallRecord {
	if response == nil || len(response.ToolCalls) == 0 {
		return []models.ToolCallRecord{}
	}

	toolCalls := make([]models.ToolCallRecord, len(response.ToolCalls))
	for i, tc := range response.ToolCalls {
		toolCalls[i] = models.ToolCallRecord{
			ToolName:  tc.Name,
			Arguments: tc.Arguments,
			Success:   true,
			Result:    nil,
			Error:     "",
		}
	}

	return toolCalls
}

// evaluateCandidate evaluates a path trace using the PathEvaluator.
func (uc *SolveWithParetoSearchStrategyEvolution) evaluateCandidate(ctx context.Context, query string, trace *models.ExecutionTrace) (models.PathScores, string, error) {
	promptTrace := uc.convertToPromptTrace(trace)

	promptScores, feedback, err := uc.evaluator.Evaluate(ctx, query, promptTrace)
	if err != nil {
		return models.PathScores{}, "", fmt.Errorf("evaluation failed: %w", err)
	}

	modelScores := models.PathScores{
		AnswerQuality: promptScores.AnswerQuality,
		Efficiency:    promptScores.Efficiency,
		TokenCost:     promptScores.TokenCost,
		Robustness:    promptScores.Robustness,
		Latency:       promptScores.Latency,
	}

	return modelScores, feedback, nil
}

// mutateCandidate creates a mutated version of the candidate.
func (uc *SolveWithParetoSearchStrategyEvolution) mutateCandidate(ctx context.Context, candidate *models.PathCandidate, feedback string) (*models.PathCandidate, error) {
	promptCandidate := uc.convertToPromptCandidate(candidate)

	var promptTrace *prompt.ExecutionTrace
	if candidate.Trace != nil {
		promptTrace = uc.convertToPromptTrace(candidate.Trace)
	} else {
		promptTrace = &prompt.ExecutionTrace{
			Query:       "",
			ToolCalls:   []prompt.ToolCallRecord{},
			FinalAnswer: "",
		}
	}

	mutatedPrompt, err := uc.mutator.MutateStrategy(ctx, promptCandidate, promptTrace, feedback)
	if err != nil {
		return nil, fmt.Errorf("mutation failed: %w", err)
	}

	return uc.convertFromPromptCandidate(mutatedPrompt), nil
}

// crossoverCandidates creates a child candidate by crossing over two parents.
func (uc *SolveWithParetoSearchStrategyEvolution) crossoverCandidates(ctx context.Context, parent1, parent2 *models.PathCandidate) (*models.PathCandidate, error) {
	promptParent1 := uc.convertToPromptCandidate(parent1)
	promptParent2 := uc.convertToPromptCandidate(parent2)

	child, err := uc.mutator.Crossover(ctx, promptParent1, promptParent2)
	if err != nil {
		return nil, fmt.Errorf("crossover failed: %w", err)
	}

	return uc.convertFromPromptCandidate(child), nil
}

// selectDiversePairFromArchive selects two diverse candidates from the Pareto front for crossover.
// It takes the archive as a parameter to ensure thread-safety.
func (uc *SolveWithParetoSearchStrategyEvolution) selectDiversePairFromArchive(archive *prompt.PathParetoArchive) (*models.PathCandidate, *models.PathCandidate) {
	front := archive.GetParetoFront()
	if len(front) < 2 {
		return nil, nil
	}

	var best1, best2 *models.PathCandidate
	maxDistance := 0.0

	for i := 0; i < len(front); i++ {
		for j := i + 1; j < len(front); j++ {
			distance := front[i].Scores.EuclideanDistance(front[j].Scores)
			if distance > maxDistance {
				maxDistance = distance
				best1 = front[i]
				best2 = front[j]
			}
		}
	}

	return best1, best2
}

// estimateTokens provides a rough estimate of token usage.
func (uc *SolveWithParetoSearchStrategyEvolution) estimateTokens(prompt, response string) int {
	totalChars := len(prompt) + len(response)
	return totalChars / 4
}

// --- Type conversion helpers ---

func (uc *SolveWithParetoSearchStrategyEvolution) convertToPromptTrace(trace *models.ExecutionTrace) *prompt.ExecutionTrace {
	if trace == nil {
		return nil
	}

	toolCalls := make([]prompt.ToolCallRecord, len(trace.ToolCalls))
	for i, tc := range trace.ToolCalls {
		toolCalls[i] = prompt.ToolCallRecord{
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			Result:    tc.Result,
			Success:   tc.Success,
			Error:     tc.Error,
		}
	}

	return &prompt.ExecutionTrace{
		Query:          trace.Query,
		ToolCalls:      toolCalls,
		ReasoningSteps: trace.ReasoningSteps,
		FinalAnswer:    trace.FinalAnswer,
		TotalTokens:    trace.TotalTokens,
		DurationMs:     trace.DurationMs,
	}
}

func (uc *SolveWithParetoSearchStrategyEvolution) convertToPromptCandidate(candidate *models.PathCandidate) *prompt.PathCandidate {
	if candidate == nil {
		return nil
	}

	var promptTrace *prompt.ExecutionTrace
	if candidate.Trace != nil {
		promptTrace = uc.convertToPromptTrace(candidate.Trace)
	}

	return &prompt.PathCandidate{
		ID:                 candidate.ID,
		RunID:              candidate.RunID,
		Generation:         candidate.Generation,
		ParentIDs:          candidate.ParentIDs,
		StrategyPrompt:     candidate.StrategyPrompt,
		AccumulatedLessons: candidate.AccumulatedLessons,
		Trace:              promptTrace,
		Scores: prompt.PathScores{
			AnswerQuality: candidate.Scores.AnswerQuality,
			Efficiency:    candidate.Scores.Efficiency,
			TokenCost:     candidate.Scores.TokenCost,
			Robustness:    candidate.Scores.Robustness,
			Latency:       candidate.Scores.Latency,
		},
		CreatedAt: candidate.CreatedAt,
	}
}

func (uc *SolveWithParetoSearchStrategyEvolution) convertFromPromptCandidate(candidate *prompt.PathCandidate) *models.PathCandidate {
	if candidate == nil {
		return nil
	}

	var modelTrace *models.ExecutionTrace
	if candidate.Trace != nil {
		toolCalls := make([]models.ToolCallRecord, len(candidate.Trace.ToolCalls))
		for i, tc := range candidate.Trace.ToolCalls {
			toolCalls[i] = models.ToolCallRecord{
				ToolName:  tc.ToolName,
				Arguments: tc.Arguments,
				Result:    tc.Result,
				Success:   tc.Success,
				Error:     tc.Error,
			}
		}
		modelTrace = &models.ExecutionTrace{
			Query:          candidate.Trace.Query,
			ToolCalls:      toolCalls,
			ReasoningSteps: candidate.Trace.ReasoningSteps,
			FinalAnswer:    candidate.Trace.FinalAnswer,
			TotalTokens:    candidate.Trace.TotalTokens,
			DurationMs:     candidate.Trace.DurationMs,
		}
	}

	return &models.PathCandidate{
		ID:                 candidate.ID,
		RunID:              candidate.RunID,
		Generation:         candidate.Generation,
		ParentIDs:          candidate.ParentIDs,
		StrategyPrompt:     candidate.StrategyPrompt,
		AccumulatedLessons: candidate.AccumulatedLessons,
		Trace:              modelTrace,
		Scores: models.PathScores{
			AnswerQuality: candidate.Scores.AnswerQuality,
			Efficiency:    candidate.Scores.Efficiency,
			TokenCost:     candidate.Scores.TokenCost,
			Robustness:    candidate.Scores.Robustness,
			Latency:       candidate.Scores.Latency,
		},
		CreatedAt: candidate.CreatedAt,
	}
}
