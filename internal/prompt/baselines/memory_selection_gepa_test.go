package baselines_test

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

func TestMemorySelectionMetric(t *testing.T) {
	metric := baselines.NewMemorySelectionMetric(nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		gold          prompt.Example
		pred          prompt.Example
		expectScore   float64
		expectInRange bool
	}{
		{
			name: "perfect_selection",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message":       "What's my daughter's name?",
					"candidate_memories": `[{"id":"mem_001","content":"Daughter is Emma"}]`,
				},
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001"]`,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001"]`,
					"relevance_reasoning": "mem_001 directly answers the user's question about their daughter's name.",
				},
			},
			expectScore:   1.0,
			expectInRange: true,
		},
		{
			name: "over_selection",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message":       "What's the weather like?",
					"candidate_memories": `[{"id":"mem_001"},{"id":"mem_002"}]`,
				},
				Outputs: map[string]any{
					"selected_memory_ids": `[]`,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001"]`,
					"relevance_reasoning": "Selected memory about weather.",
				},
			},
			expectScore:   0.4, // Low precision, good recall (nothing to recall)
			expectInRange: true,
		},
		{
			name: "missed_memory",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message":       "Tell me about my project",
					"candidate_memories": `[{"id":"mem_001"},{"id":"mem_002"}]`,
				},
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001", "mem_002"]`,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001"]`,
					"relevance_reasoning": "Selected project memory.",
				},
			},
			expectScore:   0.7, // Good precision, partial recall
			expectInRange: true,
		},
		{
			name: "correct_empty_selection",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message":       "Hello!",
					"candidate_memories": `[{"id":"mem_001"}]`,
				},
				Outputs: map[string]any{
					"selected_memory_ids": `[]`,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_memory_ids": `[]`,
					"relevance_reasoning": "Greeting doesn't require memory context.",
				},
			},
			expectScore:   0.9, // Perfect but reasoning might not be perfect
			expectInRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := metric.Score(ctx, tt.gold, tt.pred, nil)
			if err != nil {
				t.Fatalf("Score() error: %v", err)
			}

			if tt.expectInRange {
				if result.Score < tt.expectScore-0.2 || result.Score > tt.expectScore+0.2 {
					t.Errorf("Score = %.2f, expected ~%.2f", result.Score, tt.expectScore)
				}
			}

			if result.Feedback == "" {
				t.Error("Expected non-empty feedback")
			}

			t.Logf("Score: %.2f, Feedback: %s", result.Score, result.Feedback)
		})
	}
}

func TestSyntheticMemorySelectionDataset(t *testing.T) {
	trainset, valset := baselines.SyntheticMemorySelectionDataset()

	t.Logf("Generated %d training examples, %d validation examples", len(trainset), len(valset))

	if len(trainset) < 5 {
		t.Errorf("Expected at least 5 training examples, got %d", len(trainset))
	}

	if len(valset) < 2 {
		t.Errorf("Expected at least 2 validation examples, got %d", len(valset))
	}

	// Verify structure
	for i, ex := range trainset {
		if _, ok := ex.Inputs["user_message"]; !ok {
			t.Errorf("Training example %d missing user_message", i)
		}
		if _, ok := ex.Inputs["candidate_memories"]; !ok {
			t.Errorf("Training example %d missing candidate_memories", i)
		}
		if _, ok := ex.Outputs["selected_memory_ids"]; !ok {
			t.Errorf("Training example %d missing selected_memory_ids", i)
		}
	}
}

func TestMemorySelectionSignature(t *testing.T) {
	sig := baselines.MemorySelectionSignature

	if sig.Name == "" {
		t.Error("Signature should have a name")
	}

	t.Logf("Memory Selection Signature: %s", sig.Name)
}

func TestMemoryRankingSignature(t *testing.T) {
	sig := baselines.MemoryRankingSignature

	if sig.Name == "" {
		t.Error("Signature should have a name")
	}

	t.Logf("Memory Ranking Signature: %s", sig.Name)
}

func TestMemorySelectionGEPASetup(t *testing.T) {
	trainset, valset := baselines.SyntheticMemorySelectionDataset()
	metric := baselines.NewMemorySelectionMetric(nil)
	sig := baselines.MemorySelectionSignature

	t.Logf("Signature: %s", sig.Name)
	t.Logf("Seed prompt length: %d chars", len(baselines.MemorySelectionSeedPrompt))
	t.Logf("Training set: %d examples", len(trainset))
	t.Logf("Validation set: %d examples", len(valset))

	// Run a sample evaluation
	ctx := context.Background()
	gold := trainset[0]

	// Simulate perfect prediction
	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_memory_ids": gold.Outputs["selected_memory_ids"],
			"relevance_reasoning": "Selected relevant memories based on user's question context.",
		},
	}

	result, err := metric.Score(ctx, gold, pred, nil)
	if err != nil {
		t.Fatalf("Metric score failed: %v", err)
	}

	t.Logf("Sample evaluation - Score: %.2f, Feedback: %s", result.Score, result.Feedback)

	if result.Score < 0.5 {
		t.Error("Perfect match should score > 0.5")
	}
}

func TestMemorySelectionFeedbackQuality(t *testing.T) {
	metric := baselines.NewMemorySelectionMetric(nil)
	ctx := context.Background()

	testCases := []struct {
		name           string
		gold           prompt.Example
		pred           prompt.Example
		expectContains []string
	}{
		{
			name: "over_selection_feedback",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message":       "Thanks!",
					"candidate_memories": `[{"id":"mem_001"}]`,
				},
				Outputs: map[string]any{
					"selected_memory_ids": `[]`,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001"]`,
					"relevance_reasoning": "Selected memory.",
				},
			},
			expectContains: []string{"OVER-SELECTED", "conservative"},
		},
		{
			name: "missed_memory_feedback",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message":       "What's my project status?",
					"candidate_memories": `[{"id":"mem_001"}]`,
				},
				Outputs: map[string]any{
					"selected_memory_ids": `["mem_001"]`,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_memory_ids": `[]`,
					"relevance_reasoning": "No memories needed.",
				},
			},
			expectContains: []string{"MISSED"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := metric.Score(ctx, tc.gold, tc.pred, nil)
			if err != nil {
				t.Fatalf("Score() error: %v", err)
			}

			for _, expected := range tc.expectContains {
				if !containsIgnoreCase(result.Feedback, expected) {
					t.Errorf("Feedback should contain '%s', got: %s", expected, result.Feedback)
				}
			}

			t.Logf("Feedback: %s", result.Feedback)
		})
	}
}

func BenchmarkMemorySelectionMetric(b *testing.B) {
	metric := baselines.NewMemorySelectionMetric(nil)
	ctx := context.Background()

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":       "What's my project?",
			"candidate_memories": `[{"id":"mem_001"},{"id":"mem_002"},{"id":"mem_003"}]`,
		},
		Outputs: map[string]any{
			"selected_memory_ids": `["mem_001"]`,
		},
	}

	pred := prompt.Example{
		Outputs: map[string]any{
			"selected_memory_ids": `["mem_001"]`,
			"relevance_reasoning": "Selected project memory.",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = metric.Score(ctx, gold, pred, nil)
	}
}
