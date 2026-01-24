package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockLLM simulates LLM responses for testing
type MockLLM struct {
	delay time.Duration
}

func (m *MockLLM) Chat(ctx context.Context, msgs []LLMMessage, tools []Tool) (*LLMResponse, error) {
	time.Sleep(m.delay)
	return &LLMResponse{
		Content: "Mock response for testing",
	}, nil
}

func (m *MockLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 1024), nil
}

// MockMutator tracks parallel execution
type MockMutator struct {
	mu          sync.Mutex
	callTimes   []time.Time
	callCount   int
	delay       time.Duration
}

func (m *MockMutator) MutateStrategy(ctx context.Context, parent *PathCandidate, trace *ExecutionTrace, feedback string) (*PathCandidate, error) {
	m.mu.Lock()
	m.callTimes = append(m.callTimes, time.Now())
	m.callCount++
	m.mu.Unlock()

	time.Sleep(m.delay)

	return &PathCandidate{
		ID:             fmt.Sprintf("mutated_%d", m.callCount),
		Generation:     parent.Generation + 1,
		ParentIDs:      []string{parent.ID},
		StrategyPrompt: "Mutated: " + parent.StrategyPrompt[:20],
	}, nil
}

func (m *MockMutator) Crossover(ctx context.Context, p1, p2 *PathCandidate) (*PathCandidate, error) {
	m.mu.Lock()
	m.callTimes = append(m.callTimes, time.Now())
	m.callCount++
	m.mu.Unlock()

	time.Sleep(m.delay)

	return &PathCandidate{
		ID:             "crossed",
		Generation:     max(p1.Generation, p2.Generation) + 1,
		ParentIDs:      []string{p1.ID, p2.ID},
		StrategyPrompt: "Crossed strategy",
	}, nil
}

func TestParallelMutation(t *testing.T) {
	// Create mock parents with traces and feedback
	parents := []*PathCandidate{
		{
			ID:             "parent1",
			Generation:     0,
			StrategyPrompt: "Strategy 1: do something methodically",
			Trace:          &ExecutionTrace{Query: "test", FinalAnswer: "answer1"},
			Feedback:       "Good but could be better",
		},
		{
			ID:             "parent2",
			Generation:     0,
			StrategyPrompt: "Strategy 2: do something efficiently",
			Trace:          &ExecutionTrace{Query: "test", FinalAnswer: "answer2"},
			Feedback:       "Needs more detail",
		},
		{
			ID:             "parent3",
			Generation:     0,
			StrategyPrompt: "Strategy 3: do something accurately",
			Trace:          &ExecutionTrace{Query: "test", FinalAnswer: "answer3"},
			Feedback:       "Could be faster",
		},
	}

	mutator := &MockMutator{delay: 100 * time.Millisecond}

	ctx := context.Background()
	start := time.Now()

	// Simulate the parallel mutation code from pareto.go
	var mutWg sync.WaitGroup
	mutResults := make(chan *PathCandidate, len(parents)+1)

	// Mutate parents in parallel
	for _, parent := range parents {
		mutWg.Add(1)
		go func(p *PathCandidate) {
			defer mutWg.Done()
			mutated, err := mutator.MutateStrategy(ctx, p, p.Trace, p.Feedback)
			if err != nil {
				t.Errorf("Mutation failed: %v", err)
				return
			}
			if mutated != nil {
				mutResults <- mutated
			}
		}(parent)
	}

	// Crossover in parallel
	mutWg.Add(1)
	go func() {
		defer mutWg.Done()
		crossed, err := mutator.Crossover(ctx, parents[0], parents[1])
		if err == nil && crossed != nil {
			mutResults <- crossed
		}
	}()

	go func() {
		mutWg.Wait()
		close(mutResults)
	}()

	// Collect results
	var results []*PathCandidate
	for mutated := range mutResults {
		results = append(results, mutated)
	}

	elapsed := time.Since(start)

	// Verify parallel execution
	// 3 mutations + 1 crossover, each taking 100ms
	// Sequential: ~400ms, Parallel: ~100ms
	t.Logf("Elapsed time: %v", elapsed)
	t.Logf("Results count: %d", len(results))
	t.Logf("Call count: %d", mutator.callCount)

	if len(results) != 4 {
		t.Errorf("Expected 4 results (3 mutations + 1 crossover), got %d", len(results))
	}

	if mutator.callCount != 4 {
		t.Errorf("Expected 4 calls, got %d", mutator.callCount)
	}

	// Check that calls happened in parallel (within reasonable tolerance)
	// If parallel, elapsed should be ~100-150ms, not ~400ms
	if elapsed > 250*time.Millisecond {
		t.Errorf("Mutations appear to be sequential: took %v, expected ~100ms for parallel execution", elapsed)
	}

	// Verify call times overlap (parallel execution)
	if len(mutator.callTimes) >= 2 {
		// All calls should start within a small window if parallel
		firstCall := mutator.callTimes[0]
		lastCallStart := mutator.callTimes[len(mutator.callTimes)-1]
		startDiff := lastCallStart.Sub(firstCall)

		t.Logf("Time between first and last call start: %v", startDiff)

		if startDiff > 50*time.Millisecond {
			t.Errorf("Calls don't appear parallel: %v between first and last call start", startDiff)
		}
	}

	t.Log("Parallel mutation test passed!")
}

func TestParallelEvaluation(t *testing.T) {
	// Test that candidates are evaluated in parallel
	candidates := createSeedCandidates(3)

	var wg sync.WaitGroup
	results := make(chan *PathCandidate, len(candidates))
	evalTimes := make([]time.Time, 0)
	var mu sync.Mutex

	evalDelay := 100 * time.Millisecond
	start := time.Now()

	for _, c := range candidates {
		wg.Add(1)
		go func(candidate *PathCandidate) {
			defer wg.Done()

			mu.Lock()
			evalTimes = append(evalTimes, time.Now())
			mu.Unlock()

			// Simulate evaluation
			time.Sleep(evalDelay)
			candidate.Scores = PathScores{
				Effectiveness:       4.0,
				AnswerQuality: 3.5,
				Hallucination: 4.5,
				Specificity:   3.0,
				TokenCost:     4.0,
				Latency:       3.5,
			}
			results <- candidate
		}(c)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var evaluated []*PathCandidate
	for c := range results {
		evaluated = append(evaluated, c)
	}

	elapsed := time.Since(start)

	t.Logf("Elapsed time: %v", elapsed)
	t.Logf("Evaluated count: %d", len(evaluated))

	if len(evaluated) != 3 {
		t.Errorf("Expected 3 evaluated candidates, got %d", len(evaluated))
	}

	// Should take ~100ms if parallel, ~300ms if sequential
	if elapsed > 200*time.Millisecond {
		t.Errorf("Evaluation appears sequential: took %v", elapsed)
	}

	t.Log("Parallel evaluation test passed!")
}
