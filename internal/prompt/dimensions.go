package prompt

// DimensionWeights configures the relative importance of each optimization dimension
type DimensionWeights struct {
	SuccessRate    float64 `json:"successRate" yaml:"success_rate"`       // Default: 0.25
	Quality        float64 `json:"quality" yaml:"quality"`                // Default: 0.20
	Efficiency     float64 `json:"efficiency" yaml:"efficiency"`          // Default: 0.15
	Robustness     float64 `json:"robustness" yaml:"robustness"`          // Default: 0.15
	Generalization float64 `json:"generalization" yaml:"generalization"`  // Default: 0.10
	Diversity      float64 `json:"diversity" yaml:"diversity"`            // Default: 0.10
	Innovation     float64 `json:"innovation" yaml:"innovation"`          // Default: 0.05
}

// DefaultWeights returns balanced dimension weights
func DefaultWeights() DimensionWeights {
	return DimensionWeights{
		SuccessRate:    0.25,
		Quality:        0.20,
		Efficiency:     0.15,
		Robustness:     0.15,
		Generalization: 0.10,
		Diversity:      0.10,
		Innovation:     0.05,
	}
}

// Normalize ensures weights sum to 1.0
func (w *DimensionWeights) Normalize() {
	sum := w.SuccessRate + w.Quality + w.Efficiency + w.Robustness +
		w.Generalization + w.Diversity + w.Innovation

	if sum > 0 {
		w.SuccessRate /= sum
		w.Quality /= sum
		w.Efficiency /= sum
		w.Robustness /= sum
		w.Generalization /= sum
		w.Diversity /= sum
		w.Innovation /= sum
	}
}

// ToMap converts DimensionWeights to a map for interface compatibility
func (w DimensionWeights) ToMap() map[string]float64 {
	return map[string]float64{
		"successRate":    w.SuccessRate,
		"quality":        w.Quality,
		"efficiency":     w.Efficiency,
		"robustness":     w.Robustness,
		"generalization": w.Generalization,
		"diversity":      w.Diversity,
		"innovation":     w.Innovation,
	}
}

// DimensionWeightsFromMap creates DimensionWeights from a map
func DimensionWeightsFromMap(m map[string]float64) DimensionWeights {
	return DimensionWeights{
		SuccessRate:    m["successRate"],
		Quality:        m["quality"],
		Efficiency:     m["efficiency"],
		Robustness:     m["robustness"],
		Generalization: m["generalization"],
		Diversity:      m["diversity"],
		Innovation:     m["innovation"],
	}
}

// DimensionScores holds per-dimension performance metrics
type DimensionScores struct {
	SuccessRate    float64 `json:"successRate"`
	Quality        float64 `json:"quality"`
	Efficiency     float64 `json:"efficiency"`
	Robustness     float64 `json:"robustness"`
	Generalization float64 `json:"generalization"`
	Diversity      float64 `json:"diversity"`
	Innovation     float64 `json:"innovation"`
}

// WeightedScore calculates the weighted score for a set of dimension scores
func (s *DimensionScores) WeightedScore(weights DimensionWeights) float64 {
	return s.SuccessRate*weights.SuccessRate +
		s.Quality*weights.Quality +
		s.Efficiency*weights.Efficiency +
		s.Robustness*weights.Robustness +
		s.Generalization*weights.Generalization +
		s.Diversity*weights.Diversity +
		s.Innovation*weights.Innovation
}
