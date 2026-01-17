package prompt

import (
	"sync"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathParetoArchive(t *testing.T) {
	t.Run("with custom size", func(t *testing.T) {
		archive := NewPathParetoArchive(100)
		assert.NotNil(t, archive)
		assert.Equal(t, 100, archive.maxSize)
		assert.Equal(t, 0, archive.Size())
	})

	t.Run("with zero size uses default", func(t *testing.T) {
		archive := NewPathParetoArchive(0)
		assert.NotNil(t, archive)
		assert.Equal(t, 50, archive.maxSize, "should use default size of 50")
	})

	t.Run("with negative size uses default", func(t *testing.T) {
		archive := NewPathParetoArchive(-10)
		assert.NotNil(t, archive)
		assert.Equal(t, 50, archive.maxSize, "should use default size of 50")
	})
}

func TestPathParetoArchive_Add(t *testing.T) {
	t.Run("add single candidate", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		candidate := &models.PathCandidate{
			ID: "test-1",
			Scores: models.PathScores{
				AnswerQuality: 0.8,
				Efficiency:    0.7,
				TokenCost:     0.6,
				Robustness:    0.5,
				Latency:       0.4,
			},
		}

		archive.Add(candidate)
		assert.Equal(t, 1, archive.Size())
	})

	t.Run("add nil candidate does nothing", func(t *testing.T) {
		archive := NewPathParetoArchive(10)
		archive.Add(nil)
		assert.Equal(t, 0, archive.Size())
	})

	t.Run("dominated candidate is not added", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		dominant := &models.PathCandidate{
			ID: "dominant",
			Scores: models.PathScores{
				AnswerQuality: 0.9,
				Efficiency:    0.9,
				TokenCost:     0.9,
				Robustness:    0.9,
				Latency:       0.9,
			},
		}
		archive.Add(dominant)

		dominated := &models.PathCandidate{
			ID: "dominated",
			Scores: models.PathScores{
				AnswerQuality: 0.5,
				Efficiency:    0.5,
				TokenCost:     0.5,
				Robustness:    0.5,
				Latency:       0.5,
			},
		}
		archive.Add(dominated)

		assert.Equal(t, 1, archive.Size())
		assert.Equal(t, "dominant", archive.GetParetoFront()[0].ID)
	})

	t.Run("new candidate removes dominated ones", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		weaker := &models.PathCandidate{
			ID: "weaker",
			Scores: models.PathScores{
				AnswerQuality: 0.5,
				Efficiency:    0.5,
				TokenCost:     0.5,
				Robustness:    0.5,
				Latency:       0.5,
			},
		}
		archive.Add(weaker)
		assert.Equal(t, 1, archive.Size())

		stronger := &models.PathCandidate{
			ID: "stronger",
			Scores: models.PathScores{
				AnswerQuality: 0.9,
				Efficiency:    0.9,
				TokenCost:     0.9,
				Robustness:    0.9,
				Latency:       0.9,
			},
		}
		archive.Add(stronger)

		assert.Equal(t, 1, archive.Size())
		assert.Equal(t, "stronger", archive.GetParetoFront()[0].ID)
	})

	t.Run("non-dominated candidates are both kept (trade-off)", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		// High quality but low efficiency
		highQuality := &models.PathCandidate{
			ID: "high-quality",
			Scores: models.PathScores{
				AnswerQuality: 0.95,
				Efficiency:    0.3,
				TokenCost:     0.4,
				Robustness:    0.5,
				Latency:       0.4,
			},
		}
		archive.Add(highQuality)

		// High efficiency but lower quality
		highEfficiency := &models.PathCandidate{
			ID: "high-efficiency",
			Scores: models.PathScores{
				AnswerQuality: 0.6,
				Efficiency:    0.95,
				TokenCost:     0.9,
				Robustness:    0.8,
				Latency:       0.9,
			},
		}
		archive.Add(highEfficiency)

		assert.Equal(t, 2, archive.Size(), "both non-dominated candidates should be kept")
	})

	t.Run("prune when exceeding maxSize", func(t *testing.T) {
		archive := NewPathParetoArchive(3)

		// Add multiple non-dominated candidates to trigger pruning
		for i := 0; i < 5; i++ {
			candidate := &models.PathCandidate{
				ID: string(rune('a' + i)),
				Scores: models.PathScores{
					AnswerQuality: 0.5 + float64(i)*0.1,
					Efficiency:    0.9 - float64(i)*0.1,
					TokenCost:     0.5,
					Robustness:    0.5,
					Latency:       0.5,
				},
			}
			archive.Add(candidate)
		}

		assert.LessOrEqual(t, archive.Size(), 3, "archive size should not exceed maxSize")
	})
}

