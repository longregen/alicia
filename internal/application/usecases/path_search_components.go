package usecases

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"golang.org/x/sync/errgroup"
)

// ============================================================================
// Prompt Constants
// ============================================================================

// MemoryExtractionSeedPrompt is the baseline seed prompt for memory extraction.
var MemoryExtractionSeedPrompt = `You are a memory extraction specialist. Extract factual information from conversations that should be remembered for future reference.

EXTRACTION CRITERIA:
1. FACTUAL: Only extract objective facts, preferences, and explicit information
2. SIGNIFICANT: Only extract information that would be useful to recall later
3. SPECIFIC: Extract concrete details, not vague observations
4. PERSISTENT: Only extract information that remains true over time
5. USER-CENTRIC: Focus on facts about the user, their preferences, projects, and context

WHAT TO EXTRACT:
- Personal preferences (favorite foods, music, working styles)
- Biographical facts (names, birthdays, locations, roles)
- Project/work context (technologies used, deadlines, team members)
- Explicit instructions or rules the user sets
- Domain-specific knowledge relevant to the user's work
- Important relationships and connections

WHAT NOT TO EXTRACT:
- Conversational filler ("that's interesting", "I see", "thanks")
- Transient information (current time, weather, temporary states)
- Information that's only relevant to the current conversation
- Opinions about generic topics that aren't personal preferences
- Information already implied by context
- Trivial details with no future utility

RESPONSE FORMAT:
- extracted_facts: JSON array of fact strings, each fact should be self-contained and clear
- importance_scores: JSON array of floats (0.0-1.0) corresponding to each fact, where:
  * 0.9-1.0: Critical information (names, allergies, explicit rules)
  * 0.7-0.8: Important preferences or project context
  * 0.5-0.6: Useful context or secondary details
  * 0.3-0.4: Minor details that might be occasionally useful
  * 0.0-0.2: Rarely useful (should probably not be extracted)
- extraction_reasoning: Brief explanation of what was extracted and why, or why nothing was extracted

IMPORTANT: It is completely valid to extract NOTHING from conversational filler. Empty extraction is often the correct answer.`

// ToolResultMemorizationPrompt is used to analyze tool results for memorization worthiness.
var ToolResultMemorizationPrompt = `You are a memory analysis specialist. Your task is to determine whether tool results contain information worth storing as long-term memories.

CRITERIA FOR MEMORIZATION:
1. USER-SPECIFIC: Information that is personalized or specific to the user
2. DURABLE: Facts that will remain relevant for future conversations
3. ACTIONABLE: Information that could inform future responses or decisions
4. NOVEL: Information not already implied by common knowledge

WHAT TO MEMORIZE:
- User preferences discovered through tool use (e.g., favorite restaurants from search)
- Account details or configurations retrieved via tools
- Resolved technical issues and their solutions
- Project-specific information from file or database queries
- Contact information or relationships discovered
- User-specific data patterns or behaviors

WHAT NOT TO MEMORIZE:
- Transient data (current weather, time, live prices)
- Generic information available to anyone
- Error messages or failed operations
- Data too large to meaningfully summarize
- Information specific only to the current request
- Temporary states that will change

RESPONSE: Provide a JSON object with your analysis.`

// ============================================================================
// PathParetoArchive - Pareto archive for path search
// ============================================================================

// PathParetoArchive maintains non-dominated execution paths for GEPA Path Search.
// It uses Pareto dominance across 5 dimensions (AnswerQuality, Efficiency, TokenCost,
// Robustness, Latency) to maintain a diverse set of high-performing paths.
type PathParetoArchive struct {
	candidates []*models.PathCandidate
	maxSize    int
	mu         sync.RWMutex
}

// NewPathParetoArchive creates a new Pareto archive with the given maximum size
func NewPathParetoArchive(maxSize int) *PathParetoArchive {
	if maxSize <= 0 {
		maxSize = 50 // Default archive size
	}
	return &PathParetoArchive{
		candidates: make([]*models.PathCandidate, 0),
		maxSize:    maxSize,
	}
}

// Add inserts a candidate if it's non-dominated.
// Removes any existing candidates that the new one dominates.
// If the archive exceeds maxSize, prunes using crowding distance.
func (a *PathParetoArchive) Add(candidate *models.PathCandidate) {
	if candidate == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if dominated by any existing candidate
	for _, existing := range a.candidates {
		if a.dominates(existing.Scores, candidate.Scores) {
			return // Dominated, don't add
		}
	}

	// Remove any candidates dominated by the new one
	a.candidates = a.filterNonDominated(candidate.Scores)

	// Add new candidate
	a.candidates = append(a.candidates, candidate)

	// Prune using crowding distance if too large
	if len(a.candidates) > a.maxSize {
		a.pruneWithCrowding()
	}
}

