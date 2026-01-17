package baselines

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPathEvalLLMService implements ports.LLMService for testing PathEvaluator
type mockPathEvalLLMService struct {
	responses map[string]*ports.LLMResponse
	callCount int
}

func newMockPathEvalLLM() *mockPathEvalLLMService {
	return &mockPathEvalLLMService{
		responses: make(map[string]*ports.LLMResponse),
	}
}

func (m *mockPathEvalLLMService) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	m.callCount++

	// Return different responses based on prompt content
	if len(messages) > 0 {
		content := messages[0].Content

		// Check for different evaluation prompts
		if containsSubstring(content, "Rate the quality") {
			return &ports.LLMResponse{Content: "SCORE: 8 REASON: Good answer"}, nil
		}
		if containsSubstring(content, "Check if this answer contains hallucinations") {
			return &ports.LLMResponse{Content: "HALLUCINATED: false REASON: Claims are supported"}, nil
		}
		if containsSubstring(content, "specificity") {
			return &ports.LLMResponse{Content: "SPECIFICITY_SCORE: 0.9"}, nil
		}
		if containsSubstring(content, "severity") {
			return &ports.LLMResponse{Content: "SEVERITY_PENALTY: 0.2"}, nil
		}
	}

	return &ports.LLMResponse{Content: "SCORE: 5"}, nil
}

func (m *mockPathEvalLLMService) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	return m.Chat(ctx, messages)
}

func (m *mockPathEvalLLMService) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

func (m *mockPathEvalLLMService) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringSearch(s, substr))
}

func containsSubstringSearch(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewPathEvaluator(t *testing.T) {
	llm := newMockPathEvalLLM()
	evaluator := NewPathEvaluator(llm)

	require.NotNil(t, evaluator)
	assert.Equal(t, llm, evaluator.llm)
}

func TestPathEvaluator_HeuristicScreen(t *testing.T) {
	evaluator := NewPathEvaluator(newMockPathEvalLLM())

	tests := []struct {
		name        string
		trace       *prompt.ExecutionTrace
		minExpected float64
		maxExpected float64
	}{
		{
			name: "good answer with successful tools",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "The quarterly revenue was $1.5 million, representing a 15% increase from last year.",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "database_query", Success: true},
					{ToolName: "calculate", Success: true},
				},
			},
			minExpected: 0.7, // Has answer + specific data + successful tools + good length
			maxExpected: 1.0,
		},
		{
			name: "empty answer",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "",
				ToolCalls:   []prompt.ToolCallRecord{},
			},
			minExpected: 0.0,
			maxExpected: 0.3, // Only gets partial credit for no tools
		},
		{
			name: "non-answer response",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "I'm not sure how to answer this question. Unable to determine the result.",
				ToolCalls:   []prompt.ToolCallRecord{},
			},
			minExpected: 0.0,
			maxExpected: 0.6, // non-answer detection but still gets some credit for structure
		},
		{
			name: "failed tool calls",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Could not retrieve the data due to errors.",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "api_call", Success: false, Error: "timeout"},
					{ToolName: "api_call", Success: false, Error: "connection refused"},
				},
			},
			minExpected: 0.0,
			maxExpected: 0.5, // Failed calls reduce success rate
		},
		{
			name: "mixed tool success",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Found 5 matching records.",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "search", Success: true},
					{ToolName: "filter", Success: false, Error: "invalid filter"},
					{ToolName: "count", Success: true},
				},
			},
			minExpected: 0.4,
			maxExpected: 1.0, // Good answer with partial success
		},
		{
			name: "very short answer",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Yes",
				ToolCalls:   []prompt.ToolCallRecord{},
			},
			minExpected: 0.2,
			maxExpected: 0.6, // Short but present gets some credit
		},
		{
			name: "answer with specific data",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "John Smith was born on January 15, 1985. His account balance is $2,500.50.",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "user_lookup", Success: true},
				},
			},
			minExpected: 0.7, // Specific data (numbers, names, dates) + good structure
			maxExpected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := evaluator.heuristicScreen(tt.trace)
			assert.GreaterOrEqual(t, score, tt.minExpected, "score should be at least %f", tt.minExpected)
			assert.LessOrEqual(t, score, tt.maxExpected, "score should be at most %f", tt.maxExpected)
		})
	}
}

