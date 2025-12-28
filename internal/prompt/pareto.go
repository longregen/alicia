package prompt

import (
	"sort"
	"sync"
	"time"
)

// EliteSolution represents one optimized prompt in the Pareto archive
type EliteSolution struct {
	ID           string          `json:"id"`
	Instructions string          `json:"instructions"`
	Demos        []Example       `json:"demos"`
	Scores       DimensionScores `json:"scores"`
	Generation   int             `json:"generation"`
	Coverage     int             `json:"coverage"` // Number of examples this solution solves best
	CreatedAt    time.Time       `json:"createdAt"`
}

// ParetoArchive manages the collection of elite solutions
// It maintains a set of non-dominated solutions representing different
// trade-offs across the 7 optimization dimensions
type ParetoArchive struct {
	Solutions []*EliteSolution
	MaxSize   int
	mu        sync.RWMutex
}

// NewParetoArchive creates a new Pareto archive with the given maximum size
func NewParetoArchive(maxSize int) *ParetoArchive {
	if maxSize <= 0 {
		maxSize = 50 // Default archive size
	}
	return &ParetoArchive{
		Solutions: make([]*EliteSolution, 0),
		MaxSize:   maxSize,
	}
}

// Add inserts a solution if it's non-dominated
// Returns true if the solution was added, false if it was dominated
func (p *ParetoArchive) Add(solution *EliteSolution) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if solution is dominated by any existing solution
	for _, existing := range p.Solutions {
		if dominates(existing.Scores, solution.Scores) {
			return false // Dominated, don't add
		}
	}

	// Remove any solutions dominated by the new one
	p.Solutions = filterNonDominated(p.Solutions, solution.Scores)

	// Add new solution
	p.Solutions = append(p.Solutions, solution)

	// Enforce max size using crowding distance
	if len(p.Solutions) > p.MaxSize {
		p.Solutions = pruneByDiversity(p.Solutions, p.MaxSize)
	}

	return true
}

// SelectByWeights returns the best solution for given dimension weights
func (p *ParetoArchive) SelectByWeights(weights DimensionWeights) *EliteSolution {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Solutions) == 0 {
		return nil
	}

	var best *EliteSolution
	bestScore := -1.0

	for _, sol := range p.Solutions {
		score := sol.Scores.WeightedScore(weights)
		if score > bestScore {
			bestScore = score
			best = sol
		}
	}

	return best
}

// SelectByCoverage selects a solution proportional to its coverage
// This implements GEPA's coverage-based selection strategy
func (p *ParetoArchive) SelectByCoverage() *EliteSolution {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Solutions) == 0 {
		return nil
	}

	// Calculate total coverage
	totalCoverage := 0
	for _, sol := range p.Solutions {
		totalCoverage += sol.Coverage
		if sol.Coverage == 0 {
			totalCoverage++ // Ensure non-zero probability
		}
	}

	// Select proportional to coverage (simple deterministic approach)
	// In production, use random selection with weights
	maxCoverage := 0
	var selected *EliteSolution
	for _, sol := range p.Solutions {
		coverage := sol.Coverage
		if coverage == 0 {
			coverage = 1
		}
		if coverage > maxCoverage {
			maxCoverage = coverage
			selected = sol
		}
	}

	return selected
}

// GetAll returns all solutions in the archive
func (p *ParetoArchive) GetAll() []*EliteSolution {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*EliteSolution, len(p.Solutions))
	copy(result, p.Solutions)
	return result
}

// Size returns the number of solutions in the archive
func (p *ParetoArchive) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.Solutions)
}

// GetBest returns the solution with the highest weighted score for default weights
func (p *ParetoArchive) GetBest() *EliteSolution {
	return p.SelectByWeights(DefaultWeights())
}

// UpdateCoverage updates the coverage count for a solution
func (p *ParetoArchive) UpdateCoverage(solutionID string, coverage int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, sol := range p.Solutions {
		if sol.ID == solutionID {
			sol.Coverage = coverage
			return
		}
	}
}

// dominates checks if solution A dominates solution B
// A dominates B if A is at least as good as B in all dimensions
// and strictly better in at least one dimension
func dominates(a, b DimensionScores) bool {
	aScores := []float64{a.SuccessRate, a.Quality, a.Efficiency, a.Robustness, a.Generalization, a.Diversity, a.Innovation}
	bScores := []float64{b.SuccessRate, b.Quality, b.Efficiency, b.Robustness, b.Generalization, b.Diversity, b.Innovation}

	atLeastAsGood := true
	strictlyBetter := false

	for i := 0; i < len(aScores); i++ {
		if aScores[i] < bScores[i] {
			atLeastAsGood = false
			break
		}
		if aScores[i] > bScores[i] {
			strictlyBetter = true
		}
	}

	return atLeastAsGood && strictlyBetter
}

// filterNonDominated removes solutions dominated by the new solution's scores
func filterNonDominated(solutions []*EliteSolution, newScores DimensionScores) []*EliteSolution {
	result := make([]*EliteSolution, 0, len(solutions))
	for _, sol := range solutions {
		if !dominates(newScores, sol.Scores) {
			result = append(result, sol)
		}
	}
	return result
}

// pruneByDiversity removes solutions to maintain diversity using crowding distance
func pruneByDiversity(solutions []*EliteSolution, maxSize int) []*EliteSolution {
	if len(solutions) <= maxSize {
		return solutions
	}

	// Calculate crowding distance for each solution
	distances := calculateCrowdingDistances(solutions)

	// Create index-distance pairs and sort by distance (descending)
	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(solutions))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist // Higher distance = more diverse = keep
	})

	// Keep the most diverse solutions
	result := make([]*EliteSolution, maxSize)
	for i := 0; i < maxSize; i++ {
		result[i] = solutions[pairs[i].idx]
	}
	return result
}

// calculateCrowdingDistances computes NSGA-II style crowding distances
func calculateCrowdingDistances(solutions []*EliteSolution) []float64 {
	n := len(solutions)
	if n == 0 {
		return nil
	}

	distances := make([]float64, n)

	// For each dimension, sort and calculate partial distances
	dimensions := []func(*DimensionScores) float64{
		func(s *DimensionScores) float64 { return s.SuccessRate },
		func(s *DimensionScores) float64 { return s.Quality },
		func(s *DimensionScores) float64 { return s.Efficiency },
		func(s *DimensionScores) float64 { return s.Robustness },
		func(s *DimensionScores) float64 { return s.Generalization },
		func(s *DimensionScores) float64 { return s.Diversity },
		func(s *DimensionScores) float64 { return s.Innovation },
	}

	for _, getDim := range dimensions {
		// Create sorted indices for this dimension
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool {
			return getDim(&solutions[indices[i]].Scores) < getDim(&solutions[indices[j]].Scores)
		})

		// Get min and max for normalization
		minVal := getDim(&solutions[indices[0]].Scores)
		maxVal := getDim(&solutions[indices[n-1]].Scores)
		dimRange := maxVal - minVal

		if dimRange == 0 {
			continue // All same value, no contribution to distance
		}

		// Boundary points get infinite distance (always kept)
		distances[indices[0]] = 1e9
		distances[indices[n-1]] = 1e9

		// Interior points get distance based on neighbors
		for i := 1; i < n-1; i++ {
			neighborDist := getDim(&solutions[indices[i+1]].Scores) - getDim(&solutions[indices[i-1]].Scores)
			distances[indices[i]] += neighborDist / dimRange
		}
	}

	return distances
}
