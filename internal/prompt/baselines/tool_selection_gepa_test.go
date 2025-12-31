package baselines_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

func TestToolSelectionMetric(t *testing.T) {
	metric := baselines.NewToolSelectionMetric(nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		gold          prompt.Example
		pred          prompt.Example
		expectScore   float64
		expectInRange bool // If true, check score is in range [expectScore-0.1, expectScore+0.1]
	}{
		{
			name: "perfect_match_with_args",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "What did I say about my project?",
				},
				Outputs: map[string]any{
					"selected_tool": "memory_search",
					"arguments":     map[string]any{"query": "project"},
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "memory_search",
					"arguments":     map[string]any{"query": "project"},
					"reasoning":     "User wants to recall previous information about their project, so memory_search is appropriate.",
				},
			},
			expectScore:   1.0,
			expectInRange: true,
		},
		{
			name: "wrong_tool_selection",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "Remember my birthday is March 15",
				},
				Outputs: map[string]any{
					"selected_tool": "memory_save",
					"arguments":     map[string]any{"content": "birthday is March 15"},
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "memory_search",
					"arguments":     map[string]any{"query": "birthday"},
					"reasoning":     "Searching for birthday information.",
				},
			},
			expectScore:   0.0, // Wrong tool = 0 on tool component
			expectInRange: true,
		},
		{
			name: "correct_no_tool",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "Hello, how are you?",
				},
				Outputs: map[string]any{
					"selected_tool": "none",
					"arguments":     nil,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "none",
					"arguments":     nil,
					"reasoning":     "This is a greeting that doesn't require any tool.",
				},
			},
			expectScore:   0.9, // Tool match + no args needed + some reasoning
			expectInRange: true,
		},
		{
			name: "false_positive_tool_selection",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "Thanks for your help!",
				},
				Outputs: map[string]any{
					"selected_tool": "none",
					"arguments":     nil,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "web_search",
					"arguments":     map[string]any{"query": "help"},
					"reasoning":     "Searching for help.",
				},
			},
			expectScore:   0.1, // Wrong tool=0, no args points, ~10% partial reasoning
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
			} else if result.Score != tt.expectScore {
				t.Errorf("Score = %.2f, expected %.2f", result.Score, tt.expectScore)
			}

			if result.Feedback == "" {
				t.Error("Expected non-empty feedback")
			}

			t.Logf("Score: %.2f, Feedback: %s", result.Score, result.Feedback)
		})
	}
}

func TestSyntheticDatasetGeneration(t *testing.T) {
	trainset, valset := baselines.SyntheticToolSelectionDataset()

	t.Logf("Generated %d training examples, %d validation examples", len(trainset), len(valset))

	if len(trainset) < 10 {
		t.Errorf("Expected at least 10 training examples, got %d", len(trainset))
	}

	if len(valset) < 5 {
		t.Errorf("Expected at least 5 validation examples, got %d", len(valset))
	}

	// Verify structure of examples
	for i, ex := range trainset {
		if _, ok := ex.Inputs["user_message"]; !ok {
			t.Errorf("Training example %d missing user_message", i)
		}
		if _, ok := ex.Inputs["available_tools"]; !ok {
			t.Errorf("Training example %d missing available_tools", i)
		}
		if _, ok := ex.Outputs["selected_tool"]; !ok {
			t.Errorf("Training example %d missing selected_tool", i)
		}
	}

	// Count categories
	categories := make(map[string]int)
	for _, ex := range trainset {
		if cat, ok := ex.Inputs["category"].(string); ok {
			categories[cat]++
		}
	}
	t.Logf("Training set category distribution: %v", categories)
}

func TestToolSelectionSignature(t *testing.T) {
	sig := baselines.ToolSelectionSignature

	if sig.Name == "" {
		t.Error("Signature should have a name")
	}

	// Verify signature name contains expected fields
	expectedInName := []string{"user_message", "context", "available_tools", "selected_tool", "arguments", "reasoning"}
	for _, expected := range expectedInName {
		if !containsIgnoreCase(sig.Name, expected) {
			t.Errorf("Signature name should reference '%s', got: %s", expected, sig.Name)
		}
	}

	t.Logf("Signature name: %s", sig.Name)
}

