package prompt

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMService implements ports.LLMService for testing
type mockLLMService struct {
	chatResponse *ports.LLMResponse
	chatError    error
}

func (m *mockLLMService) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	return m.chatResponse, m.chatError
}

func (m *mockLLMService) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	return m.chatResponse, m.chatError
}

func (m *mockLLMService) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

func (m *mockLLMService) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

func TestNewPathMutator(t *testing.T) {
	t.Run("with both LLMs provided", func(t *testing.T) {
		mainLLM := &mockLLMService{}
		reflectionLLM := &mockLLMService{}

		mutator := NewPathMutator(mainLLM, reflectionLLM)
		require.NotNil(t, mutator)
		assert.Equal(t, mainLLM, mutator.llm)
		assert.Equal(t, reflectionLLM, mutator.reflectionLLM)
	})

	t.Run("with nil reflection LLM uses main LLM", func(t *testing.T) {
		mainLLM := &mockLLMService{}

		mutator := NewPathMutator(mainLLM, nil)
		require.NotNil(t, mutator)
		assert.Equal(t, mainLLM, mutator.llm)
		assert.Equal(t, mainLLM, mutator.reflectionLLM, "should use main LLM for reflection when not provided")
	})
}

func TestParseLessons(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected []string
	}{
		{
			name: "standard lessons format",
			response: `LESSONS_LEARNED:
- Always verify tool outputs before using them
- Consider edge cases in queries
- Break complex queries into smaller steps

IMPROVED_STRATEGY:
New strategy here...`,
			expected: []string{
				"Always verify tool outputs before using them",
				"Consider edge cases in queries",
				"Break complex queries into smaller steps",
			},
		},
		{
			name: "lessons with bullet points",
			response: `LESSONS_LEARNED:
* First lesson here
* Second lesson here

IMPROVED_STRATEGY:
Strategy text`,
			expected: []string{
				"First lesson here",
				"Second lesson here",
			},
		},
		{
			name: "lessons with mixed bullet styles",
			response: `LESSONS_LEARNED:
- Lesson with dash
* Lesson with asterisk
  Lesson with bullet

IMPROVED_STRATEGY:
Strategy text`,
			expected: []string{
				"Lesson with dash",
				"Lesson with asterisk",
				"Lesson with bullet",
			},
		},
		{
			name:     "no lessons section",
			response: "Just some random text without the expected format",
			expected: nil,
		},
		{
			name: "empty lessons section",
			response: `LESSONS_LEARNED:

IMPROVED_STRATEGY:
Some strategy`,
			expected: nil,
		},
		{
			name: "short lessons filtered out",
			response: `LESSONS_LEARNED:
- OK
- This is a valid lesson
- No

IMPROVED_STRATEGY:
Strategy`,
			expected: []string{
				"This is a valid lesson",
			},
		},
		{
			name: "lessons at end of response",
			response: `IMPROVED_STRATEGY:
Strategy text here

LESSONS_LEARNED:
- Final lesson one
- Final lesson two`,
			expected: []string{
				"Final lesson one",
				"Final lesson two",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lessons := parseLessons(tt.response)
			assert.Equal(t, tt.expected, lessons)
		})
	}
}

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name: "standard strategy format",
			response: `LESSONS_LEARNED:
- Some lesson

IMPROVED_STRATEGY:
This is the new and improved strategy.
It spans multiple lines.
With detailed instructions.`,
			expected: `This is the new and improved strategy.
It spans multiple lines.
With detailed instructions.`,
		},
		{
			name: "strategy only",
			response: `IMPROVED_STRATEGY:
Simple strategy text here`,
			expected: "Simple strategy text here",
		},
		{
			name:     "no strategy section",
			response: "Just random text without strategy marker",
			expected: "",
		},
		{
			name:     "empty strategy",
			response: "IMPROVED_STRATEGY:",
			expected: "",
		},
		{
			name: "strategy with trailing lessons section removed",
			response: `IMPROVED_STRATEGY:
Main strategy content here

LESSONS_LEARNED:
These should not be included`,
			expected: "Main strategy content here",
		},
		{
			name: "case insensitive matching",
			response: `improved_strategy:
Case insensitive strategy text`,
			expected: "Case insensitive strategy text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := parseStrategy(tt.response)
			assert.Equal(t, tt.expected, strategy)
		})
	}
}

