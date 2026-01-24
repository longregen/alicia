package main

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
)

var (
	lessonsRegex         = regexp.MustCompile(`(?is)LESSONS_LEARNED:\s*(.*?)(?:IMPROVED_STRATEGY:|MERGED_STRATEGY:|$)`)
	improvedStrategyRegex = regexp.MustCompile(`(?is)IMPROVED_STRATEGY:\s*(.*)`)
	mergedStrategyRegex   = regexp.MustCompile(`(?is)MERGED_STRATEGY:\s*(.*)`)
)

type PathMutator struct {
	llm      *LLMClient
	langfuse *langfuse.Client
}

func NewPathMutator(llm *LLMClient, lf *langfuse.Client) *PathMutator {
	return &PathMutator{llm: llm, langfuse: lf}
}

const mutationStrategyFallbackPrompt = `Analyze this execution trace and improve the strategy.

ORIGINAL QUERY: {{query}}

STRATEGY USED:
{{strategy}}

EXECUTION TRACE:
{{trace}}

FEEDBACK: {{feedback}}

ACCUMULATED LESSONS:
{{lessons}}

Based on what worked and what didn't, provide:
1. LESSONS_LEARNED: New lessons from this attempt (bullet points, each on its own line starting with "- ")
2. IMPROVED_STRATEGY: A better strategy prompt for the next attempt

The improved strategy should be specific, actionable, and address the failures observed.

Format your response exactly like this:
LESSONS_LEARNED:
- lesson 1
- lesson 2

IMPROVED_STRATEGY:
Your improved strategy text here...`

const crossoverFallbackPrompt = `Merge these two successful strategies into one.

STRATEGY 1 (from path with scores: effectiveness={{scores1}}):
{{strategy1}}

Lessons learned:
{{lessons1}}

STRATEGY 2 (from path with scores: effectiveness={{scores2}}):
{{strategy2}}

Lessons learned:
{{lessons2}}

Create a MERGED_STRATEGY that combines the best elements:
- Keep what makes each strategy effective
- Resolve conflicts in favor of accuracy
- Be specific and actionable

Format your response exactly like this:
MERGED_STRATEGY:
Your merged strategy text here...`

func (m *PathMutator) getMutationPrompt(ctx context.Context, vars map[string]string) string {
	if m.langfuse != nil {
		prompt, err := m.langfuse.GetPromptContext(ctx, "alicia/pareto/mutation-strategy", langfuse.WithLabel("production"))
		if err == nil {
			return prompt.Compile(vars)
		}
		slog.Warn("langfuse GetPrompt failed for mutation-strategy, using fallback", "error", err)
	}
	return langfuse.CompileTemplate(mutationStrategyFallbackPrompt, vars)
}

func (m *PathMutator) getCrossoverPrompt(ctx context.Context, vars map[string]string) string {
	if m.langfuse != nil {
		prompt, err := m.langfuse.GetPromptContext(ctx, "alicia/pareto/mutation-crossover", langfuse.WithLabel("production"))
		if err == nil {
			return prompt.Compile(vars)
		}
		slog.Warn("langfuse GetPrompt failed for mutation-crossover, using fallback", "error", err)
	}
	return langfuse.CompileTemplate(crossoverFallbackPrompt, vars)
}

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

	promptVars := map[string]string{
		"query":    trace.Query,
		"strategy": candidate.StrategyPrompt,
		"trace":    formatExecutionTrace(trace),
		"feedback": feedback,
		"lessons":  lessonsStr,
	}

	prompt := m.getMutationPrompt(ctx, promptVars)

	response, err := m.llm.ChatWithOptions(ctx, []LLMMessage{{Role: "user", Content: prompt}}, nil, ChatOptions{GenerationName: "pareto.mutator"})
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for mutation: %w", err)
	}

	newLessons := parseMutationLessons(response.Content)
	newStrategy := parseMutationStrategy(response.Content)

	if newStrategy == "" {
		newStrategy = candidate.StrategyPrompt + "\n\nAdditional guidance based on previous attempt: " + feedback
	}

	allLessons := uniqueMergeLessons(candidate.AccumulatedLessons, newLessons)

	return &PathCandidate{
		ID:                 NewID("path_"),
		Generation:         candidate.Generation + 1,
		ParentIDs:          []string{candidate.ID},
		StrategyPrompt:     newStrategy,
		AccumulatedLessons: allLessons,
		CreatedAt:          time.Now(),
	}, nil
}

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

	promptVars := map[string]string{
		"strategy1": parent1.StrategyPrompt,
		"strategy2": parent2.StrategyPrompt,
		"scores1":   fmt.Sprintf("effectiveness=%.1f/5, quality=%.1f/5", parent1.Scores.Effectiveness, parent1.Scores.AnswerQuality),
		"scores2":   fmt.Sprintf("effectiveness=%.1f/5, quality=%.1f/5", parent2.Scores.Effectiveness, parent2.Scores.AnswerQuality),
		"lessons1":  lessons1,
		"lessons2":  lessons2,
	}

	prompt := m.getCrossoverPrompt(ctx, promptVars)

	response, err := m.llm.ChatWithOptions(ctx, []LLMMessage{{Role: "user", Content: prompt}}, nil, ChatOptions{GenerationName: "pareto.mutator"})
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

	return &PathCandidate{
		ID:                 NewID("path_"),
		Generation:         newGen,
		ParentIDs:          []string{parent1.ID, parent2.ID},
		StrategyPrompt:     mergedStrategy,
		AccumulatedLessons: uniqueMergeLessons(parent1.AccumulatedLessons, parent2.AccumulatedLessons),
		CreatedAt:          time.Now(),
	}, nil
}

func formatExecutionTrace(trace *ExecutionTrace) string {
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

	sb.WriteString(fmt.Sprintf("Final Answer: %s\n", langfuse.TruncateString(trace.FinalAnswer, 500, "...")))

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
	matches := improvedStrategyRegex.FindStringSubmatch(response)

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
	matches := mergedStrategyRegex.FindStringSubmatch(response)

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
