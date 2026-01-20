package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// PathSearchConfig configures a GEPA path search run.
// This extends the basic models.PathSearchConfig with additional fields
// for budget control and runtime configuration.
type PathSearchConfig struct {
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
}

// DefaultPathSearchConfig returns sensible defaults for path search configuration.
func DefaultPathSearchConfig() *PathSearchConfig {
	return &PathSearchConfig{
		MaxGenerations:     5,
		BranchesPerGen:     3,
		TargetScore:        0.85,
		MaxToolCalls:       100, // Budget: total tool calls across all paths
		MaxLLMCalls:        50,  // Budget: total LLM calls across all paths
		ParetoArchiveSize:  50,
		EnableCrossover:    true,
		ExecutionTimeoutMs: 30000, // 30 seconds per execution
	}
}

// MaxToolLoopIterations is the maximum number of LLM-tool loop iterations.
// This prevents infinite loops when tools keep being called.
const MaxToolLoopIterations = 5

// PathSearchController orchestrates GEPA path search for single-query solution discovery.
// Unlike prompt optimization which optimizes across many queries, path search explores
// multiple execution paths to find the best answer for ONE specific query.
type PathSearchController struct {
	// Core LLM services
	llmService    ports.LLMService
	reflectionLLM ports.LLMService // Optional stronger model for mutation/reflection

	// GEPA components
	mutator       *prompt.PathMutator
	evaluator     *baselines.PathEvaluator
	paretoArchive *prompt.PathParetoArchive

	// ID generator for creating unique candidate IDs
	idGenerator ports.IDGenerator

	// Configuration
	config *PathSearchConfig

	// Budget tracking
	toolCallsUsed int
	llmCallsUsed  int

	// Optional tool execution support
	// When toolRunner is non-nil, executePath runs a multi-turn agent loop
	// that actually executes tools and feeds results back to the LLM.
	// When nil, only single-turn execution capturing tool call intents is performed.
	toolRunner ports.ToolRunner
	tools        []*models.Tool // Available tools for LLM to call
}

// NewPathSearchController creates a new PathSearchController with tool execution support.
// Tool execution is the default behavior - the multi-turn agent loop will:
//  1. Call the LLM with available tools
//  2. Execute any requested tools via the ToolRunner
//  3. Feed results back to the LLM
//  4. Repeat until the LLM provides a final answer (no more tool calls)
//
// Parameters:
//   - llmService: The main LLM service for agent execution
//   - reflectionLLM: Optional stronger LLM for mutation/reflection (can be nil, defaults to llmService)
//   - idGenerator: ID generator for creating unique candidate IDs
//   - config: Configuration for the search (can be nil, defaults to DefaultPathSearchConfig)
//   - toolRunner: Runner for executing tools (can be nil to disable tool execution and use single-turn mode)
//   - tools: Available tools for the LLM to call (can be nil/empty)
func NewPathSearchController(
	llmService ports.LLMService,
	reflectionLLM ports.LLMService,
	idGenerator ports.IDGenerator,
	config *PathSearchConfig,
	toolRunner ports.ToolRunner,
	tools []*models.Tool,
) *PathSearchController {
	if config == nil {
		config = DefaultPathSearchConfig()
	}

	// Create mutator with reflection LLM (falls back to main LLM if nil)
	mutator := prompt.NewPathMutator(llmService, reflectionLLM)

	// Create evaluator
	evaluator := baselines.NewPathEvaluator(llmService)

	// Create Pareto archive
	paretoArchive := prompt.NewPathParetoArchive(config.ParetoArchiveSize)

	return &PathSearchController{
		llmService:    llmService,
		reflectionLLM: reflectionLLM,
		mutator:       mutator,
		evaluator:     evaluator,
		paretoArchive: paretoArchive,
		idGenerator:   idGenerator,
		config:        config,
		toolCallsUsed: 0,
		llmCallsUsed:  0,
		toolRunner:    toolRunner,
		tools:         tools,
	}
}

