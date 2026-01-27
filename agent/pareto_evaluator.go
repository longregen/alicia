package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
	openai "github.com/sashabaranov/go-openai"
	oteltrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

// scoreTool is the single tool used to force structured score output from LLM judges.
var scoreTool = Tool{
	Name:        "score",
	Description: "Submit your rating score.",
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stars": map[string]any{
				"type":        "number",
				"description": "Rating from 1 to 5",
				"minimum":     1,
				"maximum":     5,
			},
		},
		"required":             []string{"stars"},
		"additionalProperties": false,
	},
}

// scoreToolChoice forces the LLM to call the score tool.
var scoreToolChoice = openai.ToolChoice{
	Type:     openai.ToolTypeFunction,
	Function: openai.ToolFunction{Name: "score"},
}

// PathEvaluator evaluates execution paths across multiple dimensions
type PathEvaluator struct {
	llm      *LLMClient
	langfuse *langfuse.Client
	convID   string
	userID   string
}

func NewPathEvaluator(llm *LLMClient, lf *langfuse.Client, convID, userID string) *PathEvaluator {
	return &PathEvaluator{llm: llm, langfuse: lf, convID: convID, userID: userID}
}

// Evaluate scores a path across 6 Pareto dimensions (all 1-5 star ratings)
func (e *PathEvaluator) Evaluate(ctx context.Context, query string, trace *ExecutionTrace, parentSpanID string) (PathScores, string, error) {
	if trace == nil {
		return PathScores{}, "", fmt.Errorf("trace cannot be nil")
	}

	var scores PathScores
	g, gCtx := errgroup.WithContext(ctx)

	evalWithRetry := func(name string, fn func(context.Context) (float64, error), dest *float64) {
		g.Go(func() error {
			s, err := fn(gCtx)
			if err != nil {
				s, err = fn(gCtx)
			}
			if err != nil {
				return fmt.Errorf("%s eval: %w", name, err)
			}
			*dest = s
			return nil
		})
	}

	evalWithRetry("effectiveness", func(c context.Context) (float64, error) {
		return e.llmJudgeEffectiveness(c, query, trace, parentSpanID)
	}, &scores.Effectiveness)
	evalWithRetry("quality", func(c context.Context) (float64, error) {
		return e.llmJudgeAnswerQuality(c, query, trace, parentSpanID)
	}, &scores.AnswerQuality)
	evalWithRetry("hallucination", func(c context.Context) (float64, error) {
		return e.llmJudgeHallucination(c, trace, parentSpanID)
	}, &scores.Hallucination)
	evalWithRetry("specificity", func(c context.Context) (float64, error) {
		return e.llmJudgeSpecificity(c, query, trace, parentSpanID)
	}, &scores.Specificity)

	if err := g.Wait(); err != nil {
		slog.ErrorContext(ctx, "pareto evaluation failed after retry", "error", err, "query", query)
		return PathScores{}, "", err
	}

	// Token cost: exponential decay (5 stars at 0 tokens, approaches 1 star asymptotically)
	// Using formula: 5 * exp(-tokens / 5000) + 1 * (1 - exp(-tokens / 5000))
	// At 0 tokens: 5 stars, at ~11500 tokens: ~2 stars, at ~23000 tokens: ~1.1 stars
	tokenRatio := float64(trace.TotalTokens) / 5000.0
	scores.TokenCost = 4.0*math.Exp(-tokenRatio) + 1.0

	// Latency: exponential decay (5 stars at 0ms, 1 star at 120000ms/2min)
	// Using formula: 5 * exp(-ms / 30000) + 1 * (1 - exp(-ms / 30000))
	// At 0ms: 5 stars, at 30s: ~2.5 stars, at 2min: ~1.1 stars
	latencyRatio := float64(trace.DurationMs) / 30000.0
	scores.Latency = 4.0*math.Exp(-latencyRatio) + 1.0

	scores.Effectiveness = clampStars(scores.Effectiveness)
	scores.AnswerQuality = clampStars(scores.AnswerQuality)
	scores.Hallucination = clampStars(scores.Hallucination)
	scores.Specificity = clampStars(scores.Specificity)
	scores.TokenCost = clampStars(scores.TokenCost)
	scores.Latency = clampStars(scores.Latency)

	feedback := e.generateFeedback(query, trace, scores)
	go e.sendScoresToLangfuse(ctx, scores)

	return scores, feedback, nil
}

