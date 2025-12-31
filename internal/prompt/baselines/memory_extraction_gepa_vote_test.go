package baselines

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
)

func TestMemoryExtractionMetric_ScoreVoteExample_Downvote(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "I love pizza",
			"conversation_context": "",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["User loves pizza"]`,
			"importance_scores": `[0.7]`,
			"_vote_value":       -1, // Downvote
			"_quick_feedback":   "too_generic",
			"_vote_feedback":    "...",
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `["User loves pizza"]`,
			"importance_scores":    `[0.7]`,
			"extraction_reasoning": "Extracted preference",
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

	expectedFeedback := "The extracted memory was too generic to be useful. Extract more specific, actionable facts."
	if result.Feedback != expectedFeedback {
		t.Errorf("Expected feedback '%s', got '%s'", expectedFeedback, result.Feedback)
	}
}

func TestMemoryExtractionMetric_ScoreVoteExample_Upvote(t *testing.T) {
	metric := NewMemoryExtractionMetric(nil)

	gold := prompt.Example{
		Inputs: map[string]any{
			"conversation_text":    "My daughter Emma is 5 years old",
			"conversation_context": "",
		},
		Outputs: map[string]any{
			"extracted_facts":   `["User has a daughter named Emma who is 5 years old"]`,
			"importance_scores": `[0.9]`,
			"_vote_value":       1, // Upvote
		},
	}

	pred := prompt.Example{
		Inputs: gold.Inputs,
		Outputs: map[string]any{
			"extracted_facts":      `["User has a daughter named Emma who is 5 years old"]`,
			"importance_scores":    `[0.9]`,
			"extraction_reasoning": "Extracted biographical fact",
		},
	}

	result, err := metric.Score(context.Background(), gold, pred, nil)
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 1.0 {
		t.Errorf("Expected score 1.0 for upvote, got %f", result.Score)
	}

	expectedFeedback := "Memory extraction was marked as correct by the user."
	if result.Feedback != expectedFeedback {
		t.Errorf("Expected feedback '%s', got '%s'", expectedFeedback, result.Feedback)
	}
}

func TestMemoryExtractionMetric_BuildDiagnosticFeedback(t *testing.T) {
	tests := []struct {
		quickFeedback    string
		expectedFeedback string
	}{
		{"too_generic", "The extracted memory was too generic to be useful. Extract more specific, actionable facts."},
		{"incorrect", "The extracted information was incorrect. Only extract facts that are explicitly stated."},
		{"not_factual", "The extracted content was not factual. Focus on objective facts, not opinions or interpretations."},
		{"missing_context", "The extraction missed important context. Include relevant details that make the fact meaningful."},
		{"unknown", "The memory extraction was marked as incorrect by the user."},
	}

	for _, tt := range tests {
		t.Run(tt.quickFeedback, func(t *testing.T) {
			feedback := buildDiagnosticFeedbackForMemoryExtractionVote(tt.quickFeedback)
			if feedback != tt.expectedFeedback {
				t.Errorf("Expected '%s', got '%s'", tt.expectedFeedback, feedback)
			}
		})
	}
}
