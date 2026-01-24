package langfuse

import (
	"context"
	"fmt"
)

// Standard score configuration definitions for Alicia.
// All scores use consistent naming conventions with namespace prefixes.

// Pareto evaluation scores (NUMERIC, 1-5 scale)
var paretoScoreConfigs = []ScoreConfig{
	{
		Name:        "pareto/effectiveness",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "How effectively the response answers the user's question (1=failure, 5=excellent)",
	},
	{
		Name:        "pareto/quality",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Quality of the answer's content and presentation (1=terrible, 5=excellent)",
	},
	{
		Name:        "pareto/hallucination",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Factual accuracy based on tool outputs (1=severe hallucination, 5=fully accurate)",
	},
	{
		Name:        "pareto/specificity",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Whether the answer's level of detail matches the question (1=wrong level, 5=perfect match)",
	},
	{
		Name:        "pareto/token_cost",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Token efficiency score (1=very expensive, 5=very efficient)",
	},
	{
		Name:        "pareto/latency",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Response time score (1=very slow, 5=very fast)",
	},
	{
		Name:        "pareto/weighted_sum",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Weighted sum of all Pareto dimensions",
	},
}

// Memory extraction scores (NUMERIC, 1-5 scale)
var memoryScoreConfigs = []ScoreConfig{
	{
		Name:        "memory/importance",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Overall importance of the extracted memory (1=trivial, 5=critical)",
	},
	{
		Name:        "memory/historical",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Historical/contextual significance (1=none, 5=highly significant)",
	},
	{
		Name:        "memory/personal",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Personal relevance to the user (1=impersonal, 5=deeply personal)",
	},
	{
		Name:        "memory/factual",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(1.0),
		MaxValue:    floatPtr(5.0),
		Description: "Factual accuracy and reliability (1=uncertain, 5=verified fact)",
	},
	{
		Name:        "memory/accepted",
		DataType:    "BOOLEAN",
		Description: "Whether the memory was accepted for storage (true=accepted, false=rejected)",
	},
}

// User feedback scores (NUMERIC, -1 to 1 scale)
var userFeedbackScoreConfigs = []ScoreConfig{
	{
		Name:        "user/message_feedback",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(-1.0),
		MaxValue:    floatPtr(1.0),
		Description: "User feedback on message quality (-1=negative, 0=neutral, 1=positive)",
	},
	{
		Name:        "user/tool_use_feedback",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(-1.0),
		MaxValue:    floatPtr(1.0),
		Description: "User feedback on tool use quality (-1=negative, 0=neutral, 1=positive)",
	},
	{
		Name:        "user/memory_use_feedback",
		DataType:    "NUMERIC",
		MinValue:    floatPtr(-1.0),
		MaxValue:    floatPtr(1.0),
		Description: "User feedback on memory retrieval quality (-1=negative, 0=neutral, 1=positive)",
	},
}

// SetupScoreConfigs creates all standard Alicia score configurations in Langfuse.
// This function is idempotent - running it multiple times will not create duplicates.
func SetupScoreConfigs(ctx context.Context, client *Client) error {
	if client == nil {
		return fmt.Errorf("langfuse client is nil")
	}

	allConfigs := make([]ScoreConfig, 0, len(paretoScoreConfigs)+len(memoryScoreConfigs)+len(userFeedbackScoreConfigs))
	allConfigs = append(allConfigs, paretoScoreConfigs...)
	allConfigs = append(allConfigs, memoryScoreConfigs...)
	allConfigs = append(allConfigs, userFeedbackScoreConfigs...)

	var errors []error
	created := 0

	for _, cfg := range allConfigs {
		err := client.CreateScoreConfig(ctx, cfg)
		if err != nil {
			client.log.Printf("langfuse: failed to create score config %q: %v", cfg.Name, err)
			errors = append(errors, fmt.Errorf("failed to create %q: %w", cfg.Name, err))
		} else {
			created++
		}
	}

	client.log.Printf("langfuse: score config setup complete: %d created/verified, %d errors",
		created, len(errors))

	if len(errors) > 0 {
		return fmt.Errorf("failed to create %d score configs", len(errors))
	}

	return nil
}

// GetAllScoreConfigNames returns the names of all standard Alicia score configurations.
func GetAllScoreConfigNames() []string {
	allConfigs := make([]ScoreConfig, 0, len(paretoScoreConfigs)+len(memoryScoreConfigs)+len(userFeedbackScoreConfigs))
	allConfigs = append(allConfigs, paretoScoreConfigs...)
	allConfigs = append(allConfigs, memoryScoreConfigs...)
	allConfigs = append(allConfigs, userFeedbackScoreConfigs...)

	names := make([]string, len(allConfigs))
	for i, cfg := range allConfigs {
		names[i] = cfg.Name
	}
	return names
}

// floatPtr returns a pointer to the given float64 value.
func floatPtr(v float64) *float64 {
	return &v
}
