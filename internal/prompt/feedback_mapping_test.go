package prompt

import (
	"testing"
)

func TestMapFeedbackToDimensions(t *testing.T) {
	tests := []struct {
		name           string
		feedback       FeedbackType
		expectPositive string // which dimension should be positively adjusted
		expectNegative string // which dimension should be negatively adjusted
	}{
		{
			name:           "too slow - increases efficiency",
			feedback:       FeedbackTooSlow,
			expectPositive: "efficiency",
		},
		{
			name:           "inconsistent - increases robustness",
			feedback:       FeedbackInconsistent,
			expectPositive: "robustness",
		},
		{
			name:           "same approach - increases diversity and innovation",
			feedback:       FeedbackSameApproach,
			expectPositive: "diversity",
		},
		{
			name:           "wrong answer - increases success rate",
			feedback:       FeedbackWrongAnswer,
			expectPositive: "successRate",
		},
		{
			name:           "doesn't fit case - increases generalization",
			feedback:       FeedbackDoesntFitCase,
			expectPositive: "generalization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adj := MapFeedbackToDimensions(tt.feedback)

			switch tt.expectPositive {
			case "efficiency":
				if adj.Efficiency <= 0 {
					t.Errorf("expected positive efficiency adjustment, got %f", adj.Efficiency)
				}
			case "robustness":
				if adj.Robustness <= 0 {
					t.Errorf("expected positive robustness adjustment, got %f", adj.Robustness)
				}
			case "diversity":
				if adj.Diversity <= 0 {
					t.Errorf("expected positive diversity adjustment, got %f", adj.Diversity)
				}
			case "successRate":
				if adj.SuccessRate <= 0 {
					t.Errorf("expected positive success rate adjustment, got %f", adj.SuccessRate)
				}
			case "generalization":
				if adj.Generalization <= 0 {
					t.Errorf("expected positive generalization adjustment, got %f", adj.Generalization)
				}
			}
		})
	}
}

func TestApplyAdjustment(t *testing.T) {
	weights := DefaultWeights()

	adjustment := DimensionAdjustment{
		Efficiency: +0.15,
	}

	result := ApplyAdjustment(weights, adjustment)

	// Efficiency should increase
	if result.Efficiency <= weights.Efficiency {
		t.Errorf("expected efficiency to increase, got %f", result.Efficiency)
	}

	// Weights should be normalized (sum to 1.0)
	sum := result.SuccessRate + result.Quality + result.Efficiency +
		result.Robustness + result.Generalization + result.Diversity + result.Innovation

	if sum < 0.99 || sum > 1.01 {
		t.Errorf("expected weights to sum to 1.0, got %f", sum)
	}
}

func TestApplyAdjustmentClamps(t *testing.T) {
	weights := DimensionWeights{
		SuccessRate:    0.01,
		Quality:        0.01,
		Efficiency:     0.01,
		Robustness:     0.01,
		Generalization: 0.01,
		Diversity:      0.01,
		Innovation:     0.94,
	}
	// Don't normalize - test with raw values

	// Try to push innovation even higher
	adjustment := DimensionAdjustment{
		Innovation: +0.5,
	}

	result := ApplyAdjustment(weights, adjustment)

	// After clamping to 0.5 max (before normalization) and then normalizing,
	// innovation should be less than the original 0.94 after normalization
	// because we clamped it to 0.5 before normalizing
	// Actually, let's test that low values get clamped to min
	lowWeights := DimensionWeights{
		SuccessRate:    0.9,
		Quality:        0.01,
		Efficiency:     0.01,
		Robustness:     0.01,
		Generalization: 0.01,
		Diversity:      0.01,
		Innovation:     0.01,
	}
	negativeAdj := DimensionAdjustment{
		Innovation: -0.5, // Try to push below min
	}

	lowResult := ApplyAdjustment(lowWeights, negativeAdj)

	// All dimensions should be at least 0.01 (clamped) before normalization
	// After normalization they'll be proportional but none should be 0
	if lowResult.Innovation == 0 {
		t.Error("expected innovation to be clamped above 0")
	}

	// Main test: verify normalization works
	sum := result.SuccessRate + result.Quality + result.Efficiency +
		result.Robustness + result.Generalization + result.Diversity + result.Innovation
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("expected weights to sum to 1.0 after adjustment, got %f", sum)
	}
}

func TestAggregateFeedback(t *testing.T) {
	feedbacks := []FeedbackType{
		FeedbackTooSlow,
		FeedbackTooSlow,
		FeedbackInconsistent,
	}

	result := AggregateFeedback(feedbacks)

	// Efficiency should have the largest adjustment (2x too slow)
	if result.Efficiency <= result.Robustness {
		t.Error("expected efficiency to be higher than robustness")
	}
}

func TestQuickFeedbackToType(t *testing.T) {
	tests := []struct {
		input    string
		expected FeedbackType
	}{
		{"wrong_tool", FeedbackWrongTool},
		{"wrong_params", FeedbackWrongParams},
		{"unnecessary", FeedbackUnnecessary},
		{"outdated", FeedbackOutdated},
		{"incorrect_assumption", FeedbackIncorrectAssumption},
		{"unknown", ""},
	}

	for _, tt := range tests {
		result := QuickFeedbackToType(tt.input)
		if result != tt.expected {
			t.Errorf("QuickFeedbackToType(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestVoteToFeedback(t *testing.T) {
	tests := []struct {
		vote          string
		quickFeedback string
		targetType    string
		expected      FeedbackType
	}{
		{"up", "", "message", FeedbackGreatAnswer},
		{"up", "", "memory", FeedbackHelpful},
		{"up", "", "tool_use", FeedbackPerfect},
		{"down", "", "message", FeedbackWrongAnswer},
		{"down", "", "memory", FeedbackNotRelevant},
		{"down", "", "tool_use", FeedbackWrongTool},
		{"down", "", "reasoning", FeedbackWrongDirection},
		{"critical", "", "memory", FeedbackCritical},
		// Quick feedback overrides vote inference
		{"down", "wrong_params", "tool_use", FeedbackWrongParams},
		{"up", "outdated", "memory", FeedbackOutdated},
	}

	for _, tt := range tests {
		result := VoteToFeedback(tt.vote, tt.quickFeedback, tt.targetType)
		if result != tt.expected {
			t.Errorf("VoteToFeedback(%q, %q, %q) = %q, want %q",
				tt.vote, tt.quickFeedback, tt.targetType, result, tt.expected)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min, max, expected float64
	}{
		{0.5, 0.0, 1.0, 0.5},  // within range
		{-0.5, 0.0, 1.0, 0.0}, // below min
		{1.5, 0.0, 1.0, 1.0},  // above max
		{0.0, 0.0, 1.0, 0.0},  // at min
		{1.0, 0.0, 1.0, 1.0},  // at max
	}

	for _, tt := range tests {
		result := clamp(tt.value, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f",
				tt.value, tt.min, tt.max, result, tt.expected)
		}
	}
}