func TestPathParetoArchive_Dominates(t *testing.T) {
	archive := NewPathParetoArchive(10)

	tests := []struct {
		name     string
		scoreA   models.PathScores
		scoreB   models.PathScores
		expected bool
	}{
		{
			name: "A dominates B - all better",
			scoreA: models.PathScores{
				AnswerQuality: 0.9,
				Efficiency:    0.9,
				TokenCost:     0.9,
				Robustness:    0.9,
				Latency:       0.9,
			},
			scoreB: models.PathScores{
				AnswerQuality: 0.5,
				Efficiency:    0.5,
				TokenCost:     0.5,
				Robustness:    0.5,
				Latency:       0.5,
			},
			expected: true,
		},
		{
			name: "A dominates B - equal in some, better in one",
			scoreA: models.PathScores{
				AnswerQuality: 0.9,
				Efficiency:    0.8,
				TokenCost:     0.8,
				Robustness:    0.8,
				Latency:       0.8,
			},
			scoreB: models.PathScores{
				AnswerQuality: 0.8,
				Efficiency:    0.8,
				TokenCost:     0.8,
				Robustness:    0.8,
				Latency:       0.8,
			},
			expected: true,
		},
		{
			name: "no dominance - equal scores",
			scoreA: models.PathScores{
				AnswerQuality: 0.8,
				Efficiency:    0.8,
				TokenCost:     0.8,
				Robustness:    0.8,
				Latency:       0.8,
			},
			scoreB: models.PathScores{
				AnswerQuality: 0.8,
				Efficiency:    0.8,
				TokenCost:     0.8,
				Robustness:    0.8,
				Latency:       0.8,
			},
			expected: false,
		},
		{
			name: "no dominance - trade-off",
			scoreA: models.PathScores{
				AnswerQuality: 0.95,
				Efficiency:    0.3,
				TokenCost:     0.5,
				Robustness:    0.5,
				Latency:       0.5,
			},
			scoreB: models.PathScores{
				AnswerQuality: 0.3,
				Efficiency:    0.95,
				TokenCost:     0.5,
				Robustness:    0.5,
				Latency:       0.5,
			},
			expected: false,
		},
		{
			name: "no dominance - A worse in one dimension",
			scoreA: models.PathScores{
				AnswerQuality: 0.9,
				Efficiency:    0.9,
				TokenCost:     0.4, // Worse than B
				Robustness:    0.9,
				Latency:       0.9,
			},
			scoreB: models.PathScores{
				AnswerQuality: 0.5,
				Efficiency:    0.5,
				TokenCost:     0.5,
				Robustness:    0.5,
				Latency:       0.5,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := archive.dominates(tt.scoreA, tt.scoreB)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathParetoArchive_SelectForMutation(t *testing.T) {
	t.Run("empty archive returns nil", func(t *testing.T) {
		archive := NewPathParetoArchive(10)
		selected := archive.SelectForMutation(3)
		assert.Nil(t, selected)
	})

	t.Run("n <= 0 returns nil", func(t *testing.T) {
		archive := NewPathParetoArchive(10)
		archive.Add(&models.PathCandidate{ID: "test", Scores: models.PathScores{AnswerQuality: 0.5}})

		assert.Nil(t, archive.SelectForMutation(0))
		assert.Nil(t, archive.SelectForMutation(-1))
	})

	t.Run("n >= archive size returns all candidates", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		for i := 0; i < 3; i++ {
			archive.Add(&models.PathCandidate{
				ID: string(rune('a' + i)),
				Scores: models.PathScores{
					AnswerQuality: 0.5 + float64(i)*0.1,
					Efficiency:    0.8 - float64(i)*0.1,
					TokenCost:     0.5,
					Robustness:    0.5,
					Latency:       0.5,
				},
			})
		}

		selected := archive.SelectForMutation(10)
		assert.Equal(t, 3, len(selected))
	})

	t.Run("selects n candidates using crowding distance", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		// Add candidates with different trade-offs
		for i := 0; i < 5; i++ {
			archive.Add(&models.PathCandidate{
				ID: string(rune('a' + i)),
				Scores: models.PathScores{
					AnswerQuality: 0.5 + float64(i)*0.1,
					Efficiency:    0.9 - float64(i)*0.1,
					TokenCost:     0.5,
					Robustness:    0.5,
					Latency:       0.5,
				},
			})
		}

		selected := archive.SelectForMutation(2)
		require.Equal(t, 2, len(selected))

		// Boundary candidates (min/max on dimensions) should be preferred
		// due to infinite crowding distance
	})
}

func TestPathParetoArchive_GetBestByQuality(t *testing.T) {
	t.Run("empty archive returns nil", func(t *testing.T) {
		archive := NewPathParetoArchive(10)
		assert.Nil(t, archive.GetBestByQuality())
	})

	t.Run("returns candidate with highest AnswerQuality", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		// Add with different quality but still non-dominated (trade-offs)
		low := &models.PathCandidate{
			ID:     "low",
			Scores: models.PathScores{AnswerQuality: 0.5, Efficiency: 0.9, TokenCost: 0.9, Robustness: 0.9, Latency: 0.9},
		}
		medium := &models.PathCandidate{
			ID:     "medium",
			Scores: models.PathScores{AnswerQuality: 0.7, Efficiency: 0.7, TokenCost: 0.7, Robustness: 0.7, Latency: 0.7},
		}
		high := &models.PathCandidate{
			ID:     "high",
			Scores: models.PathScores{AnswerQuality: 0.9, Efficiency: 0.5, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5},
		}

		archive.Add(low)
		archive.Add(medium)
		archive.Add(high)

		best := archive.GetBestByQuality()
		require.NotNil(t, best)
		assert.Equal(t, "high", best.ID)
		assert.Equal(t, 0.9, best.Scores.AnswerQuality)
	})
}

func TestPathParetoArchive_Size(t *testing.T) {
	archive := NewPathParetoArchive(10)
	assert.Equal(t, 0, archive.Size())

	// Add first candidate with low quality but high efficiency
	archive.Add(&models.PathCandidate{ID: "1", Scores: models.PathScores{AnswerQuality: 0.5, Efficiency: 0.9, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}})
	assert.Equal(t, 1, archive.Size())

	// Add second candidate with high quality but low efficiency (trade-off, neither dominates)
	archive.Add(&models.PathCandidate{ID: "2", Scores: models.PathScores{AnswerQuality: 0.9, Efficiency: 0.4, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}})
	assert.Equal(t, 2, archive.Size())
}

func TestPathParetoArchive_GetParetoFront(t *testing.T) {
	t.Run("empty archive returns empty slice", func(t *testing.T) {
		archive := NewPathParetoArchive(10)
		front := archive.GetParetoFront()
		assert.NotNil(t, front)
		assert.Len(t, front, 0)
	})

	t.Run("returns copy of all candidates", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		c1 := &models.PathCandidate{ID: "1", Scores: models.PathScores{AnswerQuality: 0.9, Efficiency: 0.5}}
		c2 := &models.PathCandidate{ID: "2", Scores: models.PathScores{AnswerQuality: 0.5, Efficiency: 0.9}}

		archive.Add(c1)
		archive.Add(c2)

		front := archive.GetParetoFront()
		assert.Len(t, front, 2)

		// Verify it's a copy (modifying returned slice doesn't affect archive)
		front[0] = nil
		assert.NotNil(t, archive.GetParetoFront()[0])
	})
}