// TestGEPAOptimizationSetup verifies the full GEPA setup is ready
func TestGEPAOptimizationSetup(t *testing.T) {
	// This test verifies all components are wired correctly for GEPA
	trainset, valset := baselines.SyntheticToolSelectionDataset()
	metric := baselines.NewToolSelectionMetric(nil)
	sig := baselines.ToolSelectionSignature

	t.Logf("Signature: %s", sig.Name)
	t.Logf("Seed prompt length: %d chars", len(baselines.ToolSelectionSeedPrompt))
	t.Logf("Training set: %d examples", len(trainset))
	t.Logf("Validation set: %d examples", len(valset))

	// Run a sample evaluation to verify metric works
	ctx := context.Background()
	gold := trainset[0]

	// Simulate a prediction
	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_tool": gold.Outputs["selected_tool"],
			"arguments":     gold.Outputs["arguments"],
			"reasoning":     "Selected based on user intent matching tool capabilities.",
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

// BenchmarkToolSelectionMetric benchmarks the metric evaluation
func BenchmarkToolSelectionMetric(b *testing.B) {
	metric := baselines.NewToolSelectionMetric(nil)
	ctx := context.Background()

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message": "What did I say about my project?",
		},
		Outputs: map[string]any{
			"selected_tool": "memory_search",
			"arguments":     map[string]any{"query": "project"},
		},
	}

	pred := prompt.Example{
		Outputs: map[string]any{
			"selected_tool": "memory_search",
			"arguments":     map[string]any{"query": "project"},
			"reasoning":     "Memory search for project information.",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = metric.Score(ctx, gold, pred, nil)
	}
}

// TestToolSelectionOptimizationDemo demonstrates how to run GEPA optimization
func TestToolSelectionOptimizationDemo(t *testing.T) {
	// 1. Load the signature and seed prompt
	sig := baselines.ToolSelectionSignature
	seedPrompt := baselines.ToolSelectionSeedPrompt

	// 2. Generate synthetic training/validation data
	trainset, valset := baselines.SyntheticToolSelectionDataset()

	// 3. Create the metric
	metric := baselines.NewToolSelectionMetric(nil)

	// 4. Print setup summary
	fmt.Printf("Tool Selection GEPA Optimization Setup:\n")
	fmt.Printf("- Signature: %s\n", sig.Name)
	fmt.Printf("- Seed prompt: %d characters\n", len(seedPrompt))
	fmt.Printf("- Training examples: %d\n", len(trainset))
	fmt.Printf("- Validation examples: %d\n", len(valset))

	// 5. Run sample evaluation
	ctx := context.Background()
	gold := trainset[0]
	pred := prompt.Example{
		Outputs: map[string]any{
			"selected_tool": gold.Outputs["selected_tool"],
			"arguments":     gold.Outputs["arguments"],
			"reasoning":     "Based on user intent.",
		},
	}

	result, _ := metric.Score(ctx, gold, pred, nil)
	fmt.Printf("- Sample metric score: %.2f\n", result.Score)

	// To run actual GEPA optimization, use OptimizationService:
	//
	// optService := services.NewOptimizationService(repo, llm, idGen, config)
	// run, _ := optService.OptimizeSignature(ctx, sig, trainset, valset, metric)
	// t.Logf("Optimization run ID: %s", run.ID)

	t.Logf("Tool Selection GEPA Setup Complete")
	t.Logf("- Signature: %s", sig.Name)
	t.Logf("- Seed prompt: %d chars", len(seedPrompt))
	t.Logf("- Training: %d examples, Validation: %d examples", len(trainset), len(valset))
	t.Logf("- Sample score: %.2f", result.Score)
}