// NewPathSearchControllerWithTools is deprecated: use NewPathSearchController instead.
// Tool execution is now the default behavior in NewPathSearchController.
// This function is kept for backward compatibility but simply delegates to NewPathSearchController.
//
// Deprecated: Use NewPathSearchController with toolRunner and tools parameters.
func NewPathSearchControllerWithTools(
	llmService ports.LLMService,
	reflectionLLM ports.LLMService,
	idGenerator ports.IDGenerator,
	config *PathSearchConfig,
	toolRunner ports.ToolRunner,
	tools []*models.Tool,
) *PathSearchController {
	return NewPathSearchController(llmService, reflectionLLM, idGenerator, config, toolRunner, tools)
}

// SetToolRunner sets or updates the tool runner and available tools.
// This allows enabling/disabling tool execution after construction.
func (c *PathSearchController) SetToolRunner(runner ports.ToolRunner, tools []*models.Tool) {
	c.toolRunner = runner
	c.tools = tools
}

// Search implements ports.PathSearchService.Search.
// It explores execution paths to find the best answer for a single query.
// Uses GEPA's evolutionary approach:
//  1. Initialize with a seed candidate
//  2. For each generation:
//     - Select candidates from Pareto front for mutation
//     - Execute each candidate path
//     - Evaluate (get scores + feedback)
//     - Update Pareto archive
//     - Track best by answer quality
//     - Early exit if target reached
//     - Mutate strategies based on feedback
//     - Crossover between Pareto-optimal paths
//  3. Return PathSearchResult with best path found
func (c *PathSearchController) Search(ctx context.Context, query string, config *models.PathSearchConfig) (*models.PathSearchResult, error) {
	// Apply config overrides if provided
	c.applyConfig(config)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Reset budget tracking for this search
	c.toolCallsUsed = 0
	c.llmCallsUsed = 0

	// Clear Pareto archive for fresh search
	c.paretoArchive.Clear()

	// Generate a run ID for this search
	runID := c.generateRunID()

	// Initialize with seed candidate
	seed := c.createSeedCandidate(runID, query)

	// Execute the seed candidate to get initial trace and scores
	trace, err := c.executePath(ctx, query, seed)
	if err != nil {
		// Even if execution fails, continue with empty trace
		trace = &models.ExecutionTrace{
			Query:       query,
			ToolCalls:   []models.ToolCallRecord{},
			FinalAnswer: "",
			DurationMs:  0,
		}
	}
	seed.SetTrace(trace)

	// Evaluate the seed
	scores, feedback, err := c.evaluateCandidate(ctx, query, trace)
	if err != nil {
		// Use zero scores on evaluation error
		scores = models.PathScores{}
		feedback = "Evaluation failed: " + err.Error()
	}
	seed.SetScores(scores)

	// Add seed to Pareto archive
	c.paretoArchive.Add(seed)

	var bestPath *models.PathCandidate = seed

	// Main evolutionary loop
	for gen := 0; gen < c.config.MaxGenerations; gen++ {
		// Check budget limits
		if c.toolCallsUsed >= c.config.MaxToolCalls || c.llmCallsUsed >= c.config.MaxLLMCalls {
			break
		}

		// Select candidates from Pareto front for this generation
		selected := c.paretoArchive.SelectForMutation(c.config.BranchesPerGen)
		if len(selected) == 0 {
			// No candidates to select from, use best path
			if bestPath != nil {
				selected = []*models.PathCandidate{bestPath}
			} else {
				break
			}
		}

		// Process each selected candidate
		for _, candidate := range selected {
			// Check budget limits
			if c.toolCallsUsed >= c.config.MaxToolCalls || c.llmCallsUsed >= c.config.MaxLLMCalls {
				break
			}

			// Get feedback for this candidate (use last evaluation if available)
			candidateFeedback := feedback
			if candidate.Trace != nil {
				_, candidateFeedback, _ = c.evaluateCandidate(ctx, query, candidate.Trace)
			}

			// Mutate strategy based on feedback
			mutated, err := c.mutateCandidate(ctx, candidate, candidateFeedback)
			if err != nil {
				// Continue with original candidate if mutation fails
				continue
			}

			// Execute the mutated path
			mutatedTrace, err := c.executePath(ctx, query, mutated)
			if err != nil {
				continue
			}
			mutated.SetTrace(mutatedTrace)

			// Evaluate
			mutatedScores, mutatedFeedback, err := c.evaluateCandidate(ctx, query, mutatedTrace)
			if err != nil {
				continue
			}
			mutated.SetScores(mutatedScores)
			feedback = mutatedFeedback // Update feedback for next iteration

			// Update Pareto archive
			c.paretoArchive.Add(mutated)

			// Track best by answer quality
			if mutatedScores.AnswerQuality > bestPath.Scores.AnswerQuality {
				bestPath = mutated
			}

			// Early exit if target reached
			if mutatedScores.AnswerQuality >= c.config.TargetScore {
				return &models.PathSearchResult{
					BestPath:   bestPath,
					Answer:     mutatedTrace.FinalAnswer,
					Score:      mutatedScores.AnswerQuality,
					Iterations: gen + 1,
				}, nil
			}
		}

		// Crossover between Pareto-optimal paths (if enabled and archive has enough diversity)
		if c.config.EnableCrossover && c.paretoArchive.Size() >= 2 {
			parent1, parent2 := c.selectDiversePair()
			if parent1 != nil && parent2 != nil {
				child, err := c.crossoverCandidates(ctx, parent1, parent2)
				if err == nil && child != nil {
					// Execute and evaluate crossover child
					childTrace, err := c.executePath(ctx, query, child)
					if err == nil {
						child.SetTrace(childTrace)
						childScores, _, _ := c.evaluateCandidate(ctx, query, childTrace)
						child.SetScores(childScores)
						c.paretoArchive.Add(child)

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
		return nil, fmt.Errorf("no valid path found after %d generations", c.config.MaxGenerations)
	}

	return &models.PathSearchResult{
		BestPath:   bestPath,
		Answer:     bestPath.Trace.FinalAnswer,
		Score:      bestPath.Scores.AnswerQuality,
		Iterations: c.config.MaxGenerations,
	}, nil
}

// executePath runs the agent with the candidate's strategy and captures the execution trace.
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
func (c *PathSearchController) executePath(ctx context.Context, query string, candidate *models.PathCandidate) (*models.ExecutionTrace, error) {
	if candidate == nil {
		return nil, fmt.Errorf("candidate cannot be nil")
	}

	startTime := time.Now()

	// Build the agent prompt with strategy and accumulated lessons
	agentPrompt := c.buildAgentPrompt(candidate, query)

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
	if c.toolRunner == nil {
		return c.executePathSingleTurn(ctx, messages, agentPrompt, query, startTime)
	}

	// Multi-turn execution with tool loop
	return c.executePathWithToolLoop(ctx, messages, agentPrompt, query, startTime)
}

// executePathSingleTurn performs single-turn execution without actual tool execution.
// This is the original behavior - captures tool call intents but doesn't execute them.
func (c *PathSearchController) executePathSingleTurn(ctx context.Context, messages []ports.LLMMessage, agentPrompt, query string, startTime time.Time) (*models.ExecutionTrace, error) {
	c.llmCallsUsed++

	var response *ports.LLMResponse
	var err error

	// Use ChatWithTools if tools are available, otherwise use Chat
	if len(c.tools) > 0 {
		response, err = c.llmService.ChatWithTools(ctx, messages, c.tools)
	} else {
		response, err = c.llmService.Chat(ctx, messages)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	duration := time.Since(startTime)

	// Parse any tool calls from response (not executed, just captured)
	toolCalls := c.parseToolCalls(response)

	// Build execution trace
	trace := &models.ExecutionTrace{
		Query:          query,
		ToolCalls:      toolCalls,
		ReasoningSteps: []string{},
		FinalAnswer:    response.Content,
		TotalTokens:    c.estimateTokens(agentPrompt, response.Content),
		DurationMs:     duration.Milliseconds(),
	}

	// Track tool calls in budget
	c.toolCallsUsed += len(toolCalls)

	return trace, nil
}

// executePathWithToolLoop performs multi-turn execution with actual tool execution.
// Runs an agent loop: LLM -> tools -> LLM -> tools -> ... -> final answer
func (c *PathSearchController) executePathWithToolLoop(ctx context.Context, messages []ports.LLMMessage, agentPrompt, query string, startTime time.Time) (*models.ExecutionTrace, error) {
	currentMessages := make([]ports.LLMMessage, len(messages))
	copy(currentMessages, messages)

	var allToolCalls []models.ToolCallRecord
	var finalAnswer string
	var totalTokens int

	for iteration := 0; iteration < MaxToolLoopIterations; iteration++ {
		c.llmCallsUsed++

		var response *ports.LLMResponse
		var err error

		// Use ChatWithTools if tools are available
		if len(c.tools) > 0 {
			response, err = c.llmService.ChatWithTools(ctx, currentMessages, c.tools)
		} else {
			response, err = c.llmService.Chat(ctx, currentMessages)
		}

		if err != nil {
			return nil, fmt.Errorf("LLM call failed on iteration %d: %w", iteration, err)
		}

		// Estimate tokens for this turn
		totalTokens += c.estimateTokens("", response.Content)

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
			result, execErr := c.toolRunner.RunTool(ctx, tc.Name, tc.Arguments)

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
			c.toolCallsUsed++
		}

		// If this was the last iteration and we still have tool calls, use last response as answer
		if iteration == MaxToolLoopIterations-1 {
			finalAnswer = response.Content
			if finalAnswer == "" {
				finalAnswer = "Max tool execution iterations reached."
			}
		}
	}

	duration := time.Since(startTime)

	// Build execution trace with all tool calls and results
	trace := &models.ExecutionTrace{
		Query:          query,
		ToolCalls:      allToolCalls,
		ReasoningSteps: []string{},
		FinalAnswer:    finalAnswer,
		TotalTokens:    totalTokens + c.estimateTokens(agentPrompt, ""),
		DurationMs:     duration.Milliseconds(),
	}

	return trace, nil
}

// createSeedCandidate creates a generation-0 seed candidate with a default strategy.
// Uses NewSeedCandidate from models if available.
func (c *PathSearchController) createSeedCandidate(runID, query string) *models.PathCandidate {
	candidateID := c.generateCandidateID()
	return models.NewSeedCandidate(candidateID, runID)
}

// selectDiversePair selects two diverse candidates from the Pareto front for crossover.
// Picks candidates that are far apart in objective space to maximize diversity.
func (c *PathSearchController) selectDiversePair() (*models.PathCandidate, *models.PathCandidate) {
	front := c.paretoArchive.GetParetoFront()
	if len(front) < 2 {
		return nil, nil
	}

	// Find the two most diverse candidates based on Euclidean distance in score space
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

// buildAgentPrompt constructs the full prompt for agent execution.
// Combines the candidate's strategy with accumulated lessons.
func (c *PathSearchController) buildAgentPrompt(candidate *models.PathCandidate, query string) string {
	var sb strings.Builder

	// Add strategy prompt
	sb.WriteString(candidate.StrategyPrompt)
	sb.WriteString("\n\n")

	// Add accumulated lessons if any
	if len(candidate.AccumulatedLessons) > 0 {
		sb.WriteString("ACCUMULATED LESSONS FROM PREVIOUS ATTEMPTS:\n")
		for i, lesson := range candidate.AccumulatedLessons {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, lesson))
		}
		sb.WriteString("\n")
	}

	// Add query instruction
	sb.WriteString("Think step by step. Use tools as needed to find the best answer.\n")
	sb.WriteString(fmt.Sprintf("\nQuery: %s", query))

	return sb.String()
}

// parseToolCalls extracts tool call information from LLM response.
// For now, returns empty since we're simulating without actual tool execution.
func (c *PathSearchController) parseToolCalls(response *ports.LLMResponse) []models.ToolCallRecord {
	if response == nil || len(response.ToolCalls) == 0 {
		return []models.ToolCallRecord{}
	}

	toolCalls := make([]models.ToolCallRecord, len(response.ToolCalls))
	for i, tc := range response.ToolCalls {
		toolCalls[i] = models.ToolCallRecord{
			ToolName:  tc.Name,
			Arguments: tc.Arguments,
			Success:   true, // Assume success for now (no actual execution)
			Result:    nil,
			Error:     "",
		}
	}

	return toolCalls
}

// evaluateCandidate evaluates a path trace using the PathEvaluator.
// Converts between models types and prompt types as needed.
func (c *PathSearchController) evaluateCandidate(ctx context.Context, query string, trace *models.ExecutionTrace) (models.PathScores, string, error) {
	// Convert models.ExecutionTrace to prompt.ExecutionTrace for evaluation
	promptTrace := c.convertToPromptTrace(trace)

	// Evaluate using the baselines evaluator
	c.llmCallsUsed++ // Evaluation uses LLM calls
	promptScores, feedback, err := c.evaluator.Evaluate(ctx, query, promptTrace)
	if err != nil {
		return models.PathScores{}, "", fmt.Errorf("evaluation failed: %w", err)
	}

	// Convert prompt.PathScores to models.PathScores
	modelScores := models.PathScores{
		AnswerQuality: promptScores.AnswerQuality,
		Efficiency:    promptScores.Efficiency,
		TokenCost:     promptScores.TokenCost,
		Robustness:    promptScores.Robustness,
		Latency:       promptScores.Latency,
	}

	return modelScores, feedback, nil
}

// mutateCandidate creates a mutated version of the candidate using the PathMutator.
// Converts between models types and prompt types as needed.
func (c *PathSearchController) mutateCandidate(ctx context.Context, candidate *models.PathCandidate, feedback string) (*models.PathCandidate, error) {
	// Convert models.PathCandidate to prompt.PathCandidate
	promptCandidate := c.convertToPromptCandidate(candidate)

	// Convert models.ExecutionTrace to prompt.ExecutionTrace
	var promptTrace *prompt.ExecutionTrace
	if candidate.Trace != nil {
		promptTrace = c.convertToPromptTrace(candidate.Trace)
	} else {
		// Create empty trace if none exists
		promptTrace = &prompt.ExecutionTrace{
			Query:       "",
			ToolCalls:   []prompt.ToolCallRecord{},
			FinalAnswer: "",
		}
	}

	// Mutate using the prompt mutator
	c.llmCallsUsed++ // Mutation uses LLM calls
	mutatedPrompt, err := c.mutator.MutateStrategy(ctx, promptCandidate, promptTrace, feedback)
	if err != nil {
		return nil, fmt.Errorf("mutation failed: %w", err)
	}

	// Convert back to models.PathCandidate
	return c.convertFromPromptCandidate(mutatedPrompt), nil
}

// crossoverCandidates creates a child candidate by crossing over two parents.
func (c *PathSearchController) crossoverCandidates(ctx context.Context, parent1, parent2 *models.PathCandidate) (*models.PathCandidate, error) {
	// Convert to prompt types
	promptParent1 := c.convertToPromptCandidate(parent1)
	promptParent2 := c.convertToPromptCandidate(parent2)

	// Perform crossover
	c.llmCallsUsed++ // Crossover uses LLM calls
	child, err := c.mutator.Crossover(ctx, promptParent1, promptParent2)
	if err != nil {
		return nil, fmt.Errorf("crossover failed: %w", err)
	}

	// Convert back to models type
	return c.convertFromPromptCandidate(child), nil
}

// convertToPromptTrace converts models.ExecutionTrace to prompt.ExecutionTrace.
func (c *PathSearchController) convertToPromptTrace(trace *models.ExecutionTrace) *prompt.ExecutionTrace {
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

// convertToPromptCandidate converts models.PathCandidate to prompt.PathCandidate.
func (c *PathSearchController) convertToPromptCandidate(candidate *models.PathCandidate) *prompt.PathCandidate {
	if candidate == nil {
		return nil
	}

	// Convert trace if present
	var promptTrace *prompt.ExecutionTrace
	if candidate.Trace != nil {
		promptTrace = c.convertToPromptTrace(candidate.Trace)
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

// convertFromPromptCandidate converts prompt.PathCandidate to models.PathCandidate.
func (c *PathSearchController) convertFromPromptCandidate(candidate *prompt.PathCandidate) *models.PathCandidate {
	if candidate == nil {
		return nil
	}

	// Convert trace if present
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

// generateRunID generates a unique run ID for this search.
func (c *PathSearchController) generateRunID() string {
	return c.idGenerator.GenerateOptimizationRunID()
}

// generateCandidateID generates a unique candidate ID.
func (c *PathSearchController) generateCandidateID() string {
	return c.idGenerator.GeneratePromptCandidateID()
}

// estimateTokens provides a rough estimate of token usage.
// This is a simple heuristic; a real implementation would use tokenizer.
func (c *PathSearchController) estimateTokens(prompt, response string) int {
	// Rough estimate: ~4 characters per token on average
	totalChars := len(prompt) + len(response)
	return totalChars / 4
}

// applyConfig applies optional configuration overrides to the controller.
// If config is nil, the existing configuration is preserved.
func (c *PathSearchController) applyConfig(config *models.PathSearchConfig) {
	if config == nil {
		return
	}

	if config.MaxGenerations > 0 {
		c.config.MaxGenerations = config.MaxGenerations
	}
	if config.BranchesPerGen > 0 {
		c.config.BranchesPerGen = config.BranchesPerGen
	}
	if config.TargetScore > 0 && config.TargetScore <= 1.0 {
		c.config.TargetScore = config.TargetScore
	}
}

// GetBestByQuality returns the candidate with the highest answer quality score.
func (c *PathSearchController) GetBestByQuality() *models.PathCandidate {
	return c.paretoArchive.GetBestByQuality()
}

// GetBudgetUsage returns the current budget usage.
func (c *PathSearchController) GetBudgetUsage() (toolCalls, llmCalls int) {
	return c.toolCallsUsed, c.llmCallsUsed
}

// SearchWithSeed implements ports.PathSearchService.SearchWithSeed.
// Starts search from a custom seed strategy instead of the default.
// Useful when you have domain-specific knowledge about how to approach certain queries.
func (c *PathSearchController) SearchWithSeed(ctx context.Context, query string, seedStrategy string, config *models.PathSearchConfig) (*models.PathSearchResult, error) {
	// Apply config overrides if provided
	c.applyConfig(config)

	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if seedStrategy == "" {
		return nil, fmt.Errorf("seed strategy cannot be empty")
	}

	// Reset budget tracking for this search
	c.toolCallsUsed = 0
	c.llmCallsUsed = 0

	// Clear Pareto archive for fresh search
	c.paretoArchive.Clear()

	// Generate a run ID for this search
	runID := c.generateRunID()

	// Create custom seed candidate with provided strategy
	candidateID := c.generateCandidateID()
	seed := models.NewPathCandidate(candidateID, runID, 0, nil, seedStrategy, nil)

	// Execute the seed candidate to get initial trace and scores
	trace, err := c.executePath(ctx, query, seed)
	if err != nil {
		trace = &models.ExecutionTrace{
			Query:       query,
			ToolCalls:   []models.ToolCallRecord{},
			FinalAnswer: "",
			DurationMs:  0,
		}
	}
	seed.SetTrace(trace)

	// Evaluate the seed
	scores, feedback, err := c.evaluateCandidate(ctx, query, trace)
	if err != nil {
		scores = models.PathScores{}
		feedback = "Evaluation failed: " + err.Error()
	}
	seed.SetScores(scores)

	// Add seed to Pareto archive
	c.paretoArchive.Add(seed)

	// Continue with regular search loop from here
	var bestPath *models.PathCandidate = seed

	for gen := 0; gen < c.config.MaxGenerations; gen++ {
		if c.toolCallsUsed >= c.config.MaxToolCalls || c.llmCallsUsed >= c.config.MaxLLMCalls {
			break
		}

		selected := c.paretoArchive.SelectForMutation(c.config.BranchesPerGen)
		if len(selected) == 0 {
			if bestPath != nil {
				selected = []*models.PathCandidate{bestPath}
			} else {
				break
			}
		}

		for _, candidate := range selected {
			if c.toolCallsUsed >= c.config.MaxToolCalls || c.llmCallsUsed >= c.config.MaxLLMCalls {
				break
			}

			candidateFeedback := feedback
			if candidate.Trace != nil {
				_, candidateFeedback, _ = c.evaluateCandidate(ctx, query, candidate.Trace)
			}

			mutated, err := c.mutateCandidate(ctx, candidate, candidateFeedback)
			if err != nil {
				continue
			}

			mutatedTrace, err := c.executePath(ctx, query, mutated)
			if err != nil {
				continue
			}
			mutated.SetTrace(mutatedTrace)

			mutatedScores, mutatedFeedback, err := c.evaluateCandidate(ctx, query, mutatedTrace)
			if err != nil {
				continue
			}
			mutated.SetScores(mutatedScores)
			feedback = mutatedFeedback

			c.paretoArchive.Add(mutated)

			if mutatedScores.AnswerQuality > bestPath.Scores.AnswerQuality {
				bestPath = mutated
			}

			if mutatedScores.AnswerQuality >= c.config.TargetScore {
				return &models.PathSearchResult{
					BestPath:   bestPath,
					Answer:     mutatedTrace.FinalAnswer,
					Score:      mutatedScores.AnswerQuality,
					Iterations: gen + 1,
				}, nil
			}
		}

		if c.config.EnableCrossover && c.paretoArchive.Size() >= 2 {
			parent1, parent2 := c.selectDiversePair()
			if parent1 != nil && parent2 != nil {
				child, err := c.crossoverCandidates(ctx, parent1, parent2)
				if err == nil && child != nil {
					childTrace, err := c.executePath(ctx, query, child)
					if err == nil {
						child.SetTrace(childTrace)
						childScores, _, _ := c.evaluateCandidate(ctx, query, childTrace)
						child.SetScores(childScores)
						c.paretoArchive.Add(child)

						if childScores.AnswerQuality > bestPath.Scores.AnswerQuality {
							bestPath = child
						}
					}
				}
			}
		}
	}

	if bestPath == nil || bestPath.Trace == nil {
		return nil, fmt.Errorf("no valid path found after %d generations", c.config.MaxGenerations)
	}

	return &models.PathSearchResult{
		BestPath:   bestPath,
		Answer:     bestPath.Trace.FinalAnswer,
		Score:      bestPath.Scores.AnswerQuality,
		Iterations: c.config.MaxGenerations,
	}, nil
}

// Ensure PathSearchController implements the PathSearchService interface
var _ ports.PathSearchService = (*PathSearchController)(nil)

// GetParetoFront implements ports.PathSearchService.GetParetoFront.
// Returns the current Pareto-optimal candidates from the archive.
func (c *PathSearchController) GetParetoFront(ctx context.Context, runID string) ([]*models.PathCandidate, error) {
	return c.paretoArchive.GetParetoFront(), nil
}

// GetCandidate implements ports.PathSearchService.GetCandidate.
// Retrieves a candidate by ID from the current Pareto archive.
func (c *PathSearchController) GetCandidate(ctx context.Context, id string) (*models.PathCandidate, error) {
	candidate := c.paretoArchive.GetByID(id)
	if candidate == nil {
		return nil, fmt.Errorf("candidate not found: %s", id)
	}
	return candidate, nil
}

// GetCandidatesByRun implements ports.PathSearchService.GetCandidatesByRun.
// Returns all candidates from a search run.
// Note: Currently returns all candidates in the Pareto archive since we don't persist runs yet.
func (c *PathSearchController) GetCandidatesByRun(ctx context.Context, runID string) ([]*models.PathCandidate, error) {
	return c.paretoArchive.GetParetoFront(), nil
}