// dominates returns true if scoreA dominates scoreB.
func (a *PathParetoArchive) dominates(scoreA, scoreB models.PathScores) bool {
	aVals := []float64{
		scoreA.AnswerQuality,
		scoreA.Efficiency,
		scoreA.TokenCost,
		scoreA.Robustness,
		scoreA.Latency,
	}
	bVals := []float64{
		scoreB.AnswerQuality,
		scoreB.Efficiency,
		scoreB.TokenCost,
		scoreB.Robustness,
		scoreB.Latency,
	}

	atLeastAsGood := true
	strictlyBetter := false

	for i := 0; i < len(aVals); i++ {
		if aVals[i] < bVals[i] {
			atLeastAsGood = false
			break
		}
		if aVals[i] > bVals[i] {
			strictlyBetter = true
		}
	}

	return atLeastAsGood && strictlyBetter
}

// filterNonDominated removes candidates dominated by the given scores
func (a *PathParetoArchive) filterNonDominated(newScores models.PathScores) []*models.PathCandidate {
	result := make([]*models.PathCandidate, 0, len(a.candidates))
	for _, c := range a.candidates {
		if !a.dominates(newScores, c.Scores) {
			result = append(result, c)
		}
	}
	return result
}

// SelectForMutation picks n candidates from the archive for mutation.
func (a *PathParetoArchive) SelectForMutation(n int) []*models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.candidates) == 0 || n <= 0 {
		return nil
	}

	if n >= len(a.candidates) {
		result := make([]*models.PathCandidate, len(a.candidates))
		copy(result, a.candidates)
		return result
	}

	return a.selectByCrowding(n)
}

// pruneWithCrowding removes the least diverse candidates when archive exceeds maxSize.
func (a *PathParetoArchive) pruneWithCrowding() {
	if len(a.candidates) <= a.maxSize {
		return
	}

	distances := a.calculateCrowdingDistances()

	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(a.candidates))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist
	})

	result := make([]*models.PathCandidate, a.maxSize)
	for i := 0; i < a.maxSize; i++ {
		result[i] = a.candidates[pairs[i].idx]
	}
	a.candidates = result
}

// selectByCrowding selects n candidates using crowding distance for diversity.
func (a *PathParetoArchive) selectByCrowding(n int) []*models.PathCandidate {
	if n <= 0 || len(a.candidates) == 0 {
		return nil
	}

	if n >= len(a.candidates) {
		result := make([]*models.PathCandidate, len(a.candidates))
		copy(result, a.candidates)
		return result
	}

	distances := a.calculateCrowdingDistances()

	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(a.candidates))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist
	})

	result := make([]*models.PathCandidate, n)
	for i := 0; i < n; i++ {
		result[i] = a.candidates[pairs[i].idx]
	}
	return result
}

// calculateCrowdingDistances computes NSGA-II style crowding distances.
func (a *PathParetoArchive) calculateCrowdingDistances() []float64 {
	n := len(a.candidates)
	if n == 0 {
		return nil
	}

	distances := make([]float64, n)

	dimensions := []func(*models.PathScores) float64{
		func(s *models.PathScores) float64 { return s.AnswerQuality },
		func(s *models.PathScores) float64 { return s.Efficiency },
		func(s *models.PathScores) float64 { return s.TokenCost },
		func(s *models.PathScores) float64 { return s.Robustness },
		func(s *models.PathScores) float64 { return s.Latency },
	}

	for _, getDim := range dimensions {
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool {
			return getDim(&a.candidates[indices[i]].Scores) < getDim(&a.candidates[indices[j]].Scores)
		})

		minVal := getDim(&a.candidates[indices[0]].Scores)
		maxVal := getDim(&a.candidates[indices[n-1]].Scores)
		dimRange := maxVal - minVal

		if dimRange == 0 {
			continue
		}

		distances[indices[0]] = 1e9
		distances[indices[n-1]] = 1e9

		for i := 1; i < n-1; i++ {
			neighborDist := getDim(&a.candidates[indices[i+1]].Scores) - getDim(&a.candidates[indices[i-1]].Scores)
			distances[indices[i]] += neighborDist / dimRange
		}
	}

	return distances
}

// GetParetoFront returns a copy of all candidates in the archive.
func (a *PathParetoArchive) GetParetoFront() []*models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*models.PathCandidate, len(a.candidates))
	copy(result, a.candidates)
	return result
}

// Size returns the number of candidates in the archive
func (a *PathParetoArchive) Size() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.candidates)
}