func (e *PathEvaluator) sendScoresToLangfuse(ctx context.Context, scores PathScores) {
	if e.langfuse == nil {
		return
	}

	span := oteltrace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		slog.WarnContext(ctx, "pareto evaluation: no valid trace context, skipping langfuse score ingestion")
		return
	}
	traceID := span.SpanContext().TraceID().String()

	weights := DefaultPathScoreWeights()
	weightedSum := scores.WeightedSum(weights)

	lfScores := []langfuse.ScoreParams{
		{TraceID: traceID, Name: "pareto/effectiveness", Value: scores.Effectiveness, DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "pareto/quality", Value: scores.AnswerQuality, DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "pareto/hallucination", Value: scores.Hallucination, DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "pareto/specificity", Value: scores.Specificity, DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "pareto/token_cost", Value: scores.TokenCost, DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "pareto/latency", Value: scores.Latency, DataType: langfuse.ScoreDataTypeNumeric},
		{TraceID: traceID, Name: "pareto/weighted_sum", Value: weightedSum, DataType: langfuse.ScoreDataTypeNumeric},
	}

	// Use a detached context with timeout so we don't block on the main request
	sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.langfuse.CreateScoreBatch(sendCtx, lfScores); err != nil {
		slog.ErrorContext(ctx, "failed to send pareto scores to langfuse", "error", err)
	} else {
		slog.InfoContext(ctx, "sent pareto scores to langfuse", "trace_id", traceID)
	}
}

func clampStars(score float64) float64 {
	if score < 1.0 {
		return 1.0
	}
	if score > 5.0 {
		return 5.0
	}
	return score
}

