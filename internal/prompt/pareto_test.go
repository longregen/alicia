package prompt

import (
	"testing"
	"time"
)

func TestNewParetoArchive(t *testing.T) {
	archive := NewParetoArchive(10)
	if archive.MaxSize != 10 {
		t.Errorf("expected MaxSize 10, got %d", archive.MaxSize)
	}
	if archive.Size() != 0 {
		t.Errorf("expected empty archive, got size %d", archive.Size())
	}
}

func TestNewParetoArchiveDefaultSize(t *testing.T) {
	archive := NewParetoArchive(0)
	if archive.MaxSize != 50 {
		t.Errorf("expected default MaxSize 50, got %d", archive.MaxSize)
	}
}

func TestParetoArchiveAdd(t *testing.T) {
	archive := NewParetoArchive(10)

	solution := &EliteSolution{
		ID:           "sol1",
		Instructions: "Test instructions",
		Scores: DimensionScores{
			SuccessRate: 0.8,
			Quality:     0.7,
			Efficiency:  0.6,
		},
		CreatedAt: time.Now(),
	}

	added := archive.Add(solution)
	if !added {
		t.Error("expected solution to be added")
	}
	if archive.Size() != 1 {
		t.Errorf("expected size 1, got %d", archive.Size())
	}
}

func TestParetoArchiveAddDominated(t *testing.T) {
	archive := NewParetoArchive(10)

	// Add a good solution
	good := &EliteSolution{
		ID: "good",
		Scores: DimensionScores{
			SuccessRate:    0.9,
			Quality:        0.9,
			Efficiency:     0.9,
			Robustness:     0.9,
			Generalization: 0.9,
			Diversity:      0.9,
			Innovation:     0.9,
		},
	}
	archive.Add(good)

	// Try to add a dominated solution
	bad := &EliteSolution{
		ID: "bad",
		Scores: DimensionScores{
			SuccessRate:    0.5,
			Quality:        0.5,
			Efficiency:     0.5,
			Robustness:     0.5,
			Generalization: 0.5,
			Diversity:      0.5,
			Innovation:     0.5,
		},
	}
	added := archive.Add(bad)
	if added {
		t.Error("dominated solution should not be added")
	}
	if archive.Size() != 1 {
		t.Errorf("expected size 1, got %d", archive.Size())
	}
}

func TestParetoArchiveRemovesDominated(t *testing.T) {
	archive := NewParetoArchive(10)

	// Add a mediocre solution
	mediocre := &EliteSolution{
		ID: "mediocre",
		Scores: DimensionScores{
			SuccessRate:    0.5,
			Quality:        0.5,
			Efficiency:     0.5,
			Robustness:     0.5,
			Generalization: 0.5,
			Diversity:      0.5,
			Innovation:     0.5,
		},
	}
	archive.Add(mediocre)
	if archive.Size() != 1 {
		t.Errorf("expected size 1, got %d", archive.Size())
	}

	// Add a better solution that dominates
	better := &EliteSolution{
		ID: "better",
		Scores: DimensionScores{
			SuccessRate:    0.9,
			Quality:        0.9,
			Efficiency:     0.9,
			Robustness:     0.9,
			Generalization: 0.9,
			Diversity:      0.9,
			Innovation:     0.9,
		},
	}
	archive.Add(better)

	// Mediocre should be removed
	if archive.Size() != 1 {
		t.Errorf("expected size 1 after domination, got %d", archive.Size())
	}

	// Only better should remain
	all := archive.GetAll()
	if all[0].ID != "better" {
		t.Errorf("expected 'better' to remain, got '%s'", all[0].ID)
	}
}

func TestParetoArchiveSelectByWeights(t *testing.T) {
	archive := NewParetoArchive(10)

	// Add solutions with different strengths
	accurate := &EliteSolution{
		ID: "accurate",
		Scores: DimensionScores{
			SuccessRate: 0.95,
			Quality:     0.7,
			Efficiency:  0.5,
		},
	}
	fast := &EliteSolution{
		ID: "fast",
		Scores: DimensionScores{
			SuccessRate: 0.7,
			Quality:     0.6,
			Efficiency:  0.95,
		},
	}

	archive.Add(accurate)
	archive.Add(fast)

	// Select with accuracy-focused weights
	accuracyWeights := DimensionWeights{
		SuccessRate: 0.8,
		Quality:     0.1,
		Efficiency:  0.1,
	}
	selected := archive.SelectByWeights(accuracyWeights)
	if selected.ID != "accurate" {
		t.Errorf("expected 'accurate' for accuracy weights, got '%s'", selected.ID)
	}

	// Select with efficiency-focused weights
	efficiencyWeights := DimensionWeights{
		SuccessRate: 0.1,
		Quality:     0.1,
		Efficiency:  0.8,
	}
	selected = archive.SelectByWeights(efficiencyWeights)
	if selected.ID != "fast" {
		t.Errorf("expected 'fast' for efficiency weights, got '%s'", selected.ID)
	}
}