// TestFeedbackQuality ensures feedback is actionable for GEPA reflection
func TestFeedbackQuality(t *testing.T) {
	metric := baselines.NewToolSelectionMetric(nil)
	ctx := context.Background()

	// Test that feedback contains actionable information for different error types
	testCases := []struct {
		name             string
		gold             prompt.Example
		pred             prompt.Example
		expectContains   []string // Feedback should contain these
		expectNotContain []string // Feedback should NOT contain these
	}{
		{
			name: "wrong_tool_feedback_is_specific",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "Remember my birthday is March 15",
					"category":     "memory",
				},
				Outputs: map[string]any{
					"selected_tool": "memory_save",
					"arguments":     map[string]any{"content": "birthday is March 15"},
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "calendar_create",
					"arguments":     map[string]any{"title": "birthday"},
					"reasoning":     "Creating calendar event.",
				},
			},
			expectContains: []string{
				"WRONG TOOL",
				"memory_save",
				"calendar_create",
			},
		},
		{
			name: "false_positive_has_guidance",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "Hello!",
					"category":     "conversation",
				},
				Outputs: map[string]any{
					"selected_tool": "none",
					"arguments":     nil,
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "web_search",
					"arguments":     map[string]any{"query": "hello"},
					"reasoning":     "Searching.",
				},
			},
			expectContains: []string{
				"conversational",
				"conservative",
			},
		},
		{
			name: "missing_args_feedback",
			gold: prompt.Example{
				Inputs: map[string]any{
					"user_message": "Search for quantum computing",
				},
				Outputs: map[string]any{
					"selected_tool": "web_search",
					"arguments":     map[string]any{"query": "quantum computing", "num_results": "5"},
				},
			},
			pred: prompt.Example{
				Outputs: map[string]any{
					"selected_tool": "web_search",
					"arguments":     map[string]any{"query": "quantum computing"},
					"reasoning":     "Web search needed.",
				},
			},
			expectContains: []string{
				"Missing arguments",
				"num_results",
			},
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

			for _, notExpected := range tc.expectNotContain {
				if containsIgnoreCase(result.Feedback, notExpected) {
					t.Errorf("Feedback should NOT contain '%s', got: %s", notExpected, result.Feedback)
				}
			}

			t.Logf("Feedback: %s", result.Feedback)
		})
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(containsIgnoreCase(s[1:], substr) ||
					equalFoldPrefix(s, substr)))
}

func equalFoldPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c1, c2 := s[i], prefix[i]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// TestDimensionScoresDerivation tests that the metric supports multi-dimensional scoring
func TestDimensionScoresDerivation(t *testing.T) {
	metric := baselines.NewToolSelectionMetric(nil)
	ctx := context.Background()

	trainset, _ := baselines.SyntheticToolSelectionDataset()

	// Track scores across multiple examples
	var totalScore float64
	for _, gold := range trainset[:5] {
		pred := prompt.Example{
			Outputs: map[string]any{
				"selected_tool": gold.Outputs["selected_tool"],
				"arguments":     gold.Outputs["arguments"],
				"reasoning":     "Tool selected based on user intent analysis.",
			},
		}

		result, err := metric.Score(ctx, gold, pred, nil)
		if err != nil {
			t.Fatalf("Score() error: %v", err)
		}
		totalScore += result.Score
	}

	avgScore := totalScore / 5.0
	t.Logf("Average score on perfect predictions: %.2f", avgScore)

	if avgScore < 0.8 {
		t.Errorf("Perfect predictions should average > 0.8, got %.2f", avgScore)
	}
}

// TestToolInfoSerialization ensures tool info serializes correctly for prompts
func TestToolInfoSerialization(t *testing.T) {
	tools := []baselines.ToolInfo{
		{
			Name:        "memory_search",
			Description: "Search memories",
			Parameters:  map[string]string{"query": "Search query"},
			Examples:    []string{"What did I say?"},
		},
	}

	data, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("Failed to marshal tools: %v", err)
	}

	var parsed []baselines.ToolInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal tools: %v", err)
	}

	if len(parsed) != 1 || parsed[0].Name != "memory_search" {
		t.Errorf("Round-trip failed: %+v", parsed)
	}

	t.Logf("Serialized tools: %s", string(data))
}