// ============================================================================
// PathMutator - Mutates path candidates using LLM reflection
// ============================================================================

// PathMutator mutates path candidates using LLM reflection.
type PathMutator struct {
	llm           ports.LLMService
	reflectionLLM ports.LLMService
}

// NewPathMutator creates a new PathMutator.
func NewPathMutator(llm, reflectionLLM ports.LLMService) *PathMutator {
	if reflectionLLM == nil {
		reflectionLLM = llm
	}
	return &PathMutator{
		llm:           llm,
		reflectionLLM: reflectionLLM,
	}
}

// MutateStrategy uses LLM to evolve the strategy based on execution trace.
func (m *PathMutator) MutateStrategy(ctx context.Context, candidate *models.PathCandidate, trace *models.ExecutionTrace, feedback string) (*models.PathCandidate, error) {
	if candidate == nil {
		return nil, fmt.Errorf("candidate cannot be nil")
	}
	if trace == nil {
		return nil, fmt.Errorf("trace cannot be nil")
	}

	lessonsStr := ""
	if len(candidate.AccumulatedLessons) > 0 {
		lessonsStr = "- " + strings.Join(candidate.AccumulatedLessons, "\n- ")
	} else {
		lessonsStr = "(none yet)"
	}

	prompt := fmt.Sprintf(`Analyze this execution trace and improve the strategy.

ORIGINAL QUERY: %s

STRATEGY USED:
%s

EXECUTION TRACE:
%s

FEEDBACK: %s

ACCUMULATED LESSONS:
%s

Based on what worked and what didn't, provide:
1. LESSONS_LEARNED: New lessons from this attempt (bullet points, each on its own line starting with "- ")
2. IMPROVED_STRATEGY: A better strategy prompt for the next attempt

The improved strategy should be specific, actionable, and address the failures observed.

Format your response exactly like this:
LESSONS_LEARNED:
- lesson 1
- lesson 2

IMPROVED_STRATEGY:
Your improved strategy text here...`,
		trace.Query,
		candidate.StrategyPrompt,
		formatExecutionTrace(trace),
		feedback,
		lessonsStr,
	)

	response, err := m.reflectionLLM.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for mutation: %w", err)
	}

	newLessons := parseMutationLessons(response.Content)
	newStrategy := parseMutationStrategy(response.Content)

	if newStrategy == "" {
		newStrategy = candidate.StrategyPrompt + "\n\nAdditional guidance based on previous attempt: " + feedback
	}

	allLessons := uniqueMergeLessons(candidate.AccumulatedLessons, newLessons)

	return &models.PathCandidate{
		ID:                 generatePathCandidateID(),
		RunID:              candidate.RunID,
		Generation:         candidate.Generation + 1,
		ParentIDs:          []string{candidate.ID},
		StrategyPrompt:     newStrategy,
		AccumulatedLessons: allLessons,
		CreatedAt:          time.Now(),
	}, nil
}

// Crossover merges strategies from two Pareto-optimal paths.
func (m *PathMutator) Crossover(ctx context.Context, parent1, parent2 *models.PathCandidate) (*models.PathCandidate, error) {
	if parent1 == nil || parent2 == nil {
		return nil, fmt.Errorf("both parents must be non-nil")
	}

	lessons1 := "(none)"
	if len(parent1.AccumulatedLessons) > 0 {
		lessons1 = "- " + strings.Join(parent1.AccumulatedLessons, "\n- ")
	}
	lessons2 := "(none)"
	if len(parent2.AccumulatedLessons) > 0 {
		lessons2 = "- " + strings.Join(parent2.AccumulatedLessons, "\n- ")
	}

	prompt := fmt.Sprintf(`Merge these two successful strategies into one.

STRATEGY 1 (from path with scores: quality=%.2f, efficiency=%.2f):
%s

Lessons learned:
%s

STRATEGY 2 (from path with scores: quality=%.2f, efficiency=%.2f):
%s

Lessons learned:
%s

Create a MERGED_STRATEGY that combines the best elements:
- Keep what makes each strategy effective
- Resolve conflicts in favor of robustness
- Be specific and actionable

Format your response exactly like this:
MERGED_STRATEGY:
Your merged strategy text here...`,
		parent1.Scores.AnswerQuality, parent1.Scores.Efficiency,
		parent1.StrategyPrompt,
		lessons1,
		parent2.Scores.AnswerQuality, parent2.Scores.Efficiency,
		parent2.StrategyPrompt,
		lessons2,
	)

	response, err := m.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for crossover: %w", err)
	}

	mergedStrategy := parseMergedStrategyText(response.Content)
	if mergedStrategy == "" {
		mergedStrategy = fmt.Sprintf("Combined approach:\n\nFrom strategy 1:\n%s\n\nFrom strategy 2:\n%s",
			parent1.StrategyPrompt, parent2.StrategyPrompt)
	}

	newGen := parent1.Generation
	if parent2.Generation > newGen {
		newGen = parent2.Generation
	}
	newGen++

	return &models.PathCandidate{
		ID:                 generatePathCandidateID(),
		RunID:              parent1.RunID,
		Generation:         newGen,
		ParentIDs:          []string{parent1.ID, parent2.ID},
		StrategyPrompt:     mergedStrategy,
		AccumulatedLessons: uniqueMergeLessons(parent1.AccumulatedLessons, parent2.AccumulatedLessons),
		CreatedAt:          time.Now(),
	}, nil
}

