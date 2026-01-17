package ports

import (
	"context"

	"github.com/longregen/alicia/internal/domain/models"
)

// PathSearchService orchestrates GEPA path search for single-query solution discovery.
// Unlike prompt optimization which optimizes across many queries, path search explores
// multiple execution paths to find the best answer for ONE specific query.
//
// The core GEPA mechanisms apply:
//   - Pareto selection maintains diverse high-performing paths (quality vs efficiency vs cost)
//   - Genetic mutation evolves strategy/reflection TEXT via LLM (not numerical params)
//   - Lessons accumulate across attempts to guide future paths
type PathSearchService interface {
	// Search explores execution paths to find the best answer for a single query.
	// It uses GEPA's evolutionary approach: generate paths, evaluate, select Pareto-optimal,
	// mutate strategies based on feedback, and repeat until target score or max generations.
	//
	// Returns the best path found along with its answer and score.
	Search(ctx context.Context, query string, config *models.PathSearchConfig) (*models.PathSearchResult, error)

	// SearchWithSeed starts search from a custom seed strategy instead of the default.
	// Useful when you have domain-specific knowledge about how to approach certain queries.
	SearchWithSeed(ctx context.Context, query string, seedStrategy string, config *models.PathSearchConfig) (*models.PathSearchResult, error)

	// GetParetoFront returns the current Pareto-optimal candidates from the archive.
	// Useful for inspecting the diversity of solutions found.
	GetParetoFront(ctx context.Context, runID string) ([]*models.PathCandidate, error)

	// GetCandidate retrieves a specific path candidate by ID.
	GetCandidate(ctx context.Context, id string) (*models.PathCandidate, error)

	// GetCandidatesByRun retrieves all candidates from a search run.
	GetCandidatesByRun(ctx context.Context, runID string) ([]*models.PathCandidate, error)
}

// AgentExecutor executes a query with a given strategy and returns the execution trace.
// This is the interface between GEPA path search and the actual agent execution engine.
//
// Implementations should:
//   - Execute the query using the provided strategy prompt
//   - Track all tool calls and their results
//   - Capture reasoning steps if available
//   - Record timing and token usage
type AgentExecutor interface {
	// Execute runs the agent with the given strategy and query, returning the execution trace.
	// The strategy prompt guides how the agent approaches the query.
	// The accumulated lessons provide context from previous attempts.
	Execute(ctx context.Context, query string, strategyPrompt string, accumulatedLessons []string) (*models.ExecutionTrace, error)

	// ExecuteWithTimeout is like Execute but with an explicit timeout.
	// Returns an error if execution exceeds the timeout.
	ExecuteWithTimeout(ctx context.Context, query string, strategyPrompt string, accumulatedLessons []string, timeoutMs int64) (*models.ExecutionTrace, error)
}

// PathEvaluator evaluates execution paths across multiple dimensions.
// Evaluation is multi-objective for Pareto selection, scoring:
//   - AnswerQuality: correctness, completeness, no hallucinations
//   - Efficiency: fewer tool calls = better
//   - TokenCost: lower token usage = better
//   - Robustness: error handling, self-correction ability
//   - Latency: time to answer (inverted: fast = high)
//
// Evaluation uses a tiered approach:
//  1. Fast heuristic screening for obvious failures
//  2. LLM evaluation for promising candidates (quality, hallucination, specificity)
type PathEvaluator interface {
	// Evaluate scores a path across all 5 Pareto dimensions.
	// Returns scores and rich feedback for strategy mutation.
	//
	// The feedback string contains actionable observations about:
	//   - What worked and what didn't
	//   - Specific failures or inefficiencies
	//   - Suggestions for improvement
	Evaluate(ctx context.Context, query string, trace *models.ExecutionTrace) (models.PathScores, string, error)

	// EvaluateAnswerQuality performs LLM-based answer quality evaluation.
	// Used for high-stakes evaluation when heuristics are insufficient.
	EvaluateAnswerQuality(ctx context.Context, query string, answer string) (float64, error)

	// CheckHallucinations verifies that answer claims are supported by tool outputs.
	// Returns true if hallucinations are detected.
	CheckHallucinations(ctx context.Context, trace *models.ExecutionTrace) (bool, error)

	// EvaluateRobustness scores error handling and self-correction.
	// Considers: error count, recovery attempts, severity of failures.
	EvaluateRobustness(ctx context.Context, trace *models.ExecutionTrace) (float64, error)
}

