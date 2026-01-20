package usecases

import (
	"testing"

	"github.com/longregen/alicia/internal/ports"
)

// TestParetoResponseGeneratorImplementsInterface ensures that
// ParetoResponseGenerator correctly implements the ports.ParetoResponseGenerator interface.
func TestParetoResponseGeneratorImplementsInterface(t *testing.T) {
	// This is a compile-time check - if ParetoResponseGenerator doesn't implement
	// ports.ParetoResponseGenerator, this won't compile.
	var _ ports.ParetoResponseGenerator = (*ParetoResponseGenerator)(nil)
}

// TestParetoGenerateResponseAdapterImplementsInterface ensures that
// ParetoGenerateResponseAdapter correctly implements the ports.GenerateResponseUseCase interface.
func TestParetoGenerateResponseAdapterImplementsInterface(t *testing.T) {
	var _ ports.GenerateResponseUseCase = (*ParetoGenerateResponseAdapter)(nil)
}

// TestParetoRegenerateResponseImplementsInterface ensures that
// ParetoRegenerateResponse correctly implements the ports.RegenerateResponseUseCase interface.
func TestParetoRegenerateResponseImplementsInterface(t *testing.T) {
	var _ ports.RegenerateResponseUseCase = (*ParetoRegenerateResponse)(nil)
}

// TestParetoContinueResponseImplementsInterface ensures that
// ParetoContinueResponse correctly implements the ports.ContinueResponseUseCase interface.
func TestParetoContinueResponseImplementsInterface(t *testing.T) {
	var _ ports.ContinueResponseUseCase = (*ParetoContinueResponse)(nil)
}

// TestDefaultParetoResponseConfig tests that the default configuration is valid.
func TestDefaultParetoResponseConfig(t *testing.T) {
	config := DefaultParetoResponseConfig()

	if config == nil {
		t.Fatal("DefaultParetoResponseConfig returned nil")
	}

	if config.MaxGenerations <= 0 {
		t.Errorf("MaxGenerations should be > 0, got %d", config.MaxGenerations)
	}

	if config.BranchesPerGen <= 0 {
		t.Errorf("BranchesPerGen should be > 0, got %d", config.BranchesPerGen)
	}

	if config.TargetScore <= 0 || config.TargetScore > 1.0 {
		t.Errorf("TargetScore should be in (0, 1], got %f", config.TargetScore)
	}

	if config.MaxToolCalls <= 0 {
		t.Errorf("MaxToolCalls should be > 0, got %d", config.MaxToolCalls)
	}

	if config.MaxLLMCalls <= 0 {
		t.Errorf("MaxLLMCalls should be > 0, got %d", config.MaxLLMCalls)
	}

	if config.ParetoArchiveSize <= 0 {
		t.Errorf("ParetoArchiveSize should be > 0, got %d", config.ParetoArchiveSize)
	}

	if config.ExecutionTimeoutMs <= 0 {
		t.Errorf("ExecutionTimeoutMs should be > 0, got %d", config.ExecutionTimeoutMs)
	}

	if config.MaxToolLoopIterations <= 0 {
		t.Errorf("MaxToolLoopIterations should be > 0, got %d", config.MaxToolLoopIterations)
	}
}
