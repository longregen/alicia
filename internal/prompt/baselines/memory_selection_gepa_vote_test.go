package baselines

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
)

func TestMemorySelectionMetric_ScoreVoteExample_Downvote(t *testing.T) {
	metric := NewMemorySelectionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":         "What was my project?",
			"conversation_context": "",
			"candidate_memories":   "[]",
		},
		Outputs: map[string]any{
			"selected_memory_id": "mem_123",
			"_vote_value":        -1, // Downvote
			"_quick_feedback":    "wrong_context",
			"_vote_feedback":     "...",
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_memory_id": "mem_123",
			"reasoning":          "This memory seems relevant",
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

	expectedFeedback := "This memory was retrieved but wasn't relevant to the user's actual intent."
	if result.Feedback != expectedFeedback {
		t.Errorf("Expected feedback '%s', got '%s'", expectedFeedback, result.Feedback)
	}
}

func TestMemorySelectionMetric_ScoreVoteExample_Upvote(t *testing.T) {
	metric := NewMemorySelectionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":         "What was my project?",
			"conversation_context": "",
			"candidate_memories":   "[]",
		},
		Outputs: map[string]any{
			"selected_memory_id": "mem_456",
			"_vote_value":        1, // Upvote
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_memory_id": "mem_456",
			"reasoning":          "This memory is relevant",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 1.0 {
		t.Errorf("Expected score 1.0 for upvote with matching memory, got %f", result.Score)
	}
}

func TestMemorySelectionMetric_ScoreVoteExample_Upvote_DifferentMemory(t *testing.T) {
	metric := NewMemorySelectionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"user_message":         "What was my project?",
			"conversation_context": "",
			"candidate_memories":   "[]",
		},
		Outputs: map[string]any{
			"selected_memory_id": "mem_456",
			"_vote_value":        1, // Upvote
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"selected_memory_id": "mem_789", // Different memory
			"reasoning":          "This memory is relevant",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 0.5 {
		t.Errorf("Expected score 0.5 for upvote with different memory, got %f", result.Score)
	}
}

func TestMemorySelectionMetric_BuildDiagnosticFeedback(t *testing.T) {
	tests := []struct {
		quickFeedback    string
		expectedFeedback string
	}{
		{"wrong_context", "This memory was retrieved but wasn't relevant to the user's actual intent."},
		{"too_generic", "This memory was too generic to be useful. More specific memories should be prioritized."},
		{"outdated", "This memory contains outdated information. Consider recency when selecting memories."},
		{"incorrect", "This memory contains incorrect information that should not have been used."},
		{"unknown", "The memory selection was marked as incorrect by the user."},
	}

	for _, tt := range tests {
		t.Run(tt.quickFeedback, func(t *testing.T) {
			feedback := buildDiagnosticFeedbackForMemoryVote(tt.quickFeedback)
			if feedback != tt.expectedFeedback {
				t.Errorf("Expected '%s', got '%s'", tt.expectedFeedback, feedback)
			}
		})
	}
}