func TestFormatTrace(t *testing.T) {
	t.Run("nil trace", func(t *testing.T) {
		result := formatTrace(nil)
		assert.Equal(t, "(no trace available)", result)
	})

	t.Run("empty trace", func(t *testing.T) {
		trace := &ExecutionTrace{
			Query:       "Test query",
			DurationMs:  1000,
			TotalTokens: 500,
		}

		result := formatTrace(trace)
		assert.Contains(t, result, "Query: Test query")
		assert.Contains(t, result, "Duration: 1000ms")
		assert.Contains(t, result, "Total Tokens: 500")
	})

	t.Run("trace with reasoning steps", func(t *testing.T) {
		trace := &ExecutionTrace{
			Query:          "Test query",
			DurationMs:     1000,
			TotalTokens:    500,
			ReasoningSteps: []string{"First step", "Second step", "Third step"},
		}

		result := formatTrace(trace)
		assert.Contains(t, result, "Reasoning Steps:")
		assert.Contains(t, result, "1. First step")
		assert.Contains(t, result, "2. Second step")
		assert.Contains(t, result, "3. Third step")
	})

	t.Run("trace with successful tool calls", func(t *testing.T) {
		trace := &ExecutionTrace{
			Query:       "Test query",
			DurationMs:  1000,
			TotalTokens: 500,
			ToolCalls: []ToolCallRecord{
				{
					ToolName:  "search",
					Arguments: map[string]any{"query": "test"},
					Result:    "Found 10 results",
					Success:   true,
				},
			},
		}

		result := formatTrace(trace)
		assert.Contains(t, result, "Tool Calls:")
		assert.Contains(t, result, "search")
		assert.Contains(t, result, "SUCCESS")
		assert.Contains(t, result, "Result:")
	})

	t.Run("trace with failed tool calls", func(t *testing.T) {
		trace := &ExecutionTrace{
			Query:       "Test query",
			DurationMs:  1000,
			TotalTokens: 500,
			ToolCalls: []ToolCallRecord{
				{
					ToolName:  "database_query",
					Arguments: map[string]any{"sql": "SELECT *"},
					Success:   false,
					Error:     "Connection timeout",
				},
			},
		}

		result := formatTrace(trace)
		assert.Contains(t, result, "FAILED")
		assert.Contains(t, result, "Connection timeout")
	})

	t.Run("trace with long result truncated", func(t *testing.T) {
		longResult := ""
		for i := 0; i < 300; i++ {
			longResult += "x"
		}

		trace := &ExecutionTrace{
			Query:       "Test query",
			DurationMs:  1000,
			TotalTokens: 500,
			ToolCalls: []ToolCallRecord{
				{
					ToolName: "search",
					Result:   longResult,
					Success:  true,
				},
			},
		}

		result := formatTrace(trace)
		assert.Contains(t, result, "...")
	})

	t.Run("trace with final answer", func(t *testing.T) {
		trace := &ExecutionTrace{
			Query:       "Test query",
			DurationMs:  1000,
			TotalTokens: 500,
			FinalAnswer: "The answer is 42.",
		}

		result := formatTrace(trace)
		assert.Contains(t, result, "Final Answer: The answer is 42.")
	})
}