func TestPathParetoArchive_GetByID(t *testing.T) {
	archive := NewPathParetoArchive(10)

	candidate := &models.PathCandidate{
		ID:     "test-id",
		Scores: models.PathScores{AnswerQuality: 0.8},
	}
	archive.Add(candidate)

	t.Run("found", func(t *testing.T) {
		found := archive.GetByID("test-id")
		require.NotNil(t, found)
		assert.Equal(t, "test-id", found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		notFound := archive.GetByID("nonexistent")
		assert.Nil(t, notFound)
	})
}

func TestPathParetoArchive_GetBestByWeightedScore(t *testing.T) {
	t.Run("empty archive returns nil", func(t *testing.T) {
		archive := NewPathParetoArchive(10)
		weights := DefaultPathWeights()
		assert.Nil(t, archive.GetBestByWeightedScore(weights))
	})

	t.Run("returns best according to weights", func(t *testing.T) {
		archive := NewPathParetoArchive(10)

		qualityFocused := &models.PathCandidate{
			ID: "quality",
			Scores: models.PathScores{
				AnswerQuality: 0.95,
				Efficiency:    0.3,
				TokenCost:     0.3,
				Robustness:    0.3,
				Latency:       0.3,
			},
		}
		efficiencyFocused := &models.PathCandidate{
			ID: "efficiency",
			Scores: models.PathScores{
				AnswerQuality: 0.5,
				Efficiency:    0.95,
				TokenCost:     0.95,
				Robustness:    0.95,
				Latency:       0.95,
			},
		}

		archive.Add(qualityFocused)
		archive.Add(efficiencyFocused)

		// With default weights (0.4 for quality, 0.15 for others)
		// Quality: 0.95*0.4 + 0.3*0.6 = 0.38 + 0.18 = 0.56
		// Efficiency: 0.5*0.4 + 0.95*0.6 = 0.20 + 0.57 = 0.77
		best := archive.GetBestByWeightedScore(DefaultPathWeights())
		require.NotNil(t, best)
		assert.Equal(t, "efficiency", best.ID)

		// With quality-focused weights
		qualityWeights := PathWeights{
			AnswerQuality: 0.8,
			Efficiency:    0.05,
			TokenCost:     0.05,
			Robustness:    0.05,
			Latency:       0.05,
		}
		best = archive.GetBestByWeightedScore(qualityWeights)
		require.NotNil(t, best)
		assert.Equal(t, "quality", best.ID)
	})
}

func TestPathParetoArchive_Clear(t *testing.T) {
	archive := NewPathParetoArchive(10)

	// Add candidates that don't dominate each other (trade-offs)
	archive.Add(&models.PathCandidate{ID: "1", Scores: models.PathScores{AnswerQuality: 0.5, Efficiency: 0.9, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}})
	archive.Add(&models.PathCandidate{ID: "2", Scores: models.PathScores{AnswerQuality: 0.9, Efficiency: 0.4, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}})
	assert.Equal(t, 2, archive.Size())

	archive.Clear()
	assert.Equal(t, 0, archive.Size())
}

func TestPathParetoArchive_ConcurrentAccess(t *testing.T) {
	archive := NewPathParetoArchive(100)

	var wg sync.WaitGroup
	numGoroutines := 10
	candidatesPerGoroutine := 10

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < candidatesPerGoroutine; i++ {
				candidate := &models.PathCandidate{
					ID: string(rune('a'+gid)) + string(rune('0'+i)),
					Scores: models.PathScores{
						AnswerQuality: float64(gid) / float64(numGoroutines),
						Efficiency:    float64(i) / float64(candidatesPerGoroutine),
						TokenCost:     0.5,
						Robustness:    0.5,
						Latency:       0.5,
					},
				}
				archive.Add(candidate)
			}
		}(g)
	}

	wg.Wait()

	// Archive should have added candidates without panicking
	// Size depends on dominance relationships
	assert.GreaterOrEqual(t, archive.Size(), 1)
}

