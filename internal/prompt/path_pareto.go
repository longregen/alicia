package prompt

import (
	"sort"
	"sync"

	"github.com/longregen/alicia/internal/domain/models"
)

// PathParetoArchive maintains non-dominated execution paths for GEPA Path Search.
// It uses Pareto dominance across 5 dimensions (AnswerQuality, Efficiency, TokenCost,
// Robustness, Latency) to maintain a diverse set of high-performing paths.
type PathParetoArchive struct {
	candidates []*models.PathCandidate
	maxSize    int
	mu         sync.RWMutex
}

// NewPathParetoArchive creates a new Pareto archive with the given maximum size
func NewPathParetoArchive(maxSize int) *PathParetoArchive {
	if maxSize <= 0 {
		maxSize = 50 // Default archive size
	}
	return &PathParetoArchive{
		candidates: make([]*models.PathCandidate, 0),
		maxSize:    maxSize,
	}
}

// Add inserts a candidate if it's non-dominated.
// Removes any existing candidates that the new one dominates.
// If the archive exceeds maxSize, prunes using crowding distance.
func (a *PathParetoArchive) Add(candidate *models.PathCandidate) {
	if candidate == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if dominated by any existing candidate
	for _, existing := range a.candidates {
		if a.dominates(existing.Scores, candidate.Scores) {
			return // Dominated, don't add
		}
	}

	// Remove any candidates dominated by the new one
	a.candidates = a.filterNonDominated(candidate.Scores)

	// Add new candidate
	a.candidates = append(a.candidates, candidate)

	// Prune using crowding distance if too large
	if len(a.candidates) > a.maxSize {
		a.pruneWithCrowding()
	}
}

// dominates returns true if scoreA dominates scoreB.
// A dominates B if A is at least as good as B in ALL 5 dimensions
// AND strictly better in at least one dimension.
func (a *PathParetoArchive) dominates(scoreA, scoreB models.PathScores) bool {
	aVals := []float64{
		scoreA.AnswerQuality,
		scoreA.Efficiency,
		scoreA.TokenCost,
		scoreA.Robustness,
		scoreA.Latency,
	}
	bVals := []float64{
		scoreB.AnswerQuality,
		scoreB.Efficiency,
		scoreB.TokenCost,
		scoreB.Robustness,
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
func (a *PathParetoArchive) filterNonDominated(newScores models.PathScores) []*models.PathCandidate {
	result := make([]*models.PathCandidate, 0, len(a.candidates))
	for _, c := range a.candidates {
		if !a.dominates(newScores, c.Scores) {
			result = append(result, c)
		}
	}
	return result
}

// SelectForMutation picks n candidates from the archive for mutation.
// Uses crowding distance to maintain diversity in selection.
func (a *PathParetoArchive) SelectForMutation(n int) []*models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.candidates) == 0 {
		return nil
	}

	if n <= 0 {
		return nil
	}

	if n >= len(a.candidates) {
		// Return a copy of all candidates
		result := make([]*models.PathCandidate, len(a.candidates))
		copy(result, a.candidates)
		return result
	}

	return a.selectByCrowding(n)
}

// pruneWithCrowding removes the least diverse candidates when archive exceeds maxSize.
// Uses NSGA-II style crowding distance to preserve diversity.
func (a *PathParetoArchive) pruneWithCrowding() {
	if len(a.candidates) <= a.maxSize {
		return
	}

	// Calculate crowding distance for each candidate
	distances := a.calculateCrowdingDistances()

	// Create index-distance pairs and sort by distance (descending)
	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(a.candidates))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist // Higher distance = more diverse = keep
	})

	// Keep the most diverse candidates
	result := make([]*models.PathCandidate, a.maxSize)
	for i := 0; i < a.maxSize; i++ {
		result[i] = a.candidates[pairs[i].idx]
	}
	a.candidates = result
}

// selectByCrowding selects n candidates using crowding distance for diversity.
// Candidates with higher crowding distance (more isolated in objective space) are preferred.
func (a *PathParetoArchive) selectByCrowding(n int) []*models.PathCandidate {
	if n <= 0 || len(a.candidates) == 0 {
		return nil
	}

	if n >= len(a.candidates) {
		result := make([]*models.PathCandidate, len(a.candidates))
		copy(result, a.candidates)
		return result
	}

	// Calculate crowding distance for each candidate
	distances := a.calculateCrowdingDistances()

	// Create index-distance pairs and sort by distance (descending)
	type indexDist struct {
		idx  int
		dist float64
	}
	pairs := make([]indexDist, len(a.candidates))
	for i, dist := range distances {
		pairs[i] = indexDist{idx: i, dist: dist}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist > pairs[j].dist // Higher distance = more diverse
	})

	// Select top n by crowding distance
	result := make([]*models.PathCandidate, n)
	for i := 0; i < n; i++ {
		result[i] = a.candidates[pairs[i].idx]
	}
	return result
}

