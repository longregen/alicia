package prompt

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/longregen/alicia/internal/ports"
)

// PathCandidate represents an execution path being explored.
// The "gene" is the strategy text, NOT numerical parameters.
type PathCandidate struct {
	ID                 string          `json:"id"`
	RunID              string          `json:"run_id"`
	Generation         int             `json:"generation"`
	ParentIDs          []string        `json:"parent_ids"`
	StrategyPrompt     string          `json:"strategy_prompt"`
	AccumulatedLessons []string        `json:"accumulated_lessons"`
	Trace              *ExecutionTrace `json:"trace,omitempty"`
	Scores             PathScores      `json:"scores"`
	CreatedAt          time.Time       `json:"created_at"`
}

// PathScores holds multi-objective scores for Pareto selection (5 dimensions).
type PathScores struct {
	AnswerQuality float64 `json:"answer_quality"` // Primary: correctness + no hallucinations
	Efficiency    float64 `json:"efficiency"`     // Fewer tool calls = better
	TokenCost     float64 `json:"token_cost"`     // Lower token usage = better
	Robustness    float64 `json:"robustness"`     // Error handling + self-correction
	Latency       float64 `json:"latency"`        // Time to answer (inverted: fast = high)
}

// ExecutionTrace captures what happened during one path attempt.
type ExecutionTrace struct {
	Query          string           `json:"query"`
	ToolCalls      []ToolCallRecord `json:"tool_calls"`
	ReasoningSteps []string         `json:"reasoning_steps"`
	FinalAnswer    string           `json:"final_answer"`
	TotalTokens    int              `json:"total_tokens"`
	DurationMs     int64            `json:"duration_ms"`
}

// ToolCallRecord captures a single tool invocation.
type ToolCallRecord struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
	Result    any            `json:"result"`
	Success   bool           `json:"success"`
	Error     string         `json:"error,omitempty"`
}

// PathMutator mutates path candidates using LLM reflection.
type PathMutator struct {
	llm           ports.LLMService
	reflectionLLM ports.LLMService // Optional stronger model for reflection; defaults to llm
}

// NewPathMutator creates a new PathMutator.
// If reflectionLLM is nil, llm is used for reflection as well.
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
func (m *PathMutator) MutateStrategy(ctx context.Context, candidate *PathCandidate, trace *ExecutionTrace, feedback string) (*PathCandidate, error) {
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
		formatTrace(trace),
		feedback,
		lessonsStr,
	)

	response, err := m.reflectionLLM.Chat(ctx, []ports.LLMMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for mutation: %w", err)
	}

	// Parse new lessons and strategy from response
	newLessons := parseLessons(response.Content)
	newStrategy := parseStrategy(response.Content)

	// If we couldn't parse a new strategy, fall back to original with minor modification
	if newStrategy == "" {
		newStrategy = candidate.StrategyPrompt + "\n\nAdditional guidance based on previous attempt: " + feedback
	}

	// Combine accumulated lessons, avoiding duplicates
	allLessons := uniqueMerge(candidate.AccumulatedLessons, newLessons)

	return &PathCandidate{
		ID:                 generatePathID(),
		RunID:              candidate.RunID,
		Generation:         candidate.Generation + 1,
		ParentIDs:          []string{candidate.ID},
		StrategyPrompt:     newStrategy,
		AccumulatedLessons: allLessons,
		CreatedAt:          time.Now(),
	}, nil
}

// Crossover merges strategies from two Pareto-optimal paths.
func (m *PathMutator) Crossover(ctx context.Context, parent1, parent2 *PathCandidate) (*PathCandidate, error) {
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

	// Parse the merged strategy
	mergedStrategy := parseMergedStrategy(response.Content)
	if mergedStrategy == "" {
		// Fallback: concatenate strategies
		mergedStrategy = fmt.Sprintf("Combined approach:\n\nFrom strategy 1:\n%s\n\nFrom strategy 2:\n%s",
			parent1.StrategyPrompt, parent2.StrategyPrompt)
	}

	// Determine new generation
	newGen := parent1.Generation
	if parent2.Generation > newGen {
		newGen = parent2.Generation
	}
	newGen++

	return &PathCandidate{
		ID:                 generatePathID(),
		RunID:              parent1.RunID,
		Generation:         newGen,
		ParentIDs:          []string{parent1.ID, parent2.ID},
		StrategyPrompt:     mergedStrategy,
		AccumulatedLessons: uniqueMerge(parent1.AccumulatedLessons, parent2.AccumulatedLessons),
		CreatedAt:          time.Now(),
	}, nil
}

// parseLessons extracts lessons from the LLM response.
func parseLessons(response string) []string {
	// Look for LESSONS_LEARNED: section
	lessonsRegex := regexp.MustCompile(`(?is)LESSONS_LEARNED:\s*(.*?)(?:IMPROVED_STRATEGY:|MERGED_STRATEGY:|$)`)
	matches := lessonsRegex.FindStringSubmatch(response)

	var lessons []string
	if len(matches) > 1 {
		lessonsSection := strings.TrimSpace(matches[1])
		lines := strings.Split(lessonsSection, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Remove bullet points and dashes
			line = strings.TrimPrefix(line, "-")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimPrefix(line, "•")
			line = strings.TrimSpace(line)
			if line != "" && len(line) > 3 { // Skip very short or empty lines
				lessons = append(lessons, line)
			}
		}
	}
	return lessons
}

// parseStrategy extracts the improved strategy from the LLM response.
func parseStrategy(response string) string {
	// Look for IMPROVED_STRATEGY: section
	strategyRegex := regexp.MustCompile(`(?is)IMPROVED_STRATEGY:\s*(.*)`)
	matches := strategyRegex.FindStringSubmatch(response)

	if len(matches) > 1 {
		strategy := strings.TrimSpace(matches[1])
		// Clean up any trailing sections that might have been captured
		if idx := strings.Index(strings.ToUpper(strategy), "LESSONS_LEARNED:"); idx > 0 {
			strategy = strings.TrimSpace(strategy[:idx])
		}
		return strategy
	}
	return ""
}

// parseMergedStrategy extracts the merged strategy from the LLM response.
func parseMergedStrategy(response string) string {
	// Look for MERGED_STRATEGY: section
	strategyRegex := regexp.MustCompile(`(?is)MERGED_STRATEGY:\s*(.*)`)
	matches := strategyRegex.FindStringSubmatch(response)

	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: try IMPROVED_STRATEGY
	return parseStrategy(response)
}

// formatTrace formats an execution trace for LLM consumption.
func formatTrace(trace *ExecutionTrace) string {
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
			sb.WriteString(fmt.Sprintf("  %d. %s(%v) -> %s\n", i+1, tc.ToolName, formatArgs(tc.Arguments), status))
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

	sb.WriteString(fmt.Sprintf("Final Answer: %s\n", truncateString(trace.FinalAnswer, 500)))

	return sb.String()
}

// formatArgs formats tool arguments for display.
func formatArgs(args map[string]any) string {
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

// uniqueMerge combines two string slices, removing duplicates.
func uniqueMerge(a, b []string) []string {
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

// generatePathID generates a unique path candidate ID.
func generatePathID() string {
	return "path_" + uuid.New().String()[:8]
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