func TestContainsSpecificData(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		expected bool
	}{
		{
			name:     "empty string",
			answer:   "",
			expected: false,
		},
		{
			name:     "contains numbers",
			answer:   "The total is 42",
			expected: true,
		},
		{
			name:     "contains decimal numbers",
			answer:   "The price is 19.99",
			expected: true,
		},
		{
			name:     "contains percentage",
			answer:   "Increased by 15%",
			expected: true,
		},
		{
			name:     "contains proper nouns (names)",
			answer:   "John Smith and Mary Johnson attended the meeting",
			expected: true,
		},
		{
			name:     "contains date pattern MM/DD/YYYY",
			answer:   "The event is on 12/25/2024",
			expected: true,
		},
		{
			name:     "contains date pattern YYYY-MM-DD",
			answer:   "Updated on 2024-01-15",
			expected: true,
		},
		{
			name:     "contains month name",
			answer:   "The deadline is January 15",
			expected: true,
		},
		{
			name:     "generic text without specific data",
			answer:   "this is a generic response without any specific information",
			expected: false,
		},
		{
			name:     "only common words capitalized",
			answer:   "The answer is that it depends on the context",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSpecificData(tt.answer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountSuccessfulToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		trace    *prompt.ExecutionTrace
		expected int
	}{
		{
			name:     "no tool calls",
			trace:    &prompt.ExecutionTrace{ToolCalls: []prompt.ToolCallRecord{}},
			expected: 0,
		},
		{
			name: "all successful",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: true},
					{ToolName: "b", Success: true},
					{ToolName: "c", Success: true},
				},
			},
			expected: 3,
		},
		{
			name: "all failed",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: false},
					{ToolName: "b", Success: false},
				},
			},
			expected: 0,
		},
		{
			name: "mixed",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: true},
					{ToolName: "b", Success: false},
					{ToolName: "c", Success: true},
					{ToolName: "d", Success: false},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countSuccessfulToolCalls(tt.trace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountFailedToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		trace    *prompt.ExecutionTrace
		expected int
	}{
		{
			name:     "no tool calls",
			trace:    &prompt.ExecutionTrace{ToolCalls: []prompt.ToolCallRecord{}},
			expected: 0,
		},
		{
			name: "all successful",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: true},
					{ToolName: "b", Success: true},
				},
			},
			expected: 0,
		},
		{
			name: "all failed",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: false, Error: "error1"},
					{ToolName: "b", Success: false, Error: "error2"},
					{ToolName: "c", Success: false, Error: "error3"},
				},
			},
			expected: 3,
		},
		{
			name: "mixed",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: true},
					{ToolName: "b", Success: false},
					{ToolName: "c", Success: false},
					{ToolName: "d", Success: true},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countFailedToolCalls(tt.trace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathEvaluator_GenerateFeedback(t *testing.T) {
	evaluator := NewPathEvaluator(newMockPathEvalLLM())
	ctx := context.Background()

	tests := []struct {
		name             string
		query            string
		trace            *prompt.ExecutionTrace
		scores           prompt.PathScores
		expectedContains []string
	}{
		{
			name:  "very low answer quality",
			query: "What is the revenue?",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "I don't know",
				ToolCalls:   []prompt.ToolCallRecord{},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.2,
				Efficiency:    0.8,
				TokenCost:     0.8,
				Robustness:    0.8,
				Latency:       0.8,
			},
			expectedContains: []string{"very low"},
		},
		{
			name:  "below average answer quality",
			query: "What is the revenue?",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Around 100",
				ToolCalls:   []prompt.ToolCallRecord{{Success: true}},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.4,
				Efficiency:    0.8,
				TokenCost:     0.8,
				Robustness:    0.8,
				Latency:       0.8,
			},
			expectedContains: []string{"below average"},
		},
		{
			name:  "failed tool calls",
			query: "Fetch the data",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Could not complete",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "fetch", Success: false, Error: "timeout"},
					{ToolName: "fetch", Success: false, Error: "timeout"},
				},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.5,
				Robustness:    0.6,
			},
			expectedContains: []string{"tool call(s) failed"},
		},
		{
			name:  "high number of tool calls with low quality",
			query: "Simple query",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Result",
				ToolCalls: []prompt.ToolCallRecord{
					{Success: true}, {Success: true}, {Success: true},
					{Success: true}, {Success: true}, {Success: true},
					{Success: true}, {Success: true}, {Success: true},
				},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.6,
				Efficiency:    0.1,
			},
			expectedContains: []string{"tool calls"}, // Either "Many tool calls" or "High number of tool calls"
		},
		{
			name:  "very high token usage",
			query: "Test",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Answer",
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.8,
				TokenCost:     0.2,
			},
			expectedContains: []string{"token usage"},
		},
		{
			name:  "slow execution",
			query: "Test",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Answer",
				DurationMs:  25000,
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.8,
				Latency:       0.2,
			},
			expectedContains: []string{"slow execution"}, // Contains "Very slow execution"
		},
		{
			name:  "no answer produced",
			query: "What is X?",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "",
				ToolCalls:   []prompt.ToolCallRecord{},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.1,
			},
			expectedContains: []string{"No meaningful answer"},
		},
		{
			name:  "repeated failures",
			query: "Fetch data",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Failed",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "api_call", Success: false, Error: "error"},
					{ToolName: "api_call", Success: false, Error: "error"},
				},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.3,
				Robustness:    0.4,
			},
			expectedContains: []string{"similar failures"},
		},
		{
			name:  "successful path",
			query: "Get user info",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "User John, age 30, active since 2020",
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "user_lookup", Success: true},
				},
			},
			scores: prompt.PathScores{
				AnswerQuality: 0.9,
				Efficiency:    0.9,
				TokenCost:     0.8,
				Robustness:    0.95,
				Latency:       0.85,
			},
			expectedContains: []string{"successfully"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedback := evaluator.generateFeedback(ctx, tt.query, tt.trace, tt.scores)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, feedback, expected, "feedback should contain: %s", expected)
			}
		})
	}
}

