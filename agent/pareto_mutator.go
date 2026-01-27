package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
	openai "github.com/sashabaranov/go-openai"
)

var mutateTool = Tool{
	Name:        "mutate_result",
	Description: "Submit mutation analysis with lessons learned and improved strategy.",
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"lessons": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "New lessons learned from this attempt",
			},
			"improved_strategy": map[string]any{
				"type":        "string",
				"description": "Improved strategy prompt for the next attempt",
			},
		},
		"required":             []string{"lessons", "improved_strategy"},
		"additionalProperties": false,
	},
}

var mutateToolChoice = openai.ToolChoice{
	Type:     openai.ToolTypeFunction,
	Function: openai.ToolFunction{Name: "mutate_result"},
}

var crossoverTool = Tool{
	Name:        "crossover_result",
	Description: "Submit the merged strategy combining the best elements of both.",
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"merged_strategy": map[string]any{
				"type":        "string",
				"description": "Merged strategy combining the best elements of both",
			},
		},
		"required":             []string{"merged_strategy"},
		"additionalProperties": false,
	},
}

var crossoverToolChoice = openai.ToolChoice{
	Type:     openai.ToolTypeFunction,
	Function: openai.ToolFunction{Name: "crossover_result"},
}

type PathMutator struct {
	llm          *LLMClient
	langfuse     *langfuse.Client
	convID       string
	userID       string
	parentSpanID string
}

func NewPathMutator(llm *LLMClient, lf *langfuse.Client, convID, userID string) *PathMutator {
	return &PathMutator{llm: llm, langfuse: lf, convID: convID, userID: userID}
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

Based on what worked and what didn't, provide new lessons and an improved strategy.
The improved strategy should be specific, actionable, and address the failures observed.

Call the mutate_result tool with your analysis.`

const crossoverFallbackPrompt = `Merge these two successful strategies into one.

STRATEGY 1 (from path with scores: effectiveness={{scores1}}):
{{strategy1}}

Lessons learned:
{{lessons1}}

STRATEGY 2 (from path with scores: effectiveness={{scores2}}):
{{strategy2}}

Lessons learned:
{{lessons2}}

Create a merged strategy that combines the best elements:
- Keep what makes each strategy effective
- Resolve conflicts in favor of accuracy
- Be specific and actionable

Call the crossover_result tool with your merged strategy.`


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

	prompt := RetrievePromptTemplate("alicia/pareto/mutation-strategy", mutationStrategyFallbackPrompt, promptVars)

	response, err := MakeLLMCall(ctx, m.llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, []Tool{mutateTool}, LLMCallOptions{
		GenerationName:      "pareto.mutate",
		Prompt:              prompt,
		ToolChoice:          mutateToolChoice,
		ConvID:              m.convID,
		UserID:              m.userID,
		TraceName:           "agent:pareto",
		ParentObservationID: m.parentSpanID,
		NoRetry:             true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for mutation: %w", err)
	}

	newLessons, newStrategy, _ := parseMutateToolCall(response)

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

	prompt := RetrievePromptTemplate("alicia/pareto/mutation-crossover", crossoverFallbackPrompt, promptVars)

	response, err := MakeLLMCall(ctx, m.llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, []Tool{crossoverTool}, LLMCallOptions{
		GenerationName:      "pareto.crossover",
		Prompt:              prompt,
		ToolChoice:          crossoverToolChoice,
		ConvID:              m.convID,
		UserID:              m.userID,
		TraceName:           "agent:pareto",
		ParentObservationID: m.parentSpanID,
		NoRetry:             true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for crossover: %w", err)
	}

	mergedStrategy, _ := parseCrossoverToolCall(response)
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

func parseMutateToolCall(resp *LLMResponse) ([]string, string, error) {
	for _, tc := range resp.ToolCalls {
		if tc.Name == "mutate_result" {
			var lessons []string
			var strategy string

			if v, ok := tc.Arguments["lessons"]; ok {
				raw, err := json.Marshal(v)
				if err == nil {
					json.Unmarshal(raw, &lessons)
				}
			}
			if v, ok := tc.Arguments["improved_strategy"]; ok {
				if s, ok := v.(string); ok {
					strategy = s
				}
			}
			return lessons, strategy, nil
		}
	}
	return nil, "", fmt.Errorf("LLM did not call mutate_result tool; content: %.100s", resp.Content)
}

func parseCrossoverToolCall(resp *LLMResponse) (string, error) {
	for _, tc := range resp.ToolCalls {
		if tc.Name == "crossover_result" {
			if v, ok := tc.Arguments["merged_strategy"]; ok {
				if s, ok := v.(string); ok {
					return s, nil
				}
			}
		}
	}
	return "", fmt.Errorf("LLM did not call crossover_result tool; content: %.100s", resp.Content)
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