// calculateCrowdingDistances computes NSGA-II style crowding distances.
// For each objective, candidates are sorted and assigned distance based on
// their neighbors' positions. Distances are summed across all objectives.
// Boundary points (min/max on any objective) get infinite distance.
func (a *PathParetoArchive) calculateCrowdingDistances() []float64 {
	n := len(a.candidates)
	if n == 0 {
		return nil
	}

	distances := make([]float64, n)

	// Define dimension extractors for all 5 PathScores dimensions
	dimensions := []func(*models.PathScores) float64{
		func(s *models.PathScores) float64 { return s.AnswerQuality },
		func(s *models.PathScores) float64 { return s.Efficiency },
		func(s *models.PathScores) float64 { return s.TokenCost },
		func(s *models.PathScores) float64 { return s.Robustness },
		func(s *models.PathScores) float64 { return s.Latency },
	}

	for _, getDim := range dimensions {
		// Create sorted indices for this dimension
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool {
			return getDim(&a.candidates[indices[i]].Scores) < getDim(&a.candidates[indices[j]].Scores)
		})

		// Get min and max for normalization
		minVal := getDim(&a.candidates[indices[0]].Scores)
		maxVal := getDim(&a.candidates[indices[n-1]].Scores)
		dimRange := maxVal - minVal

		if dimRange == 0 {
			continue // All same value, no contribution to distance
		}

		// Boundary points get infinite distance (always kept)
		distances[indices[0]] = 1e9
		distances[indices[n-1]] = 1e9

		// Interior points get distance based on neighbors
		for i := 1; i < n-1; i++ {
			neighborDist := getDim(&a.candidates[indices[i+1]].Scores) - getDim(&a.candidates[indices[i-1]].Scores)
			distances[indices[i]] += neighborDist / dimRange
		}
	}

	return distances
}

// GetBestByQuality returns the candidate with the highest AnswerQuality score.
// Returns nil if the archive is empty.
func (a *PathParetoArchive) GetBestByQuality() *models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.candidates) == 0 {
		return nil
	}

	var best *models.PathCandidate
	bestQuality := -1.0

	for _, c := range a.candidates {
		if c.Scores.AnswerQuality > bestQuality {
			bestQuality = c.Scores.AnswerQuality
			best = c
		}
	}

	return best
}

// GetParetoFront returns a copy of all candidates in the archive.
// All candidates in the archive are, by definition, on the Pareto front
// (none is dominated by another).
func (a *PathParetoArchive) GetParetoFront() []*models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*models.PathCandidate, len(a.candidates))
	copy(result, a.candidates)
	return result
}

// Size returns the number of candidates in the archive
func (a *PathParetoArchive) Size() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.candidates)
}

// GetByID returns the candidate with the given ID, or nil if not found
func (a *PathParetoArchive) GetByID(id string) *models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, c := range a.candidates {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// GetBestByWeightedScore returns the candidate with the highest weighted score
// using the provided weights for each dimension.
func (a *PathParetoArchive) GetBestByWeightedScore(weights PathWeights) *models.PathCandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.candidates) == 0 {
		return nil
	}

	var best *models.PathCandidate
	bestScore := -1.0

	for _, c := range a.candidates {
		score := weights.WeightedScore(c.Scores)
		if score > bestScore {
			bestScore = score
			best = c
		}
	}

	return best
}

// Clear removes all candidates from the archive
func (a *PathParetoArchive) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.candidates = make([]*models.PathCandidate, 0)
}

// PathWeights defines weights for each PathScores dimension
type PathWeights struct {
	AnswerQuality float64
	Efficiency    float64
	TokenCost     float64
	Robustness    float64
	Latency       float64
}

// DefaultPathWeights returns balanced weights for path scoring
func DefaultPathWeights() PathWeights {
	return PathWeights{
		AnswerQuality: 0.40, // Primary objective
		Efficiency:    0.15,
		TokenCost:     0.15,
		Robustness:    0.15,
		Latency:       0.15,
	}
}

// WeightedScore calculates the weighted score for the given PathScores
func (w PathWeights) WeightedScore(scores models.PathScores) float64 {
	return scores.AnswerQuality*w.AnswerQuality +
		scores.Efficiency*w.Efficiency +
		scores.TokenCost*w.TokenCost +
		scores.Robustness*w.Robustness +
		scores.Latency*w.Latency
}

// Normalize ensures weights sum to 1.0
func (w *PathWeights) Normalize() {
	sum := w.AnswerQuality + w.Efficiency + w.TokenCost + w.Robustness + w.Latency
	if sum > 0 {
		w.AnswerQuality /= sum
		w.Efficiency /= sum
		w.TokenCost /= sum
		w.Robustness /= sum
		w.Latency /= sum
	}
}
