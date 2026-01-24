package langfuse

import (
	"context"
	"strings"
)

// Standard Alicia evaluator names
const (
	EvaluatorEffectiveness = "alicia-effectiveness"
	EvaluatorQuality       = "alicia-quality"
	EvaluatorHallucination = "alicia-hallucination"
)

// AliciaEvaluatorConfigs returns the standard evaluator configurations for Alicia.
// These evaluators use LLM-as-a-judge for post-hoc evaluation of traces.
func AliciaEvaluatorConfigs() []EvaluatorConfig {
	return []EvaluatorConfig{
		{
			Name:          EvaluatorEffectiveness,
			Description:   "Evaluates how effectively the response answers the user's question (1-5 stars)",
			EvaluatorType: "llm",
			Model:         "gpt-4",
			Template: `Rate how effectively this response answers the user's question.

QUESTION: {{input}}

RESPONSE: {{output}}

Rate on a 1-5 scale:
1: Complete failure - no answer, error, or completely wrong topic
2: Poor - attempted but missed the point or gave unusable answer
3: Partial - addressed the question but incomplete or partially wrong
4: Good - answered the question with minor issues
5: Excellent - fully and correctly answered the question

Output format: STARS: [1-5]`,
			VariableMapping: map[string]string{
				"input":  "$.input",  // JSONPath to trace input
				"output": "$.output", // JSONPath to trace output
			},
			TargetFilter: map[string]any{
				"tags": []string{"alicia-response"},
			},
			Sampling:  0.1, // 10% of traces to manage costs
			ScoreName: "alicia/effectiveness",
		},
		{
			Name:          EvaluatorQuality,
			Description:   "Evaluates the quality of the answer's content and presentation (1-5 stars)",
			EvaluatorType: "llm",
			Model:         "gpt-4",
			Template: `Rate the quality of this answer's content and presentation.

QUESTION: {{input}}

ANSWER: {{output}}

Rate on a 1-5 scale:
1: Terrible - incoherent, unhelpful, or harmful
2: Poor - hard to understand, poorly organized, or mostly unhelpful
3: Acceptable - understandable but could be clearer or more helpful
4: Good - clear, well-organized, and helpful
5: Excellent - exceptionally clear, insightful, and perfectly addresses the need

Output format: STARS: [1-5]`,
			VariableMapping: map[string]string{
				"input":  "$.input",
				"output": "$.output",
			},
			TargetFilter: map[string]any{
				"tags": []string{"alicia-response"},
			},
			Sampling:  0.1,
			ScoreName: "alicia/quality",
		},
		{
			Name:          EvaluatorHallucination,
			Description:   "Evaluates factual accuracy - checks for hallucinations in the response (1-5 stars)",
			EvaluatorType: "llm",
			Model:         "gpt-4",
			Template: `Rate the factual accuracy of this answer. Check if claims are supported by the conversation context.

CONTEXT/INPUT: {{input}}

ANSWER: {{output}}

Rate on a 1-5 scale:
1: Severe hallucination - makes up facts, contradicts context
2: Significant hallucination - several unsupported claims or embellishments
3: Some hallucination - a few minor unsupported details
4: Mostly accurate - reasonable inferences, no major fabrications
5: Fully accurate - all claims supported by context

Output format: STARS: [1-5]`,
			VariableMapping: map[string]string{
				"input":  "$.input",
				"output": "$.output",
			},
			TargetFilter: map[string]any{
				"tags": []string{"alicia-response"},
			},
			Sampling:  0.1,
			ScoreName: "alicia/hallucination",
		},
	}
}

// SetupAliciaEvaluators creates the standard Alicia evaluators in Langfuse if they don't exist.
// This function is idempotent - it's safe to call multiple times.
// It returns nil on success, or an error if setup failed completely.
// Individual evaluator creation failures are logged but don't cause the function to fail.
func SetupAliciaEvaluators(ctx context.Context, client *Client) error {
	if client == nil {
		return nil
	}

	configs := AliciaEvaluatorConfigs()

	var successCount, skipCount, failCount int

	for _, cfg := range configs {
		// Check if evaluator already exists
		exists, err := client.EvaluatorExists(ctx, cfg.Name)
		if err != nil {
			// If the API is not available, log and continue
			if strings.Contains(err.Error(), "404") {
				client.log.Printf("langfuse: evaluator API not available, skipping evaluator setup")
				return nil // Not an error - API just isn't available
			}
			client.log.Printf("langfuse: failed to check if evaluator %q exists: %v", cfg.Name, err)
			failCount++
			continue
		}

		if exists {
			client.log.Printf("langfuse: evaluator %q already exists, skipping", cfg.Name)
			skipCount++
			continue
		}

		// Create the evaluator
		if err := client.CreateEvaluator(ctx, cfg); err != nil {
			// Check if it's just because the API doesn't exist
			if strings.Contains(err.Error(), "404") {
				client.log.Printf("langfuse: evaluator API not available, skipping evaluator setup")
				return nil
			}
			client.log.Printf("langfuse: failed to create evaluator %q: %v", cfg.Name, err)
			failCount++
			continue
		}

		successCount++
	}

	client.log.Printf("langfuse: evaluator setup complete - created: %d, skipped: %d, failed: %d",
		successCount, skipCount, failCount)

	return nil
}

// SetupAliciaScoreConfigs creates score configurations in Langfuse for Alicia evaluation scores.
// This is a wrapper around SetupScoreConfigs for backward compatibility.
// Deprecated: Use SetupScoreConfigs from score_configs.go instead.
func SetupAliciaScoreConfigs(ctx context.Context, client *Client) error {
	return SetupScoreConfigs(ctx, client)
}

// SetupAll runs Alicia Langfuse setup functions for score configs and evaluators.
// This is the main entry point for initializing Langfuse resources on startup.
// Note: For dataset initialization, call EnsureGoldenDataset separately with a context.
func SetupAll(ctx context.Context, client *Client) error {
	if client == nil {
		return nil
	}

	// Setup score configurations first (these are more likely to succeed)
	if err := SetupAliciaScoreConfigs(ctx, client); err != nil {
		client.log.Printf("langfuse: score config setup failed: %v", err)
		// Continue with evaluator setup even if this fails
	}

	// Setup managed evaluators (API may not be available)
	if err := SetupAliciaEvaluators(ctx, client); err != nil {
		client.log.Printf("langfuse: evaluator setup failed: %v", err)
		// This is not fatal - evaluators are optional
	}

	return nil
}