func TestPathWeights(t *testing.T) {
	t.Run("DefaultPathWeights", func(t *testing.T) {
		weights := DefaultPathWeights()
		assert.Equal(t, 0.40, weights.AnswerQuality)
		assert.Equal(t, 0.15, weights.Efficiency)
		assert.Equal(t, 0.15, weights.TokenCost)
		assert.Equal(t, 0.15, weights.Robustness)
		assert.Equal(t, 0.15, weights.Latency)
	})

	t.Run("WeightedScore", func(t *testing.T) {
		weights := PathWeights{
			AnswerQuality: 0.5,
			Efficiency:    0.1,
			TokenCost:     0.1,
			Robustness:    0.2,
			Latency:       0.1,
		}

		scores := models.PathScores{
			AnswerQuality: 1.0,
			Efficiency:    0.8,
			TokenCost:     0.6,
			Robustness:    0.9,
			Latency:       0.7,
		}

		// 1.0*0.5 + 0.8*0.1 + 0.6*0.1 + 0.9*0.2 + 0.7*0.1
		// = 0.5 + 0.08 + 0.06 + 0.18 + 0.07 = 0.89
		expected := 0.89
		actual := weights.WeightedScore(scores)
		assert.InDelta(t, expected, actual, 0.001)
	})

	t.Run("Normalize", func(t *testing.T) {
		weights := PathWeights{
			AnswerQuality: 4.0,
			Efficiency:    1.0,
			TokenCost:     1.0,
			Robustness:    2.0,
			Latency:       2.0,
		}

		weights.Normalize()

		sum := weights.AnswerQuality + weights.Efficiency + weights.TokenCost + weights.Robustness + weights.Latency
		assert.InDelta(t, 1.0, sum, 0.001)
		assert.InDelta(t, 0.4, weights.AnswerQuality, 0.001)
		assert.InDelta(t, 0.1, weights.Efficiency, 0.001)
	})

	t.Run("Normalize with zero sum", func(t *testing.T) {
		weights := PathWeights{}
		weights.Normalize() // Should not panic
		assert.Equal(t, 0.0, weights.AnswerQuality)
	})
}