// ============================================================================
// PathEvaluator - Evaluates execution paths across multiple dimensions
// ============================================================================

// PathEvaluator evaluates execution paths across multiple dimensions for GEPA Path Search.
type PathEvaluator struct {
	llm ports.LLMService
}

// NewPathEvaluator creates a new PathEvaluator.
func NewPathEvaluator(llm ports.LLMService) *PathEvaluator {
	return &PathEvaluator{llm: llm}
}

// Evaluate scores a path across 5 Pareto dimensions and returns feedback for mutation.
func (e *PathEvaluator) Evaluate(ctx context.Context, query string, trace *models.ExecutionTrace) (models.PathScores, string, error) {
	if trace == nil {
		return models.PathScores{}, "", fmt.Errorf("trace cannot be nil")
	}

	scores := models.PathScores{}

	// STAGE 1: Fast heuristic screening
	heuristicScore := e.heuristicScreen(trace)

	// STAGE 2: LLM evaluation (only for promising candidates)
	if heuristicScore >= 0.4 {
		var (
			mu               sync.Mutex
			answerQuality    float64
			answerQualityErr error
			hallucinated     bool
			hallucinationErr error
			specificityMult  float64
			specificityErr   error
			robustness       float64
			robustnessErr    error
		)

		g, gCtx := errgroup.WithContext(ctx)

		g.Go(func() error {
			q, err := e.llmJudgeAnswerQuality(gCtx, query, trace)
			mu.Lock()
			answerQuality = q
			answerQualityErr = err
			mu.Unlock()
			return nil
		})

		g.Go(func() error {
			h, err := e.llmCheckHallucinations(gCtx, trace)
			mu.Lock()
			hallucinated = h
			hallucinationErr = err
			mu.Unlock()
			return nil
		})

		g.Go(func() error {
			s, err := e.llmJudgeSpecificity(gCtx, query, trace)
			mu.Lock()
			specificityMult = s
			specificityErr = err
			mu.Unlock()
			return nil
		})

		g.Go(func() error {
			r, err := e.evaluateRobustness(gCtx, trace)
			mu.Lock()
			robustness = r
			robustnessErr = err
			mu.Unlock()
			return nil
		})

		_ = g.Wait()

		if answerQualityErr != nil {
			scores.AnswerQuality = heuristicScore
		} else {
			scores.AnswerQuality = answerQuality
		}

		if hallucinationErr == nil && hallucinated {
			scores.AnswerQuality *= 0.3
		}

		if specificityErr == nil {
			scores.AnswerQuality *= specificityMult
		}

		if robustnessErr != nil {
			failedCalls := countEvalFailedToolCalls(trace)
			if failedCalls == 0 {
				robustness = 1.0
			} else {
				robustness = maxFloatVal(0.0, 1.0-float64(failedCalls)*0.2)
			}
		}
		scores.Robustness = robustness
	} else {
		scores.AnswerQuality = heuristicScore
		failedCalls := countEvalFailedToolCalls(trace)
		if failedCalls == 0 {
			scores.Robustness = 1.0
		} else {
			scores.Robustness = maxFloatVal(0.0, 1.0-float64(failedCalls)*0.2)
		}
	}

	// DIMENSION: Efficiency
	toolCallCount := float64(len(trace.ToolCalls))
	scores.Efficiency = 1.0 - minFloatVal(1.0, toolCallCount/10.0)

	// DIMENSION: Token cost
	scores.TokenCost = 1.0 - minFloatVal(1.0, float64(trace.TotalTokens)/10000.0)

	// DIMENSION: Latency
	scores.Latency = 1.0 - minFloatVal(1.0, float64(trace.DurationMs)/30000.0)

	feedback := e.generateFeedback(ctx, query, trace, scores)

	return scores, feedback, nil
}

