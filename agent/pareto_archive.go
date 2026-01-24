package main

import (
	"sort"
	"sync"
)

// PathParetoArchive maintains non-dominated execution paths for Pareto selection.
// It uses Pareto dominance across 5 dimensions (AnswerQuality, Efficiency, TokenCost,
// Robustness, Latency) to maintain a diverse set of high-performing paths.
type PathParetoArchive struct {
	candidates []*PathCandidate
	maxSize    int
	mu         sync.RWMutex
}

// NewPathParetoArchive creates a new Pareto archive with the given maximum size
func NewPathParetoArchive(maxSize int) *PathParetoArchive {
	if maxSize <= 0 {
		maxSize = 50
	}
	return &PathParetoArchive{
		candidates: make([]*PathCandidate, 0),
		maxSize:    maxSize,
	}
}

// Add inserts a candidate if it's non-dominated.
// Removes any existing candidates that the new one dominates.
// If the archive exceeds maxSize, prunes using crowding distance.
func (a *PathParetoArchive) Add(candidate *PathCandidate) {
	if candidate == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for _, existing := range a.candidates {
		if a.dominates(existing.Scores, candidate.Scores) {
			return
		}
	}
	a.candidates = a.filterNonDominated(candidate.Scores)
	a.candidates = append(a.candidates, candidate)
	if len(a.candidates) > a.maxSize {
		a.pruneWithCrowding()
	}
}

// dominates returns true if scoreA dominates scoreB
func (a *PathParetoArchive) dominates(scoreA, scoreB PathScores) bool {
	aVals := []float64{
		scoreA.Effectiveness,
		scoreA.AnswerQuality,
		scoreA.Hallucination,
		scoreA.Specificity,
		scoreA.TokenCost,
		scoreA.Latency,
	}
	bVals := []float64{
		scoreB.Effectiveness,
		scoreB.AnswerQuality,
		scoreB.Hallucination,
		scoreB.Specificity,
		scoreB.TokenCost,
		scoreB.Latency,
	}

	atLeastAsGood := true
	strictlyBetter := false

	for i := 0; i < len(aVals); i++ {
		if aVals[i] < bVals[i] {
			atLeastAsGood = false
			break
		}
		if aVals[i] > bVals[i] {
			strictlyBetter = true
		}
	}

	return atLeastAsGood && strictlyBetter
}

// filterNonDominated removes candidates dominated by the given scores
func (a *PathParetoArchive) filterNonDominated(newScores PathScores) []*PathCandidate {
	result := make([]*PathCandidate, 0, len(a.candidates))
	for _, c := range a.candidates {
		if !a.dominates(newScores, c.Scores) {
			result = append(result, c)
		}
	}
	return result
}

// SelectForMutation picks n candidates from the archive for mutation
func (a *PathParetoArchive) SelectForMutation(n int) []*PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.candidates) == 0 || n <= 0 {
		return nil
	}

	if n >= len(a.candidates) {
		result := make([]*PathCandidate, len(a.candidates))
		copy(result, a.candidates)
		return result
	}

	return a.selectByCrowding(n)
}

// pruneWithCrowding removes the least diverse candidates when archive exceeds maxSize
func (a *PathParetoArchive) pruneWithCrowding() {
	if len(a.candidates) <= a.maxSize {
		return
	}

	distances := a.calculateCrowdingDistances()

	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(a.candidates))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist
	})

	result := make([]*PathCandidate, a.maxSize)
	for i := 0; i < a.maxSize; i++ {
		result[i] = a.candidates[pairs[i].idx]
	}
	a.candidates = result
}

// selectByCrowding selects n candidates using crowding distance for diversity
func (a *PathParetoArchive) selectByCrowding(n int) []*PathCandidate {
	if n <= 0 || len(a.candidates) == 0 {
		return nil
	}

	if n >= len(a.candidates) {
		result := make([]*PathCandidate, len(a.candidates))
		copy(result, a.candidates)
		return result
	}

	distances := a.calculateCrowdingDistances()

	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(a.candidates))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist
	})

	result := make([]*PathCandidate, n)
	for i := 0; i < n; i++ {
		result[i] = a.candidates[pairs[i].idx]
	}
	return result
}

// calculateCrowdingDistances computes NSGA-II style crowding distances
func (a *PathParetoArchive) calculateCrowdingDistances() []float64 {
	n := len(a.candidates)
	if n == 0 {
		return nil
	}

	distances := make([]float64, n)

	dimensions := []func(*PathScores) float64{
		func(s *PathScores) float64 { return s.Effectiveness },
		func(s *PathScores) float64 { return s.AnswerQuality },
		func(s *PathScores) float64 { return s.Hallucination },
		func(s *PathScores) float64 { return s.Specificity },
		func(s *PathScores) float64 { return s.TokenCost },
		func(s *PathScores) float64 { return s.Latency },
	}

	for _, getDim := range dimensions {
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool {
			return getDim(&a.candidates[indices[i]].Scores) < getDim(&a.candidates[indices[j]].Scores)
		})

		minVal := getDim(&a.candidates[indices[0]].Scores)
		maxVal := getDim(&a.candidates[indices[n-1]].Scores)
		dimRange := maxVal - minVal

		if dimRange == 0 {
			continue
		}

		// Boundary points get infinite distance
		distances[indices[0]] = 1e9
		distances[indices[n-1]] = 1e9

		for i := 1; i < n-1; i++ {
			neighborDist := getDim(&a.candidates[indices[i+1]].Scores) - getDim(&a.candidates[indices[i-1]].Scores)
			distances[indices[i]] += neighborDist / dimRange
		}
	}

	return distances
}

// GetParetoFront returns a copy of all candidates in the archive
func (a *PathParetoArchive) GetParetoFront() []*PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*PathCandidate, len(a.candidates))
	copy(result, a.candidates)
	return result
}

// Size returns the number of candidates in the archive
func (a *PathParetoArchive) Size() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.candidates)
}

// GetBestByWeightedSum returns the candidate with the highest weighted sum score
func (a *PathParetoArchive) GetBestByWeightedSum(weights PathScoreWeights) *PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.candidates) == 0 {
		return nil
	}

	var best *PathCandidate
	bestScore := -1.0

	for _, c := range a.candidates {
		score := c.Scores.WeightedSum(weights)
		if score > bestScore {
			bestScore = score
			best = c
		}
	}

	return best
}
