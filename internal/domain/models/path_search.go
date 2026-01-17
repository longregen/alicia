package models

import (
	"math"
	"time"
)

// PathCandidate represents an execution path being explored in GEPA Path Search.
// The "gene" is the strategy text (StrategyPrompt + AccumulatedLessons), NOT numerical parameters.
// Unlike prompt optimization which optimizes across many queries, path search finds
// the best answer for ONE specific query through evolved reasoning strategies.
type PathCandidate struct {
	ID         string   `json:"id"`
	RunID      string   `json:"run_id"`
	Generation int      `json:"generation"`
	ParentIDs  []string `json:"parent_ids,omitempty"`

	// THE GENE: Strategy/reflection text that guides execution
	StrategyPrompt     string   `json:"strategy_prompt"`     // How to approach this query
	AccumulatedLessons []string `json:"accumulated_lessons"` // What we've learned from previous attempts

	// Execution trace (the phenotype - what happened when this strategy was executed)
	Trace *ExecutionTrace `json:"trace,omitempty"`

	// Multi-objective scores for Pareto selection (5 dimensions)
	Scores PathScores `json:"scores"`

	// Feedback is the cached LLM evaluation feedback for this candidate.
	// This avoids redundant re-evaluation calls when the candidate is selected from the archive.
	Feedback string `json:"feedback,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// PathScores holds multi-objective scores for Pareto front selection.
// All scores are normalized to [0, 1] where higher is better.
type PathScores struct {
	AnswerQuality float64 `json:"answer_quality"` // Primary: correctness + no hallucinations
	Efficiency    float64 `json:"efficiency"`     // Fewer tool calls = better (1 - toolCalls/maxCalls)
	TokenCost     float64 `json:"token_cost"`     // Lower token usage = better (1 - tokens/maxTokens)
	Robustness    float64 `json:"robustness"`     // Error handling + self-correction ability
	Latency       float64 `json:"latency"`        // Time to answer (inverted: fast = high)
}

// ExecutionTrace captures what happened during one path attempt.
// This is the "phenotype" - the observable result of executing a strategy.
type ExecutionTrace struct {
	Query          string           `json:"query"`
	ToolCalls      []ToolCallRecord `json:"tool_calls"`
	ReasoningSteps []string         `json:"reasoning_steps,omitempty"`
	FinalAnswer    string           `json:"final_answer"`
	TotalTokens    int              `json:"total_tokens"`
	DurationMs     int64            `json:"duration_ms"`
}

// ToolCallRecord captures a single tool invocation within an execution trace.
type ToolCallRecord struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Result    any            `json:"result,omitempty"`
	Success   bool           `json:"success"`
	Error     string         `json:"error,omitempty"`
}

// PathSearchResult represents the outcome of a GEPA path search.
type PathSearchResult struct {
	BestPath   *PathCandidate `json:"best_path"`
	Answer     string         `json:"answer"`
	Score      float64        `json:"score"`
	Iterations int            `json:"iterations"`
}

// PathSearchConfig configures a GEPA path search run.
type PathSearchConfig struct {
	MaxGenerations int     `json:"max_generations"`  // Maximum evolutionary generations
	BranchesPerGen int     `json:"branches_per_gen"` // Parallel paths to explore per generation
	TargetScore    float64 `json:"target_score"`     // Early exit threshold (0-1)
}

// NewPathCandidate creates a new PathCandidate with the given parameters.
func NewPathCandidate(id, runID string, generation int, parentIDs []string, strategyPrompt string, lessons []string) *PathCandidate {
	// Ensure slices are never nil for consistent JSON serialization
	if lessons == nil {
		lessons = []string{}
	}
	if parentIDs == nil {
		parentIDs = []string{}
	}

	return &PathCandidate{
		ID:                 id,
		RunID:              runID,
		Generation:         generation,
		ParentIDs:          parentIDs,
		StrategyPrompt:     strategyPrompt,
		AccumulatedLessons: lessons,
		Scores:             PathScores{},
		CreatedAt:          time.Now().UTC(),
	}
}

// NewSeedCandidate creates a generation-0 seed candidate with a default strategy.
// This is the starting point for GEPA path search evolution.
func NewSeedCandidate(id, runID string) *PathCandidate {
	return NewPathCandidate(
		id,
		runID,
		0, // Generation 0
		nil,
		defaultSeedStrategy(),
		nil,
	)
}

// defaultSeedStrategy returns the default starting strategy for path search.
func defaultSeedStrategy() string {
	return `You are solving a query task.
1. First understand what information is needed
2. Identify relevant data sources and relationships
3. Construct appropriate queries or tool calls
4. Verify results make sense before concluding
5. Synthesize findings into a clear, accurate answer`
}

// NewPathSearchConfig creates a PathSearchConfig with sensible defaults.
func NewPathSearchConfig() *PathSearchConfig {
	return &PathSearchConfig{
		MaxGenerations: 5,
		BranchesPerGen: 3,
		TargetScore:    0.85,
	}
}

// --- PathScoreWeights for weighted aggregation ---

// PathScoreWeights holds dimension weights for weighted aggregation.
type PathScoreWeights struct {
	AnswerQuality float64 `json:"answer_quality"`
	Efficiency    float64 `json:"efficiency"`
	TokenCost     float64 `json:"token_cost"`
	Robustness    float64 `json:"robustness"`
	Latency       float64 `json:"latency"`
}

// DefaultPathScoreWeights returns default weights emphasizing answer quality.
func DefaultPathScoreWeights() PathScoreWeights {
	return PathScoreWeights{
		AnswerQuality: 0.40,
		Efficiency:    0.15,
		TokenCost:     0.15,
		Robustness:    0.15,
		Latency:       0.15,
	}
}

// --- Helper methods on PathScores for aggregation ---

// WeightedSum calculates a weighted sum of all score dimensions.
// Weights should sum to 1.0 for normalized output.
func (s PathScores) WeightedSum(weights PathScoreWeights) float64 {
	return s.AnswerQuality*weights.AnswerQuality +
		s.Efficiency*weights.Efficiency +
		s.TokenCost*weights.TokenCost +
		s.Robustness*weights.Robustness +
		s.Latency*weights.Latency
}

// Mean returns the simple average of all score dimensions.
func (s PathScores) Mean() float64 {
	return (s.AnswerQuality + s.Efficiency + s.TokenCost + s.Robustness + s.Latency) / 5.0
}

// Min returns the minimum score across all dimensions.
func (s PathScores) Min() float64 {
	return math.Min(s.AnswerQuality,
		math.Min(s.Efficiency,
			math.Min(s.TokenCost,
				math.Min(s.Robustness, s.Latency))))
}

// Max returns the maximum score across all dimensions.
func (s PathScores) Max() float64 {
	return math.Max(s.AnswerQuality,
		math.Max(s.Efficiency,
			math.Max(s.TokenCost,
				math.Max(s.Robustness, s.Latency))))
}

// AsSlice returns the scores as a slice for Pareto calculations.
// Order: [AnswerQuality, Efficiency, TokenCost, Robustness, Latency]
func (s PathScores) AsSlice() []float64 {
	return []float64{s.AnswerQuality, s.Efficiency, s.TokenCost, s.Robustness, s.Latency}
}

// FromSlice populates PathScores from a slice.
// Order: [AnswerQuality, Efficiency, TokenCost, Robustness, Latency]
func (s *PathScores) FromSlice(values []float64) {
	if len(values) >= 5 {
		s.AnswerQuality = values[0]
		s.Efficiency = values[1]
		s.TokenCost = values[2]
		s.Robustness = values[3]
		s.Latency = values[4]
	}
}

// Dominates returns true if this score Pareto-dominates other.
// Dominance: better or equal on ALL dimensions, strictly better on at least one.
func (s PathScores) Dominates(other PathScores) bool {
	betterOrEqual := s.AnswerQuality >= other.AnswerQuality &&
		s.Efficiency >= other.Efficiency &&
		s.TokenCost >= other.TokenCost &&
		s.Robustness >= other.Robustness &&
		s.Latency >= other.Latency

	strictlyBetter := s.AnswerQuality > other.AnswerQuality ||
		s.Efficiency > other.Efficiency ||
		s.TokenCost > other.TokenCost ||
		s.Robustness > other.Robustness ||
		s.Latency > other.Latency

	return betterOrEqual && strictlyBetter
}

// EuclideanDistance calculates the Euclidean distance to another score vector.
// Useful for crowding distance calculations in NSGA-II.
func (s PathScores) EuclideanDistance(other PathScores) float64 {
	dq := s.AnswerQuality - other.AnswerQuality
	de := s.Efficiency - other.Efficiency
	dt := s.TokenCost - other.TokenCost
	dr := s.Robustness - other.Robustness
	dl := s.Latency - other.Latency

	return math.Sqrt(dq*dq + de*de + dt*dt + dr*dr + dl*dl)
}

// IsZero returns true if all scores are zero (uninitialized).
func (s PathScores) IsZero() bool {
	return s.AnswerQuality == 0 && s.Efficiency == 0 && s.TokenCost == 0 &&
		s.Robustness == 0 && s.Latency == 0
}

// --- Helper methods on PathCandidate ---

// SetTrace sets the execution trace.
func (c *PathCandidate) SetTrace(trace *ExecutionTrace) {
	c.Trace = trace
}

// SetScores sets the path scores.
func (c *PathCandidate) SetScores(scores PathScores) {
	c.Scores = scores
}

// SetFeedback sets the cached evaluation feedback for this candidate.
func (c *PathCandidate) SetFeedback(feedback string) {
	c.Feedback = feedback
}

// AddLesson appends a lesson to the accumulated lessons.
func (c *PathCandidate) AddLesson(lesson string) {
	c.AccumulatedLessons = append(c.AccumulatedLessons, lesson)
}

// AddLessons appends multiple lessons to the accumulated lessons.
func (c *PathCandidate) AddLessons(lessons []string) {
	c.AccumulatedLessons = append(c.AccumulatedLessons, lessons...)
}

// HasTrace returns true if this candidate has been executed.
func (c *PathCandidate) HasTrace() bool {
	return c.Trace != nil
}

// IsSeed returns true if this is a generation-0 seed candidate.
func (c *PathCandidate) IsSeed() bool {
	return c.Generation == 0
}

// --- Helper methods on ExecutionTrace ---

// SuccessfulToolCalls returns the count of successful tool calls.
func (t *ExecutionTrace) SuccessfulToolCalls() int {
	count := 0
	for _, tc := range t.ToolCalls {
		if tc.Success {
			count++
		}
	}
	return count
}

// FailedToolCalls returns the count of failed tool calls.
func (t *ExecutionTrace) FailedToolCalls() int {
	count := 0
	for _, tc := range t.ToolCalls {
		if !tc.Success {
			count++
		}
	}
	return count
}

// ToolCallSuccessRate returns the ratio of successful to total tool calls.
// Returns 1.0 if there are no tool calls.
func (t *ExecutionTrace) ToolCallSuccessRate() float64 {
	if len(t.ToolCalls) == 0 {
		return 1.0
	}
	return float64(t.SuccessfulToolCalls()) / float64(len(t.ToolCalls))
}

// HasAnswer returns true if the trace produced a non-empty final answer.
func (t *ExecutionTrace) HasAnswer() bool {
	return t.FinalAnswer != ""
}

// TotalToolCalls returns the total number of tool calls.
func (t *ExecutionTrace) TotalToolCalls() int {
	return len(t.ToolCalls)
}

// --- Helper methods on PathSearchResult ---

// IsSuccessful returns true if a valid answer was found.
func (r *PathSearchResult) IsSuccessful() bool {
	return r.BestPath != nil && r.Answer != "" && r.Score > 0
}

// --- Helper methods on PathSearchConfig ---

// Validate checks if the configuration is valid.
func (c *PathSearchConfig) Validate() bool {
	return c.MaxGenerations > 0 &&
		c.BranchesPerGen > 0 &&
		c.TargetScore > 0 && c.TargetScore <= 1.0
}

// WithMaxGenerations returns a copy with updated MaxGenerations.
func (c PathSearchConfig) WithMaxGenerations(n int) PathSearchConfig {
	c.MaxGenerations = n
	return c
}

// WithBranchesPerGen returns a copy with updated BranchesPerGen.
func (c PathSearchConfig) WithBranchesPerGen(n int) PathSearchConfig {
	c.BranchesPerGen = n
	return c
}

// WithTargetScore returns a copy with updated TargetScore.
func (c PathSearchConfig) WithTargetScore(score float64) PathSearchConfig {
	c.TargetScore = score
	return c
}
