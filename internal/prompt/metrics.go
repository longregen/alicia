package prompt

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/longregen/alicia/internal/ports"
)

// Metric defines an evaluation function for prompt optimization
type Metric interface {
	// Score evaluates a prediction against gold truth
	// Returns score (0-1) and optional feedback for GEPA reflection
	Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error)
}

// Example represents a training or validation example
type Example struct {
	Inputs  map[string]any
	Outputs map[string]any
}

// Trace represents execution trace information
type Trace struct {
	Steps []TraceStep
}

// TraceStep represents a single step in the execution trace
type TraceStep struct {
	Name   string
	Inputs map[string]any
	Output any
	Error  error
}

// ScoreWithFeedback combines numeric score with textual feedback for GEPA
type ScoreWithFeedback struct {
	Score    float64
	Feedback string
}

// ExactMatchMetric checks if the prediction exactly matches the expected output
type ExactMatchMetric struct{}

func (m *ExactMatchMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
	expected := gold.Outputs["answer"]
	actual := pred.Outputs["answer"]

	if expected == actual {
		return ScoreWithFeedback{Score: 1.0, Feedback: "Correct!"}, nil
	}

	return ScoreWithFeedback{
		Score:    0.0,
		Feedback: fmt.Sprintf("Expected: %v, Got: %v", expected, actual),
	}, nil
}

// SemanticSimilarityMetric uses embeddings for soft matching
type SemanticSimilarityMetric struct {
	embedService ports.EmbeddingService
	threshold    float64
}

// NewSemanticSimilarityMetric creates a new semantic similarity metric
func NewSemanticSimilarityMetric(embedService ports.EmbeddingService, threshold float64) *SemanticSimilarityMetric {
	return &SemanticSimilarityMetric{
		embedService: embedService,
		threshold:    threshold,
	}
}

func (m *SemanticSimilarityMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
	// Placeholder implementation - requires embedding service integration
	expected, ok := gold.Outputs["answer"].(string)
	if !ok {
		return ScoreWithFeedback{}, fmt.Errorf("expected answer not a string")
	}

	actual, ok := pred.Outputs["answer"].(string)
	if !ok {
		return ScoreWithFeedback{}, fmt.Errorf("actual answer not a string")
	}

	if m.embedService == nil {
		// Fallback to simple string similarity
		similarity := simpleStringSimilarity(expected, actual)
		feedback := fmt.Sprintf(
			"String similarity: %.2f\nExpected: %s\nActual: %s",
			similarity, expected, actual,
		)
		return ScoreWithFeedback{Score: similarity, Feedback: feedback}, nil
	}

	// Get embeddings
	embeddings, err := m.embedService.EmbedBatch(ctx, []string{expected, actual})
	if err != nil {
		return ScoreWithFeedback{}, fmt.Errorf("embedding failed: %w", err)
	}

	if len(embeddings) != 2 {
		return ScoreWithFeedback{}, fmt.Errorf("expected 2 embeddings, got %d", len(embeddings))
	}

	// Calculate cosine similarity
	similarity := cosineSimilarity(embeddings[0].Embedding, embeddings[1].Embedding)

	feedback := fmt.Sprintf(
		"Semantic similarity: %.2f\nExpected: %s\nActual: %s",
		similarity, expected, actual,
	)

	return ScoreWithFeedback{Score: float64(similarity), Feedback: feedback}, nil
}

// LLMJudgeMetric uses an LLM to evaluate response quality
type LLMJudgeMetric struct {
	llmService ports.LLMService
	criteria   string
}

// NewLLMJudgeMetric creates a new LLM judge metric
func NewLLMJudgeMetric(llmService ports.LLMService, criteria string) *LLMJudgeMetric {
	return &LLMJudgeMetric{
		llmService: llmService,
		criteria:   criteria,
	}
}

func (m *LLMJudgeMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
	prompt := fmt.Sprintf(`Evaluate this response based on: %s

Question: %v
Expected Answer: %v
Actual Response: %v

Provide a score from 0.0 to 1.0 and explain your reasoning.
Format:
REASONING: ...
SCORE: X.X`,
		m.criteria,
		gold.Inputs["question"],
		gold.Outputs["answer"],
		pred.Outputs["answer"],
	)

	resp, err := m.llmService.Chat(ctx, []ports.LLMMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return ScoreWithFeedback{}, fmt.Errorf("llm judge failed: %w", err)
	}

	score, reasoning := parseJudgeResponse(resp.Content)
	return ScoreWithFeedback{Score: score, Feedback: reasoning}, nil
}

// parseJudgeResponse extracts score and reasoning from LLM response
func parseJudgeResponse(content string) (float64, string) {
	var score float64
	var reasoning string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SCORE:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "SCORE:"))
			fmt.Sscanf(scoreStr, "%f", &score)
		} else if strings.HasPrefix(line, "REASONING:") {
			reasoning = strings.TrimSpace(strings.TrimPrefix(line, "REASONING:"))
		}
	}

	return score, reasoning
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 calculates the square root of a float32
func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// simpleStringSimilarity provides a basic string similarity score
func simpleStringSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == b {
		return 1.0
	}

	// Simple Jaccard similarity on words
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	setA := make(map[string]bool)
	for _, word := range wordsA {
		setA[word] = true
	}

	setB := make(map[string]bool)
	for _, word := range wordsB {
		setB[word] = true
	}

	intersection := 0
	for word := range setA {
		if setB[word] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