func TestPathParetoArchive_CrowdingDistance(t *testing.T) {
	archive := NewPathParetoArchive(10)

	// Add candidates that span the range of a dimension
	// Boundary points should get high crowding distance
	candidates := []*models.PathCandidate{
		{ID: "min", Scores: models.PathScores{AnswerQuality: 0.1, Efficiency: 0.9, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}},
		{ID: "mid1", Scores: models.PathScores{AnswerQuality: 0.4, Efficiency: 0.6, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}},
		{ID: "mid2", Scores: models.PathScores{AnswerQuality: 0.5, Efficiency: 0.5, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}},
		{ID: "mid3", Scores: models.PathScores{AnswerQuality: 0.6, Efficiency: 0.4, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}},
		{ID: "max", Scores: models.PathScores{AnswerQuality: 0.9, Efficiency: 0.1, TokenCost: 0.5, Robustness: 0.5, Latency: 0.5}},
	}

	for _, c := range candidates {
		archive.Add(c)
	}

	// Select 2 - boundary points should be preferred
	selected := archive.SelectForMutation(2)
	require.Len(t, selected, 2)

	ids := make(map[string]bool)
	for _, s := range selected {
		ids[s.ID] = true
	}

	// "min" and "max" have infinite crowding distance on AnswerQuality/Efficiency
	assert.True(t, ids["min"] || ids["max"], "at least one boundary point should be selected")
}