func TestUniqueMerge(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: nil,
		},
		{
			name:     "nil slices",
			a:        nil,
			b:        nil,
			expected: nil,
		},
		{
			name:     "first slice only",
			a:        []string{"one", "two"},
			b:        nil,
			expected: []string{"one", "two"},
		},
		{
			name:     "second slice only",
			a:        nil,
			b:        []string{"three", "four"},
			expected: []string{"three", "four"},
		},
		{
			name:     "no duplicates",
			a:        []string{"one", "two"},
			b:        []string{"three", "four"},
			expected: []string{"one", "two", "three", "four"},
		},
		{
			name:     "with duplicates",
			a:        []string{"one", "two", "three"},
			b:        []string{"two", "three", "four"},
			expected: []string{"one", "two", "three", "four"},
		},
		{
			name:     "case insensitive duplicates",
			a:        []string{"One", "TWO"},
			b:        []string{"one", "two", "three"},
			expected: []string{"One", "TWO", "three"},
		},
		{
			name:     "with whitespace normalization",
			a:        []string{"  spaced  "},
			b:        []string{"spaced", "new"},
			expected: []string{"  spaced  ", "new"},
		},
		{
			name:     "empty strings filtered",
			a:        []string{"valid", ""},
			b:        []string{"", "also valid"},
			expected: []string{"valid", "also valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueMerge(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathMutator_MutateStrategy(t *testing.T) {
	t.Run("nil candidate returns error", func(t *testing.T) {
		mutator := NewPathMutator(&mockLLMService{}, nil)

		trace := &ExecutionTrace{Query: "test"}
		_, err := mutator.MutateStrategy(context.Background(), nil, trace, "feedback")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "candidate cannot be nil")
	})

	t.Run("nil trace returns error", func(t *testing.T) {
		mutator := NewPathMutator(&mockLLMService{}, nil)

		candidate := &PathCandidate{
			ID:             "test",
			StrategyPrompt: "original strategy",
		}
		_, err := mutator.MutateStrategy(context.Background(), candidate, nil, "feedback")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trace cannot be nil")
	})

	t.Run("successful mutation", func(t *testing.T) {
		mockLLM := &mockLLMService{
			chatResponse: &ports.LLMResponse{
				Content: `LESSONS_LEARNED:
- Always check for null values
- Use more specific queries

IMPROVED_STRATEGY:
New improved strategy that addresses the issues.`,
			},
		}

		mutator := NewPathMutator(mockLLM, nil)

		candidate := &PathCandidate{
			ID:                 "parent",
			RunID:              "run-1",
			Generation:         2,
			StrategyPrompt:     "Original strategy",
			AccumulatedLessons: []string{"Previous lesson"},
		}

		trace := &ExecutionTrace{
			Query:       "Test query",
			FinalAnswer: "Some answer",
			DurationMs:  1000,
		}

		result, err := mutator.MutateStrategy(context.Background(), candidate, trace, "Some feedback")
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.NotEmpty(t, result.ID)
		assert.Equal(t, "run-1", result.RunID)
		assert.Equal(t, 3, result.Generation)
		assert.Contains(t, result.ParentIDs, "parent")
		assert.Equal(t, "New improved strategy that addresses the issues.", result.StrategyPrompt)
		assert.Len(t, result.AccumulatedLessons, 3) // 1 previous + 2 new
	})

	t.Run("fallback when no strategy parsed", func(t *testing.T) {
		mockLLM := &mockLLMService{
			chatResponse: &ports.LLMResponse{
				Content: "Some response without proper format",
			},
		}

		mutator := NewPathMutator(mockLLM, nil)

		candidate := &PathCandidate{
			ID:             "parent",
			RunID:          "run-1",
			Generation:     1,
			StrategyPrompt: "Original strategy",
		}

		trace := &ExecutionTrace{Query: "Test"}

		result, err := mutator.MutateStrategy(context.Background(), candidate, trace, "Improve the approach")
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should contain fallback with feedback appended
		assert.Contains(t, result.StrategyPrompt, "Original strategy")
		assert.Contains(t, result.StrategyPrompt, "Additional guidance")
		assert.Contains(t, result.StrategyPrompt, "Improve the approach")
	})
}

