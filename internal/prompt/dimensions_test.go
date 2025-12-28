package prompt

import (
	"math"
	"testing"
)

func TestDefaultWeights(t *testing.T) {
	weights := DefaultWeights()

	// Check that all weights are positive
	if weights.SuccessRate <= 0 {
		t.Error("SuccessRate should be positive")
	}
	if weights.Quality <= 0 {
		t.Error("Quality should be positive")
	}
	if weights.Efficiency <= 0 {
		t.Error("Efficiency should be positive")
	}
	if weights.Robustness <= 0 {
		t.Error("Robustness should be positive")
	}
	if weights.Generalization <= 0 {
		t.Error("Generalization should be positive")
	}
	if weights.Diversity <= 0 {
		t.Error("Diversity should be positive")
	}
	if weights.Innovation <= 0 {
		t.Error("Innovation should be positive")
	}

	// Check that weights sum to 1.0
	sum := weights.SuccessRate + weights.Quality + weights.Efficiency +
		weights.Robustness + weights.Generalization + weights.Diversity +
		weights.Innovation

	if math.Abs(sum-1.0) > 0.0001 {
		t.Errorf("Weights should sum to 1.0, got %f", sum)
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		weights DimensionWeights
	}{
		{
			name: "already normalized",
			weights: DimensionWeights{
				SuccessRate:    0.25,
				Quality:        0.25,
				Efficiency:     0.25,
				Robustness:     0.25,
				Generalization: 0.0,
				Diversity:      0.0,
				Innovation:     0.0,
			},
		},
		{
			name: "needs normalization",
			weights: DimensionWeights{
				SuccessRate:    2.0,
				Quality:        2.0,
				Efficiency:     2.0,
				Robustness:     2.0,
				Generalization: 0.0,
				Diversity:      0.0,
				Innovation:     0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.weights
			w.Normalize()

			sum := w.SuccessRate + w.Quality + w.Efficiency +
				w.Robustness + w.Generalization + w.Diversity +
				w.Innovation

			if math.Abs(sum-1.0) > 0.0001 {
				t.Errorf("After normalization, weights should sum to 1.0, got %f", sum)
			}
		})
	}
}

func TestWeightedScore(t *testing.T) {
	scores := DimensionScores{
		SuccessRate:    0.9,
		Quality:        0.8,
		Efficiency:     0.7,
		Robustness:     0.6,
		Generalization: 0.5,
		Diversity:      0.4,
		Innovation:     0.3,
	}

	weights := DimensionWeights{
		SuccessRate:    1.0,
		Quality:        0.0,
		Efficiency:     0.0,
		Robustness:     0.0,
		Generalization: 0.0,
		Diversity:      0.0,
		Innovation:     0.0,
	}

	// With all weight on SuccessRate, result should be 0.9
	result := scores.WeightedScore(weights)
	if math.Abs(result-0.9) > 0.0001 {
		t.Errorf("Expected weighted score of 0.9, got %f", result)
	}

	// Test with default balanced weights
	weights = DefaultWeights()
	result = scores.WeightedScore(weights)
	if result < 0 || result > 1 {
		t.Errorf("Weighted score should be between 0 and 1, got %f", result)
	}
}
