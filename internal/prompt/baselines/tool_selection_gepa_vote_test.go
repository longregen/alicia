package baselines

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
)

func TestToolSelectionMetric_ScoreVoteExample_Downvote(t *testing.T) {
	metric := NewToolSelectionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":    "What's the weather?",
			"context":         "",
			"available_tools": "[]",
		},
		Outputs: map[string]any{
			"selected_tool":   "web_search",
			"arguments":       map[string]any{"query": "weather"},
			"_vote_value":     -1, // Downvote
			"_quick_feedback": "wrong_tool",
			"_vote_feedback":  "...",
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_tool": "web_search",
			"arguments":     map[string]any{"query": "weather"},
			"reasoning":     "User asked about weather",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 0.1 {
		t.Errorf("Expected score 0.1 for downvote, got %f", result.Score)
	}

	if result.Feedback == "" {
		t.Error("Expected diagnostic feedback for downvote, got empty string")
	}

	if result.Feedback != "The selected tool 'web_search' was incorrect for this query. Consider what the user actually needed." {
		t.Errorf("Unexpected feedback: %s", result.Feedback)
	}
}

func TestToolSelectionMetric_ScoreVoteExample_Upvote(t *testing.T) {
	metric := NewToolSelectionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":    "Remember my birthday is June 15",
			"context":         "",
			"available_tools": "[]",
		},
		Outputs: map[string]any{
			"selected_tool": "memory_save",
			"arguments":     map[string]any{"content": "birthday is June 15"},
			"_vote_value":   1, // Upvote
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_tool": "memory_save",
			"arguments":     map[string]any{"content": "birthday is June 15"},
			"reasoning":     "User wants to save information",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 1.0 {
		t.Errorf("Expected score 1.0 for upvote with matching tool, got %f", result.Score)
	}
}

func TestToolSelectionMetric_ScoreVoteExample_Upvote_DifferentTool(t *testing.T) {
	metric := NewToolSelectionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":    "Remember my birthday is June 15",
			"context":         "",
			"available_tools": "[]",
		},
		Outputs: map[string]any{
			"selected_tool": "memory_save",
			"arguments":     map[string]any{"content": "birthday is June 15"},
			"_vote_value":   1, // Upvote
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_tool": "memory_search", // Different tool
			"arguments":     map[string]any{"query": "birthday"},
			"reasoning":     "Searching for birthday",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 0.5 {
		t.Errorf("Expected score 0.5 for upvote with different tool, got %f", result.Score)
	}
}

func TestToolSelectionMetric_ScoreSyntheticExample(t *testing.T) {
	metric := NewToolSelectionMetric(nil)

	// Synthetic example (no vote metadata)
	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":    "What's 2+2?",
			"context":         "",
			"available_tools": "[]",
		},
		Outputs: map[string]any{
			"selected_tool": "calculator",
			"arguments":     map[string]any{"expression": "2+2"},
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_tool": "calculator",
			"arguments":     map[string]any{"expression": "2+2"},
			"reasoning":     "User asked for calculation",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	// Should use synthetic scoring logic (0.5 for tool + 0.3 for args + 0.2 for reasoning)
	if result.Score < 0.9 {
		t.Errorf("Expected high score for perfect synthetic match, got %f", result.Score)
	}
}