// heuristicScreen provides fast initial screening.
func (e *PathEvaluator) heuristicScreen(trace *models.ExecutionTrace) float64 {
	score := 0.0

	if trace.FinalAnswer != "" && !isEvalNonAnswer(trace.FinalAnswer) {
		score += 0.3
	}

	if containsEvalSpecificData(trace.FinalAnswer) {
		score += 0.2
	}

	totalCalls := float64(len(trace.ToolCalls))
	if totalCalls > 0 {
		successRate := float64(countEvalSuccessfulToolCalls(trace)) / totalCalls
		score += 0.3 * successRate
	} else {
		score += 0.15
	}

	answerLen := len(trace.FinalAnswer)
	if answerLen > 20 && answerLen < 2000 {
		score += 0.2
	} else if answerLen > 0 && answerLen <= 20 {
		score += 0.1
	}

	return score
}

func (e *PathEvaluator) llmJudgeAnswerQuality(ctx context.Context, query string, trace *models.ExecutionTrace) (float64, error) {
	promptText := fmt.Sprintf(`Rate the quality of this answer on a scale of 0-10.

QUERY: %s

ANSWER: %s

Consider holistically:
- Relevance: Does it address what was asked?
- Accuracy: Is the information correct based on the available data?
- Completeness: Does it fully answer the question?
- Clarity: Is it well-organized and easy to understand?

Output format: SCORE: [0-10] REASON: [brief explanation]`, query, truncateForEvalPrompt(trace.FinalAnswer, 1500))

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return 0.0, fmt.Errorf("LLM call failed: %w", err)
	}

	score := parseEvalScoreFromResponse(resp.Content)
	return score / 10.0, nil
}

func (e *PathEvaluator) llmCheckHallucinations(ctx context.Context, trace *models.ExecutionTrace) (bool, error) {
	toolOutputs := formatEvalToolOutputs(trace.ToolCalls)
	if toolOutputs == "" {
		return false, nil
	}

	promptText := fmt.Sprintf(`Check if this answer contains hallucinations (claims not supported by the tool outputs).

TOOL OUTPUTS:
%s

ANSWER:
%s

Does the answer make any specific factual claims (numbers, names, dates, etc.) that are NOT supported by the tool outputs above?
- Claims that are reasonable inferences from the data are OK
- Claims that contradict or go beyond the data are hallucinations

Output: HALLUCINATED: true/false REASON: [explanation]`, truncateForEvalPrompt(toolOutputs, 2000), truncateForEvalPrompt(trace.FinalAnswer, 1000))

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return false, fmt.Errorf("LLM call failed: %w", err)
	}

	return strings.Contains(strings.ToLower(resp.Content), "hallucinated: true"), nil
}

func (e *PathEvaluator) llmJudgeSpecificity(ctx context.Context, query string, trace *models.ExecutionTrace) (float64, error) {
	promptText := fmt.Sprintf(`Is the specificity of this answer appropriate for the query?

QUERY: %s
ANSWER: %s

Evaluate:
- If query asks for specific data and answer is vague: return LOW score (0.5-0.7)
- If query is open-ended and answer is appropriately general: return HIGH score (0.9-1.0)
- If answer provides concrete data when expected: return HIGH score (0.9-1.0)
- If answer is overly specific when not needed: return MEDIUM score (0.8-0.9)

Output: SPECIFICITY_SCORE: [0.5-1.0]`, query, truncateForEvalPrompt(trace.FinalAnswer, 1000))

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return 1.0, fmt.Errorf("LLM call failed: %w", err)
	}

	score := parseEvalSpecificityScore(resp.Content)
	if score < 0.5 {
		score = 0.5
	}
	if score > 1.0 {
		score = 1.0
	}
	return score, nil
}

func (e *PathEvaluator) evaluateRobustness(ctx context.Context, trace *models.ExecutionTrace) (float64, error) {
	score := 1.0

	errors := 0
	recoveries := 0
	for i, tc := range trace.ToolCalls {
		if !tc.Success {
			errors++
			if i+1 < len(trace.ToolCalls) && trace.ToolCalls[i+1].Success {
				recoveries++
			}
		}
	}

	if errors > 0 {
		severityPenalty, err := e.llmJudgeErrorSeverity(ctx, trace, errors)
		if err != nil {
			severityPenalty = minFloatVal(0.5, float64(errors)*0.15)
		}
		score -= severityPenalty
	}

	if recoveries > 0 {
		score += 0.1 * float64(recoveries)
	}

	return maxFloatVal(0.0, minFloatVal(1.0, score)), nil
}