func (e *PathEvaluator) llmJudgeEffectiveness(ctx context.Context, query string, trace *ExecutionTrace, parentSpanID string) (float64, error) {
	fallbackPrompt := `Rate how effectively this response answers the user's question.

QUESTION: {{question}}

RESPONSE: {{response}}

Rate on a 1-5 scale:
1: Complete failure - no answer, error, or completely wrong topic
2: Poor - attempted but missed the point or gave unusable answer
3: Partial - addressed the question but incomplete or partially wrong
4: Good - answered the question with minor issues
5: Excellent - fully and correctly answered the question

Call the score tool with your rating.`

	vars := map[string]string{
		"question": query,
		"response": trace.FinalAnswer,
	}

	prompt := RetrievePromptTemplate("alicia/pareto/eval-effectiveness", fallbackPrompt, vars)

	resp, err := MakeLLMCall(ctx, e.llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, []Tool{scoreTool}, LLMCallOptions{
		GenerationName:      "pareto.eval.effectiveness",
		Prompt:              prompt,
		ToolChoice:          scoreToolChoice,
		ConvID:              e.convID,
		UserID:              e.userID,
		TraceName:           "agent:pareto",
		ParentObservationID: parentSpanID,
		InputOverride:       map[string]any{"question": query, "answer": trace.FinalAnswer},
		NoRetry:             true,
	})
	if err != nil {
		return 0, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseScoreToolCall(resp)
}

func (e *PathEvaluator) llmJudgeAnswerQuality(ctx context.Context, query string, trace *ExecutionTrace, parentSpanID string) (float64, error) {
	fallbackPrompt := `Rate the quality of this answer's content and presentation.

QUESTION: {{question}}

ANSWER: {{answer}}

Rate on a 1-5 scale:
1: Terrible - incoherent, unhelpful, or harmful
2: Poor - hard to understand, poorly organized, or mostly unhelpful
3: Acceptable - understandable but could be clearer or more helpful
4: Good - clear, well-organized, and helpful
5: Excellent - exceptionally clear, insightful, and perfectly addresses the need

Call the score tool with your rating.`

	vars := map[string]string{
		"question": query,
		"answer":   trace.FinalAnswer,
	}

	prompt := RetrievePromptTemplate("alicia/pareto/eval-quality", fallbackPrompt, vars)

	resp, err := MakeLLMCall(ctx, e.llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, []Tool{scoreTool}, LLMCallOptions{
		GenerationName:      "pareto.eval.quality",
		Prompt:              prompt,
		ToolChoice:          scoreToolChoice,
		ConvID:              e.convID,
		UserID:              e.userID,
		TraceName:           "agent:pareto",
		ParentObservationID: parentSpanID,
		InputOverride:       map[string]any{"question": query, "answer": trace.FinalAnswer},
		NoRetry:             true,
	})
	if err != nil {
		return 0, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseScoreToolCall(resp)
}

func (e *PathEvaluator) llmJudgeHallucination(ctx context.Context, trace *ExecutionTrace, parentSpanID string) (float64, error) {
	toolOutputs := formatToolOutputs(trace.ToolCalls)

	// If no tools were used, we can't check for hallucinations against tool outputs
	if toolOutputs == "" {
		return 4.0, nil
	}

	fallbackPrompt := `Rate the factual accuracy of this answer based on the tool outputs.

TOOL OUTPUTS (the only source of truth):
{{tool_outputs}}

ANSWER:
{{answer}}

Rate on a 1-5 scale:
1: Severe hallucination - makes up facts not in tool outputs, contradicts data
2: Significant hallucination - several unsupported claims or embellishments
3: Some hallucination - a few minor unsupported details
4: Mostly accurate - reasonable inferences, no major fabrications
5: Fully accurate - all claims supported by tool outputs

Call the score tool with your rating.`

	vars := map[string]string{
		"tool_outputs": toolOutputs,
		"answer":       trace.FinalAnswer,
	}

	prompt := RetrievePromptTemplate("alicia/pareto/eval-hallucination", fallbackPrompt, vars)

	resp, err := MakeLLMCall(ctx, e.llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, []Tool{scoreTool}, LLMCallOptions{
		GenerationName:      "pareto.eval.hallucination",
		Prompt:              prompt,
		ToolChoice:          scoreToolChoice,
		ConvID:              e.convID,
		UserID:              e.userID,
		TraceName:           "agent:pareto",
		ParentObservationID: parentSpanID,
		InputOverride:       map[string]any{"tool_outputs": toolOutputs, "answer": trace.FinalAnswer},
		NoRetry:             true,
	})
	if err != nil {
		return 0, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseScoreToolCall(resp)
}

func (e *PathEvaluator) llmJudgeSpecificity(ctx context.Context, query string, trace *ExecutionTrace, parentSpanID string) (float64, error) {
	fallbackPrompt := `Rate whether the answer's level of detail matches what the question needs.

QUESTION: {{question}}

ANSWER: {{answer}}

Rate on a 1-5 scale:
1: Completely wrong level - way too vague for specific question, or overwhelming detail for simple question
2: Poor match - noticeably too vague or too detailed
3: Acceptable - somewhat appropriate but could be better calibrated
4: Good match - appropriate level of detail for the question type
5: Perfect match - exactly the right amount of detail and depth

Call the score tool with your rating.`

	vars := map[string]string{
		"question": query,
		"answer":   trace.FinalAnswer,
	}

	prompt := RetrievePromptTemplate("alicia/pareto/eval-specificity", fallbackPrompt, vars)

	resp, err := MakeLLMCall(ctx, e.llm, []LLMMessage{{Role: "user", Content: prompt.Text}}, []Tool{scoreTool}, LLMCallOptions{
		GenerationName:      "pareto.eval.specificity",
		Prompt:              prompt,
		ToolChoice:          scoreToolChoice,
		ConvID:              e.convID,
		UserID:              e.userID,
		TraceName:           "agent:pareto",
		ParentObservationID: parentSpanID,
		InputOverride:       map[string]any{"question": query, "answer": trace.FinalAnswer},
		NoRetry:             true,
	})
	if err != nil {
		return 0, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseScoreToolCall(resp)
}

func (e *PathEvaluator) generateFeedback(query string, trace *ExecutionTrace, scores PathScores) string {
	var feedbackParts []string

	if scores.Effectiveness < 2.5 {
		feedbackParts = append(feedbackParts, "Failed to answer the question - need a completely different approach.")
	} else if scores.Effectiveness < 3.5 {
		feedbackParts = append(feedbackParts, "Partially answered - need to address the question more directly.")
	}

	if scores.AnswerQuality < 2.5 {
		feedbackParts = append(feedbackParts, "Answer quality is poor - improve clarity and organization.")
	} else if scores.AnswerQuality < 3.5 {
		feedbackParts = append(feedbackParts, "Answer quality is mediocre - could be clearer or more helpful.")
	}

	if scores.Hallucination < 2.5 {
		feedbackParts = append(feedbackParts, "Significant hallucination detected - stick strictly to tool output facts.")
	} else if scores.Hallucination < 3.5 {
		feedbackParts = append(feedbackParts, "Some unsupported claims - be more careful about factual accuracy.")
	}

	if scores.Specificity < 2.5 {
		feedbackParts = append(feedbackParts, "Wrong level of detail - adjust specificity to match the question.")
	} else if scores.Specificity < 3.5 {
		feedbackParts = append(feedbackParts, "Detail level could be better calibrated to the question.")
	}

	if scores.TokenCost < 2.0 {
		feedbackParts = append(feedbackParts, "Very high token usage - be more concise.")
	}

	if scores.Latency < 2.0 {
		feedbackParts = append(feedbackParts, "Very slow execution - find a more direct approach.")
	}

	if len(feedbackParts) == 0 {
		return "Good execution with solid results."
	}

	return strings.Join(feedbackParts, " ")
}

func formatToolOutputs(toolCalls []ToolCallRecord) string {
	var sb strings.Builder

	for i, tc := range toolCalls {
		if tc.Success && tc.Result != nil {
			resultStr := fmt.Sprintf("%v", tc.Result)
			sb.WriteString(fmt.Sprintf("[Tool %d: %s]\n%s\n\n", i+1, tc.ToolName, resultStr))
		}
	}

	return sb.String()
}

func parseScoreToolCall(resp *LLMResponse) (float64, error) {
	for _, tc := range resp.ToolCalls {
		if tc.Name == "score" {
			if v, ok := tc.Arguments["stars"]; ok {
				switch n := v.(type) {
				case float64:
					return clampStars(n), nil
				case int:
					return clampStars(float64(n)), nil
				}
			}
		}
	}
	return 0, fmt.Errorf("LLM did not call score tool; content: %.100s", resp.Content)
}