// PathMutator evolves path strategies using LLM-based reflection.
// The "gene" being mutated is the strategy prompt and accumulated lessons (TEXT),
// not numerical parameters. Mutation happens via LLM reflection on execution traces.
//
// Key operations:
//   - MutateStrategy: Reflect on a trace and generate an improved strategy
//   - Crossover: Merge strategies from two Pareto-optimal paths
type PathMutator interface {
	// MutateStrategy uses LLM to evolve a strategy based on execution trace and feedback.
	// Analyzes what worked and what didn't, then generates an improved strategy.
	//
	// Returns a new PathCandidate with:
	//   - Incremented generation
	//   - Parent ID pointing to the input candidate
	//   - New strategy prompt addressing observed issues
	//   - Accumulated lessons including new learnings
	MutateStrategy(ctx context.Context, candidate *models.PathCandidate, trace *models.ExecutionTrace, feedback string) (*models.PathCandidate, error)

	// Crossover merges strategies from two Pareto-optimal paths.
	// Combines the strengths of both parents (e.g., one optimized for quality,
	// another for efficiency) into a balanced child strategy.
	//
	// Returns a new PathCandidate with:
	//   - Generation = max(parent1, parent2) + 1
	//   - Both parent IDs recorded
	//   - Merged strategy combining best elements
	//   - Combined lessons from both parents (deduplicated)
	Crossover(ctx context.Context, parent1, parent2 *models.PathCandidate) (*models.PathCandidate, error)

	// GenerateLessons extracts lessons learned from an execution trace.
	// Used to build the accumulated lessons that guide future attempts.
	GenerateLessons(ctx context.Context, query string, trace *models.ExecutionTrace, feedback string) ([]string, error)
}

// PathCandidateRepository handles persistence of path search candidates.
// Supports both in-memory operation and durable storage for long-running searches.
type PathCandidateRepository interface {
	// Create stores a new path candidate.
	Create(ctx context.Context, candidate *models.PathCandidate) error

	// GetByID retrieves a candidate by its ID.
	GetByID(ctx context.Context, id string) (*models.PathCandidate, error)

	// GetByRunID retrieves all candidates for a search run.
	GetByRunID(ctx context.Context, runID string) ([]*models.PathCandidate, error)

	// GetByGeneration retrieves candidates from a specific generation in a run.
	GetByGeneration(ctx context.Context, runID string, generation int) ([]*models.PathCandidate, error)

	// Update updates an existing candidate (e.g., after evaluation).
	Update(ctx context.Context, candidate *models.PathCandidate) error

	// GetParetoFront retrieves candidates on the Pareto front for a run.
	// These are candidates not dominated by any other candidate.
	GetParetoFront(ctx context.Context, runID string) ([]*models.PathCandidate, error)

	// Delete removes a candidate (used for pruning).
	Delete(ctx context.Context, id string) error
}

// PathParetoArchive maintains the Pareto-optimal set of path candidates.
// Implements NSGA-II style archive management with crowding distance for diversity.
type PathParetoArchive interface {
	// Add attempts to add a candidate to the archive.
	// If the candidate is dominated, it is rejected.
	// If the candidate dominates existing members, they are removed.
	// Returns true if the candidate was added.
	Add(candidate *models.PathCandidate) bool

	// GetFront returns all non-dominated candidates in the archive.
	GetFront() []*models.PathCandidate

	// SelectForMutation selects n candidates for mutation.
	// Uses crowding distance to maintain diversity.
	SelectForMutation(n int) []*models.PathCandidate

	// SelectDiversePair selects two diverse candidates for crossover.
	// Picks candidates that are far apart in objective space.
	SelectDiversePair() (*models.PathCandidate, *models.PathCandidate)

	// Size returns the current number of candidates in the archive.
	Size() int

	// Clear removes all candidates from the archive.
	Clear()
}