func (e *PathEvaluator) llmJudgeErrorSeverity(ctx context.Context, trace *models.ExecutionTrace, errorCount int) (float64, error) {
	errorDetails := formatEvalErrors(trace.ToolCalls)
	if errorDetails == "" {
		return 0.0, nil
	}

	hasAnswer := trace.FinalAnswer != "" && !isEvalNonAnswer(trace.FinalAnswer)

	promptText := fmt.Sprintf(`Rate the severity of these errors (return a penalty from 0.0 to 0.5):

ERRORS (%d total):
%s

FINAL ANSWER ACHIEVED: %v

Severity guidelines:
- If errors were recoverable and didn't affect final answer: LOW penalty (0.0-0.1)
- If errors caused partial data loss but answer is still useful: MEDIUM penalty (0.1-0.3)
- If errors were critical and unrecoverable, preventing a good answer: HIGH penalty (0.3-0.5)

Output: SEVERITY_PENALTY: [0.0-0.5]`, errorCount, truncateForEvalPrompt(errorDetails, 1000), hasAnswer)

	resp, err := e.llm.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: promptText}})
	if err != nil {
		return 0.0, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseEvalSeverityPenalty(resp.Content), nil
}

func (e *PathEvaluator) generateFeedback(ctx context.Context, query string, trace *models.ExecutionTrace, scores models.PathScores) string {
	var feedbackParts []string

	if scores.AnswerQuality < 0.3 {
		feedbackParts = append(feedbackParts, "Answer quality is very low - the response may be incorrect, incomplete, or irrelevant.")
	} else if scores.AnswerQuality < 0.5 {
		feedbackParts = append(feedbackParts, "Answer quality is below average - consider improving accuracy or completeness.")
	} else if scores.AnswerQuality < 0.7 {
		feedbackParts = append(feedbackParts, "Answer quality is moderate - there's room for improvement.")
	}

	failedTools := countEvalFailedToolCalls(trace)
	if failedTools > 0 {
		feedbackParts = append(feedbackParts, fmt.Sprintf("%d tool call(s) failed - consider error handling or alternative approaches.", failedTools))
	}

	wastedCalls := countEvalWastedToolCalls(trace)
	if wastedCalls > 0 {
		feedbackParts = append(feedbackParts,
			fmt.Sprintf("%d tool call(s) had results that weren't reflected in the answer - consider more focused exploration.", wastedCalls))
	}

	if len(trace.ToolCalls) > 5 && scores.AnswerQuality < 0.7 {
		feedbackParts = append(feedbackParts, "Many tool calls but moderate quality - the strategy may be inefficient.")
	} else if len(trace.ToolCalls) > 8 {
		feedbackParts = append(feedbackParts, "High number of tool calls - consider a more direct approach.")
	}

	if scores.TokenCost < 0.3 {
		feedbackParts = append(feedbackParts, "Very high token usage - consider more concise prompts or fewer iterations.")
	} else if scores.TokenCost < 0.5 {
		feedbackParts = append(feedbackParts, "High token usage - there may be opportunities for optimization.")
	}

	if scores.Latency < 0.3 {
		feedbackParts = append(feedbackParts, "Very slow execution - consider a more direct approach or parallel operations.")
	} else if scores.Latency < 0.5 {
		feedbackParts = append(feedbackParts, "Slow execution - there may be opportunities to speed up the process.")
	}

	if trace.FinalAnswer == "" || isEvalNonAnswer(trace.FinalAnswer) {
		feedbackParts = append(feedbackParts, "No meaningful answer was produced - the strategy needs significant revision.")
	}

	if hasEvalRepeatedFailures(trace) {
		feedbackParts = append(feedbackParts, "Multiple similar failures detected - consider a different approach entirely.")
	}

	if len(feedbackParts) == 0 {
		return "Path executed successfully with good results."
	}

	return strings.Join(feedbackParts, " ")
}

// ============================================================================
// Helper functions
// ============================================================================