func TestPathEvaluator_Evaluate(t *testing.T) {
	t.Run("nil trace returns error", func(t *testing.T) {
		evaluator := NewPathEvaluator(newMockPathEvalLLM())

		_, _, err := evaluator.Evaluate(context.Background(), "query", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trace cannot be nil")
	})

	t.Run("evaluates promising candidate with LLM", func(t *testing.T) {
		mockLLM := newMockPathEvalLLM()
		evaluator := NewPathEvaluator(mockLLM)

		trace := &prompt.ExecutionTrace{
			FinalAnswer: "John Smith earned $150,000 in 2024, which is a 10% increase.",
			ToolCalls: []prompt.ToolCallRecord{
				{ToolName: "database_query", Success: true, Result: "user data"},
				{ToolName: "calculate", Success: true, Result: "percentage"},
			},
			TotalTokens: 5000,
			DurationMs:  10000,
		}

		scores, feedback, err := evaluator.Evaluate(context.Background(), "What was John's salary?", trace)
		require.NoError(t, err)

		// Verify scores are in valid range
		assert.GreaterOrEqual(t, scores.AnswerQuality, 0.0)
		assert.LessOrEqual(t, scores.AnswerQuality, 1.0)
		assert.GreaterOrEqual(t, scores.Efficiency, 0.0)
		assert.LessOrEqual(t, scores.Efficiency, 1.0)
		assert.GreaterOrEqual(t, scores.TokenCost, 0.0)
		assert.LessOrEqual(t, scores.TokenCost, 1.0)
		assert.GreaterOrEqual(t, scores.Robustness, 0.0)
		assert.LessOrEqual(t, scores.Robustness, 1.0)
		assert.GreaterOrEqual(t, scores.Latency, 0.0)
		assert.LessOrEqual(t, scores.Latency, 1.0)

		// Feedback should not be empty
		assert.NotEmpty(t, feedback)

		// LLM should have been called for promising candidate
		assert.Greater(t, mockLLM.callCount, 0)
	})

	t.Run("uses heuristic only for low-quality candidate", func(t *testing.T) {
		mockLLM := newMockPathEvalLLM()
		evaluator := NewPathEvaluator(mockLLM)

		// Trace that will score low on heuristics (empty answer, failed calls)
		trace := &prompt.ExecutionTrace{
			FinalAnswer: "",
			ToolCalls: []prompt.ToolCallRecord{
				{ToolName: "api", Success: false, Error: "timeout"},
			},
			TotalTokens: 1000,
			DurationMs:  5000,
		}

		scores, _, err := evaluator.Evaluate(context.Background(), "query", trace)
		require.NoError(t, err)

		// Score should be low
		assert.Less(t, scores.AnswerQuality, 0.5)

		// LLM calls should be minimal (only for robustness check which might be skipped)
	})

	t.Run("efficiency score based on tool calls", func(t *testing.T) {
		evaluator := NewPathEvaluator(newMockPathEvalLLM())

		// Few tool calls = high efficiency
		traceFew := &prompt.ExecutionTrace{
			FinalAnswer: "Answer with specific data: $100",
			ToolCalls: []prompt.ToolCallRecord{
				{Success: true},
			},
			TotalTokens: 1000,
			DurationMs:  5000,
		}

		scoresFew, _, _ := evaluator.Evaluate(context.Background(), "query", traceFew)

		// Many tool calls = lower efficiency
		traceMany := &prompt.ExecutionTrace{
			FinalAnswer: "Answer with specific data: $100",
			ToolCalls: []prompt.ToolCallRecord{
				{Success: true}, {Success: true}, {Success: true},
				{Success: true}, {Success: true}, {Success: true},
				{Success: true}, {Success: true}, {Success: true},
			},
			TotalTokens: 1000,
			DurationMs:  5000,
		}

		scoresMany, _, _ := evaluator.Evaluate(context.Background(), "query", traceMany)

		assert.Greater(t, scoresFew.Efficiency, scoresMany.Efficiency)
	})

	t.Run("token cost score based on usage", func(t *testing.T) {
		evaluator := NewPathEvaluator(newMockPathEvalLLM())

		// Low tokens = high score
		traceLow := &prompt.ExecutionTrace{
			FinalAnswer: "Answer: 42",
			TotalTokens: 500,
			DurationMs:  5000,
		}

		scoresLow, _, _ := evaluator.Evaluate(context.Background(), "query", traceLow)

		// High tokens = lower score
		traceHigh := &prompt.ExecutionTrace{
			FinalAnswer: "Answer: 42",
			TotalTokens: 8000,
			DurationMs:  5000,
		}

		scoresHigh, _, _ := evaluator.Evaluate(context.Background(), "query", traceHigh)

		assert.Greater(t, scoresLow.TokenCost, scoresHigh.TokenCost)
	})

	t.Run("latency score based on duration", func(t *testing.T) {
		evaluator := NewPathEvaluator(newMockPathEvalLLM())

		// Fast = high score
		traceFast := &prompt.ExecutionTrace{
			FinalAnswer: "Quick answer: yes",
			TotalTokens: 1000,
			DurationMs:  2000,
		}

		scoresFast, _, _ := evaluator.Evaluate(context.Background(), "query", traceFast)

		// Slow = lower score
		traceSlow := &prompt.ExecutionTrace{
			FinalAnswer: "Quick answer: yes",
			TotalTokens: 1000,
			DurationMs:  25000,
		}

		scoresSlow, _, _ := evaluator.Evaluate(context.Background(), "query", traceSlow)

		assert.Greater(t, scoresFast.Latency, scoresSlow.Latency)
	})
}

func TestIsNonAnswer(t *testing.T) {
	tests := []struct {
		answer   string
		expected bool
	}{
		{"I don't know", true},
		{"Unable to determine the result", true},
		{"I cannot answer that question", true},
		{"I can't find the information", true},
		{"No information available", true},
		{"I'm not sure about this", true},
		{"Unable to answer at this time", true},
		{"Insufficient data to provide an answer", true},
		{"The answer is 42", false},
		{"Based on the data, revenue is $1M", false},
		{"", false}, // Empty is handled separately
		{"Here is the result you requested", false},
	}

	for _, tt := range tests {
		t.Run(tt.answer, func(t *testing.T) {
			result := isNonAnswer(tt.answer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasRepeatedFailures(t *testing.T) {
	tests := []struct {
		name     string
		trace    *prompt.ExecutionTrace
		expected bool
	}{
		{
			name: "no failures",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: true},
					{ToolName: "b", Success: true},
				},
			},
			expected: false,
		},
		{
			name: "single failure",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "a", Success: false},
					{ToolName: "b", Success: true},
				},
			},
			expected: false,
		},
		{
			name: "repeated failures same tool",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "api", Success: false},
					{ToolName: "api", Success: false},
				},
			},
			expected: true,
		},
		{
			name: "different tools failing",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "api1", Success: false},
					{ToolName: "api2", Success: false},
				},
			},
			expected: false,
		},
		{
			name: "empty trace",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{},
			},
			expected: false,
		},
		{
			name: "single call",
			trace: &prompt.ExecutionTrace{
				ToolCalls: []prompt.ToolCallRecord{
					{ToolName: "api", Success: false},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRepeatedFailures(tt.trace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseScoreFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected float64
	}{
		{
			name:     "standard format",
			response: "SCORE: 8 REASON: Good answer",
			expected: 8.0,
		},
		{
			name:     "decimal score",
			response: "SCORE: 7.5 REASON: Mostly correct",
			expected: 7.5,
		},
		{
			name:     "score at start",
			response: "8 out of 10",
			expected: 8.0,
		},
		{
			name:     "lowercase score",
			response: "score: 6",
			expected: 6.0,
		},
		{
			name:     "unparseable returns default",
			response: "The answer was good",
			expected: 5.0, // Default middle score
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseScoreFromResponse(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSpecificityScore(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected float64
	}{
		{
			name:     "standard format",
			response: "SPECIFICITY_SCORE: 0.85",
			expected: 0.85,
		},
		{
			name:     "lowercase",
			response: "specificity_score: 0.9",
			expected: 0.9,
		},
		{
			name:     "unparseable returns default",
			response: "The answer was specific enough",
			expected: 1.0, // Default to no penalty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSpecificityScore(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSeverityPenalty(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected float64
	}{
		{
			name:     "standard format",
			response: "SEVERITY_PENALTY: 0.3",
			expected: 0.3,
		},
		{
			name:     "capped at 0.5",
			response: "SEVERITY_PENALTY: 0.8",
			expected: 0.5, // Capped
		},
		{
			name:     "minimum at 0",
			response: "SEVERITY_PENALTY: 0.0",
			expected: 0.0, // Explicit zero
		},
		{
			name:     "unparseable returns default",
			response: "Errors were not critical",
			expected: 0.1, // Default modest penalty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSeverityPenalty(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMinMaxFloat(t *testing.T) {
	t.Run("minFloat", func(t *testing.T) {
		assert.Equal(t, 1.0, minFloat(1.0, 2.0))
		assert.Equal(t, 1.0, minFloat(2.0, 1.0))
		assert.Equal(t, -1.0, minFloat(-1.0, 0.0))
		assert.Equal(t, 0.5, minFloat(0.5, 0.5))
	})

	t.Run("maxFloat", func(t *testing.T) {
		assert.Equal(t, 2.0, maxFloat(1.0, 2.0))
		assert.Equal(t, 2.0, maxFloat(2.0, 1.0))
		assert.Equal(t, 0.0, maxFloat(-1.0, 0.0))
		assert.Equal(t, 0.5, maxFloat(0.5, 0.5))
	})
}

func TestTruncateForPrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "exact length unchanged",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "long string truncated with marker",
			input:    "this is a long string that needs truncation",
			maxLen:   20,
			expected: "this is a long strin...[truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateForPrompt(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountWastedToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		trace    *prompt.ExecutionTrace
		expected int
	}{
		{
			name: "no final answer means all wasted",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "",
				ToolCalls: []prompt.ToolCallRecord{
					{Success: true, Result: "data"},
					{Success: true, Result: "more data"},
				},
			},
			expected: 2,
		},
		{
			name: "failed calls not counted as wasted",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "Answer",
				ToolCalls: []prompt.ToolCallRecord{
					{Success: false, Error: "error"},
				},
			},
			expected: 0,
		},
		{
			name: "result appears in answer - not wasted",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "The user John has balance 500",
				ToolCalls: []prompt.ToolCallRecord{
					{Success: true, Result: "John"},
					{Success: true, Result: "500"},
				},
			},
			expected: 0,
		},
		{
			name: "result not in answer - wasted",
			trace: &prompt.ExecutionTrace{
				FinalAnswer: "The total is 100",
				ToolCalls: []prompt.ToolCallRecord{
					{Success: true, Result: "user data that was not used"},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countWastedToolCalls(tt.trace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractKeyTerms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // Minimum number of terms expected
	}{
		{
			name:     "numbers extracted",
			input:    "Value is 42 and 3.14",
			expected: 2,
		},
		{
			name:     "quoted strings extracted",
			input:    `Name is "John Smith" and status is "active"`,
			expected: 2,
		},
		{
			name:     "capitalized words extracted",
			input:    "User John from Seattle",
			expected: 2, // John, Seattle
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := extractKeyTerms(tt.input)
			assert.GreaterOrEqual(t, len(terms), tt.expected)
		})
	}
}

func TestIsCommonWord(t *testing.T) {
	commonWords := []string{"the", "a", "is", "are", "was", "have", "has", "do", "did", "will", "would", "could", "should", "i", "you", "he", "she", "it", "we", "they", "and", "but", "or", "if", "however"}

	for _, word := range commonWords {
		t.Run(word, func(t *testing.T) {
			assert.True(t, isCommonWord(word), "%s should be a common word", word)
		})
	}

	notCommonWords := []string{"john", "seattle", "revenue", "database", "algorithm"}
	for _, word := range notCommonWords {
		t.Run(word, func(t *testing.T) {
			assert.False(t, isCommonWord(word), "%s should not be a common word", word)
		})
	}
}