func TestPathMutator_Crossover(t *testing.T) {
	t.Run("nil parent1 returns error", func(t *testing.T) {
		mutator := NewPathMutator(&mockLLMService{}, nil)

		parent2 := &PathCandidate{ID: "p2"}
		_, err := mutator.Crossover(context.Background(), nil, parent2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both parents must be non-nil")
	})

	t.Run("nil parent2 returns error", func(t *testing.T) {
		mutator := NewPathMutator(&mockLLMService{}, nil)

		parent1 := &PathCandidate{ID: "p1"}
		_, err := mutator.Crossover(context.Background(), parent1, nil)
		assert.Error(t, err)
	})

	t.Run("successful crossover", func(t *testing.T) {
		mockLLM := &mockLLMService{
			chatResponse: &ports.LLMResponse{
				Content: `MERGED_STRATEGY:
Combined strategy taking the best from both parents.
Uses thorough exploration from parent 1.
Uses efficient execution from parent 2.`,
			},
		}

		mutator := NewPathMutator(mockLLM, nil)

		parent1 := &PathCandidate{
			ID:                 "p1",
			RunID:              "run-1",
			Generation:         3,
			StrategyPrompt:     "Strategy 1: Focus on exploration",
			AccumulatedLessons: []string{"Lesson from parent 1"},
			Scores:             PathScores{AnswerQuality: 0.8, Efficiency: 0.5},
		}

		parent2 := &PathCandidate{
			ID:                 "p2",
			RunID:              "run-1",
			Generation:         2,
			StrategyPrompt:     "Strategy 2: Focus on efficiency",
			AccumulatedLessons: []string{"Lesson from parent 2"},
			Scores:             PathScores{AnswerQuality: 0.6, Efficiency: 0.9},
		}

		result, err := mutator.Crossover(context.Background(), parent1, parent2)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.NotEmpty(t, result.ID)
		assert.Equal(t, "run-1", result.RunID)
		assert.Equal(t, 4, result.Generation) // max(3, 2) + 1
		assert.Contains(t, result.ParentIDs, "p1")
		assert.Contains(t, result.ParentIDs, "p2")
		assert.Contains(t, result.StrategyPrompt, "Combined strategy")
		assert.Len(t, result.AccumulatedLessons, 2) // Lessons from both parents
	})

	t.Run("fallback when no merged strategy parsed", func(t *testing.T) {
		mockLLM := &mockLLMService{
			chatResponse: &ports.LLMResponse{
				Content: "Unparseable response",
			},
		}

		mutator := NewPathMutator(mockLLM, nil)

		parent1 := &PathCandidate{
			ID:             "p1",
			Generation:     1,
			StrategyPrompt: "Strategy 1",
			Scores:         PathScores{AnswerQuality: 0.8},
		}

		parent2 := &PathCandidate{
			ID:             "p2",
			Generation:     2,
			StrategyPrompt: "Strategy 2",
			Scores:         PathScores{AnswerQuality: 0.7},
		}

		result, err := mutator.Crossover(context.Background(), parent1, parent2)
		require.NoError(t, err)

		// Should fall back to concatenated strategies
		assert.Contains(t, result.StrategyPrompt, "Combined approach")
		assert.Contains(t, result.StrategyPrompt, "From strategy 1")
		assert.Contains(t, result.StrategyPrompt, "From strategy 2")
	})
}

func TestParseMergedStrategy(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name: "standard merged strategy",
			response: `MERGED_STRATEGY:
This is the merged strategy text.
It combines elements from both parents.`,
			expected: `This is the merged strategy text.
It combines elements from both parents.`,
		},
		{
			name: "falls back to improved strategy",
			response: `IMPROVED_STRATEGY:
Fallback strategy text`,
			expected: "Fallback strategy text",
		},
		{
			name:     "no strategy found",
			response: "Just some random text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMergedStrategy(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected string
	}{
		{
			name:     "empty args",
			args:     map[string]any{},
			expected: "",
		},
		{
			name: "simple args",
			args: map[string]any{
				"query": "test",
			},
			expected: `query="test"`,
		},
		{
			name: "long value truncated",
			args: map[string]any{
				"data": "this is a very long string that should be truncated to fifty characters because it exceeds the limit",
			},
			expected: `data="this is a very long string that should be truncate`, // Check partial match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatArgs(tt.args)
			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
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
			name:     "long string truncated",
			input:    "this is a long string",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