func formatExecutionTrace(trace *models.ExecutionTrace) string {
	if trace == nil {
		return "(no trace available)"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Query: %s\n", trace.Query))
	sb.WriteString(fmt.Sprintf("Duration: %dms\n", trace.DurationMs))
	sb.WriteString(fmt.Sprintf("Total Tokens: %d\n\n", trace.TotalTokens))

	if len(trace.ReasoningSteps) > 0 {
		sb.WriteString("Reasoning Steps:\n")
		for i, step := range trace.ReasoningSteps {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	}

	if len(trace.ToolCalls) > 0 {
		sb.WriteString("Tool Calls:\n")
		for i, tc := range trace.ToolCalls {
			status := "SUCCESS"
			if !tc.Success {
				status = fmt.Sprintf("FAILED: %s", tc.Error)
			}
			sb.WriteString(fmt.Sprintf("  %d. %s(%v) -> %s\n", i+1, tc.ToolName, formatTraceArgs(tc.Arguments), status))
			if tc.Success && tc.Result != nil {
				resultStr := fmt.Sprintf("%v", tc.Result)
				if len(resultStr) > 200 {
					resultStr = resultStr[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("     Result: %s\n", resultStr))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Final Answer: %s\n", truncateStringForTrace(trace.FinalAnswer, 500)))

	return sb.String()
}

func formatTraceArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	var parts []string
	for k, v := range args {
		valStr := fmt.Sprintf("%v", v)
		if len(valStr) > 50 {
			valStr = valStr[:50] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%q", k, valStr))
	}
	return strings.Join(parts, ", ")
}

func parseMutationLessons(response string) []string {
	lessonsRegex := regexp.MustCompile(`(?is)LESSONS_LEARNED:\s*(.*?)(?:IMPROVED_STRATEGY:|MERGED_STRATEGY:|$)`)
	matches := lessonsRegex.FindStringSubmatch(response)

	var lessons []string
	if len(matches) > 1 {
		lessonsSection := strings.TrimSpace(matches[1])
		lines := strings.Split(lessonsSection, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "-")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimPrefix(line, "\u2022")
			line = strings.TrimSpace(line)
			if line != "" && len(line) > 3 {
				lessons = append(lessons, line)
			}
		}
	}
	return lessons
}

func parseMutationStrategy(response string) string {
	strategyRegex := regexp.MustCompile(`(?is)IMPROVED_STRATEGY:\s*(.*)`)
	matches := strategyRegex.FindStringSubmatch(response)

	if len(matches) > 1 {
		strategy := strings.TrimSpace(matches[1])
		if idx := strings.Index(strings.ToUpper(strategy), "LESSONS_LEARNED:"); idx > 0 {
			strategy = strings.TrimSpace(strategy[:idx])
		}
		return strategy
	}
	return ""
}

func parseMergedStrategyText(response string) string {
	strategyRegex := regexp.MustCompile(`(?is)MERGED_STRATEGY:\s*(.*)`)
	matches := strategyRegex.FindStringSubmatch(response)

	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return parseMutationStrategy(response)
}

func uniqueMergeLessons(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range a {
		normalized := strings.TrimSpace(strings.ToLower(s))
		if !seen[normalized] && s != "" {
			seen[normalized] = true
			result = append(result, s)
		}
	}

	for _, s := range b {
		normalized := strings.TrimSpace(strings.ToLower(s))
		if !seen[normalized] && s != "" {
			seen[normalized] = true
			result = append(result, s)
		}
	}

	return result
}

func generatePathCandidateID() string {
	return "path_" + uuid.New().String()[:8]
}

func truncateStringForTrace(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func containsEvalSpecificData(answer string) bool {
	if answer == "" {
		return false
	}

	numberRegex := regexp.MustCompile(`\d+\.?\d*%?`)
	if numberRegex.MatchString(answer) {
		return true
	}

	words := strings.Fields(answer)
	capitalizedCount := 0
	for _, word := range words {
		if len(word) > 1 {
			runes := []rune(word)
			if unicode.IsUpper(runes[0]) && !isEvalCommonWord(strings.ToLower(word)) {
				capitalizedCount++
			}
		}
	}
	if capitalizedCount >= 2 {
		return true
	}

	dateRegex := regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{2,4}|\d{4}[/-]\d{1,2}[/-]\d{1,2}|(?i)(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2}`)
	if dateRegex.MatchString(answer) {
		return true
	}

	return false
}

func countEvalSuccessfulToolCalls(trace *models.ExecutionTrace) int {
	count := 0
	for _, tc := range trace.ToolCalls {
		if tc.Success {
			count++
		}
	}
	return count
}

func countEvalFailedToolCalls(trace *models.ExecutionTrace) int {
	count := 0
	for _, tc := range trace.ToolCalls {
		if !tc.Success {
			count++
		}
	}
	return count
}

func countEvalWastedToolCalls(trace *models.ExecutionTrace) int {
	if trace.FinalAnswer == "" {
		return len(trace.ToolCalls)
	}

	answerLower := strings.ToLower(trace.FinalAnswer)
	wastedCount := 0

	for _, tc := range trace.ToolCalls {
		if !tc.Success {
			continue
		}

		resultStr := fmt.Sprintf("%v", tc.Result)
		if resultStr == "" || resultStr == "<nil>" {
			continue
		}

		keyTerms := extractEvalKeyTerms(resultStr)
		foundInAnswer := false
		for _, term := range keyTerms {
			if strings.Contains(answerLower, strings.ToLower(term)) {
				foundInAnswer = true
				break
			}
		}

		if !foundInAnswer {
			wastedCount++
		}
	}

	return wastedCount
}

func formatEvalToolOutputs(toolCalls []models.ToolCallRecord) string {
	var sb strings.Builder

	for i, tc := range toolCalls {
		if tc.Success && tc.Result != nil {
			resultStr := fmt.Sprintf("%v", tc.Result)
			if len(resultStr) > 500 {
				resultStr = resultStr[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("[Tool %d: %s]\n%s\n\n", i+1, tc.ToolName, resultStr))
		}
	}

	return sb.String()
}

func formatEvalErrors(toolCalls []models.ToolCallRecord) string {
	var sb strings.Builder

	for i, tc := range toolCalls {
		if !tc.Success {
			sb.WriteString(fmt.Sprintf("[Error %d: %s]\n", i+1, tc.ToolName))
			if tc.Error != "" {
				sb.WriteString(fmt.Sprintf("  Error: %s\n", tc.Error))
			}
			sb.WriteString(fmt.Sprintf("  Arguments: %v\n\n", tc.Arguments))
		}
	}

	return sb.String()
}

func parseEvalScoreFromResponse(response string) float64 {
	scoreRegex := regexp.MustCompile(`(?i)SCORE:\s*(\d+(?:\.\d+)?)`)
	matches := scoreRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		score, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return score
		}
	}

	numRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)`)
	matches = numRegex.FindStringSubmatch(strings.TrimSpace(response))
	if len(matches) > 1 {
		score, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return score
		}
	}

	return 5.0
}

func parseEvalSpecificityScore(response string) float64 {
	scoreRegex := regexp.MustCompile(`(?i)SPECIFICITY_SCORE:\s*(\d+(?:\.\d+)?)`)
	matches := scoreRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		score, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return score
		}
	}
	return 1.0
}

func parseEvalSeverityPenalty(response string) float64 {
	penaltyRegex := regexp.MustCompile(`(?i)SEVERITY_PENALTY:\s*(\d+(?:\.\d+)?)`)
	matches := penaltyRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		penalty, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return minFloatVal(0.5, maxFloatVal(0.0, penalty))
		}
	}
	return 0.1
}