func TestParetoArchiveGetBest(t *testing.T) {
	archive := NewParetoArchive(10)

	solution := &EliteSolution{
		ID: "only",
		Scores: DimensionScores{
			SuccessRate: 0.8,
			Quality:     0.7,
		},
	}
	archive.Add(solution)

	best := archive.GetBest()
	if best == nil {
		t.Error("expected best solution, got nil")
	}
	if best.ID != "only" {
		t.Errorf("expected 'only', got '%s'", best.ID)
	}
}

func TestParetoArchiveSelectByCoverage(t *testing.T) {
	archive := NewParetoArchive(10)

	// Create solutions that don't dominate each other (trade-off)
	high := &EliteSolution{
		ID:       "high",
		Coverage: 10,
		Scores:   DimensionScores{SuccessRate: 0.8, Quality: 0.9}, // Better quality
	}
	low := &EliteSolution{
		ID:       "low",
		Coverage: 2,
		Scores:   DimensionScores{SuccessRate: 0.9, Quality: 0.7}, // Better success rate
	}

	archive.Add(high)
	archive.Add(low)

	// Verify both are in archive (trade-off, neither dominates)
	if archive.Size() != 2 {
		t.Errorf("expected 2 solutions in archive, got %d", archive.Size())
	}

	selected := archive.SelectByCoverage()
	if selected.ID != "high" {
		t.Errorf("expected 'high' coverage solution, got '%s'", selected.ID)
	}
}

func TestParetoArchiveUpdateCoverage(t *testing.T) {
	archive := NewParetoArchive(10)

	solution := &EliteSolution{
		ID:       "sol",
		Coverage: 5,
		Scores:   DimensionScores{SuccessRate: 0.8},
	}
	archive.Add(solution)

	archive.UpdateCoverage("sol", 15)

	all := archive.GetAll()
	if all[0].Coverage != 15 {
		t.Errorf("expected coverage 15, got %d", all[0].Coverage)
	}
}

func TestDominates(t *testing.T) {
	tests := []struct {
		name     string
		a, b     DimensionScores
		expected bool
	}{
		{
			name:     "a dominates b",
			a:        DimensionScores{SuccessRate: 0.9, Quality: 0.9, Efficiency: 0.9, Robustness: 0.9, Generalization: 0.9, Diversity: 0.9, Innovation: 0.9},
			b:        DimensionScores{SuccessRate: 0.5, Quality: 0.5, Efficiency: 0.5, Robustness: 0.5, Generalization: 0.5, Diversity: 0.5, Innovation: 0.5},
			expected: true,
		},
		{
			name:     "equal scores - no dominance",
			a:        DimensionScores{SuccessRate: 0.8, Quality: 0.8},
			b:        DimensionScores{SuccessRate: 0.8, Quality: 0.8},
			expected: false,
		},
		{
			name:     "trade-off - no dominance",
			a:        DimensionScores{SuccessRate: 0.9, Quality: 0.5},
			b:        DimensionScores{SuccessRate: 0.5, Quality: 0.9},
			expected: false,
		},
		{
			name:     "a worse in one dimension - no dominance",
			a:        DimensionScores{SuccessRate: 0.9, Quality: 0.4},
			b:        DimensionScores{SuccessRate: 0.5, Quality: 0.5},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dominates(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPruneByDiversity(t *testing.T) {
	solutions := []*EliteSolution{
		{ID: "1", Scores: DimensionScores{SuccessRate: 0.1}},
		{ID: "2", Scores: DimensionScores{SuccessRate: 0.5}},
		{ID: "3", Scores: DimensionScores{SuccessRate: 0.9}},
	}

	// Prune to 2 - should keep boundary points
	pruned := pruneByDiversity(solutions, 2)
	if len(pruned) != 2 {
		t.Errorf("expected 2 solutions, got %d", len(pruned))
	}
}