func isEvalNonAnswer(answer string) bool {
	answer = strings.ToLower(strings.TrimSpace(answer))
	nonAnswerPhrases := []string{
		"unable to determine",
		"i don't know",
		"i cannot",
		"i can't",
		"no information available",
		"insufficient data",
		"cannot answer",
		"unable to answer",
		"i'm not sure",
		"i am not sure",
	}
	for _, phrase := range nonAnswerPhrases {
		if strings.Contains(answer, phrase) {
			return true
		}
	}
	return false
}

func isEvalCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true,
		"i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "what": true, "which": true, "who": true,
		"when": true, "where": true, "why": true, "how": true,
		"this": true, "that": true, "these": true, "those": true,
		"and": true, "but": true, "or": true, "nor": true, "for": true,
		"yet": true, "so": true, "if": true, "then": true, "else": true,
		"however": true, "therefore": true, "thus": true, "hence": true,
	}
	return commonWords[word]
}

func extractEvalKeyTerms(s string) []string {
	var terms []string

	numRegex := regexp.MustCompile(`\d+\.?\d*`)
	terms = append(terms, numRegex.FindAllString(s, -1)...)

	quotedRegex := regexp.MustCompile(`"([^"]+)"`)
	for _, match := range quotedRegex.FindAllStringSubmatch(s, -1) {
		if len(match) > 1 {
			terms = append(terms, match[1])
		}
	}

	words := strings.Fields(s)
	for _, word := range words {
		if len(word) > 2 {
			runes := []rune(word)
			if unicode.IsUpper(runes[0]) && !isEvalCommonWord(strings.ToLower(word)) {
				word = strings.Trim(word, ".,;:!?\"'()[]{}")
				if len(word) > 2 {
					terms = append(terms, word)
				}
			}
		}
	}

	return terms
}

func hasEvalRepeatedFailures(trace *models.ExecutionTrace) bool {
	if len(trace.ToolCalls) < 2 {
		return false
	}

	failedTools := make(map[string]int)
	for _, tc := range trace.ToolCalls {
		if !tc.Success {
			failedTools[tc.ToolName]++
		}
	}

	for _, count := range failedTools {
		if count >= 2 {
			return true
		}
	}
	return false
}

func truncateForEvalPrompt(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

func minFloatVal(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloatVal(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
