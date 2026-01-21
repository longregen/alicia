package usecases

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/ports"
)

// ParetoGenerateResponseAdapter wraps ParetoResponseGenerator to implement
// the GenerateResponseUseCase interface for backwards compatibility.
// This allows gradual migration from the old interface to the new Pareto-based system.
type ParetoGenerateResponseAdapter struct {
	paretoGenerator *ParetoResponseGenerator
}

// NewParetoGenerateResponseAdapter creates a new adapter that wraps ParetoResponseGenerator.
func NewParetoGenerateResponseAdapter(paretoGenerator *ParetoResponseGenerator) *ParetoGenerateResponseAdapter {
	return &ParetoGenerateResponseAdapter{
		paretoGenerator: paretoGenerator,
	}
}

// Execute implements ports.GenerateResponseUseCase by delegating to ParetoResponseGenerator.
func (a *ParetoGenerateResponseAdapter) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Convert from GenerateResponseInput to ParetoResponseInput
	log.Printf("[ParetoGenerateResponseAdapter] Converting input: PreviousID=%s", input.PreviousID)
	paretoInput := &ParetoResponseInput{
		ConversationID:  input.ConversationID,
		UserMessageID:   input.UserMessageID,
		MessageID:       input.MessageID,
		PreviousID:      input.PreviousID,
		EnableTools:     input.EnableTools,
		EnableReasoning: input.EnableReasoning,
		EnableStreaming: input.EnableStreaming,
		Notifier:        input.Notifier,
	}

	// If ContinueFromContent is set, use it as a seed strategy
	if input.ContinueFromContent != "" {
		paretoInput.SeedStrategy = fmt.Sprintf(`Continue from this existing content:
%s

Build upon what's already written. Do not repeat existing content.`, input.ContinueFromContent)
	}

	// Execute via Pareto generator
	paretoOutput, err := a.paretoGenerator.Execute(ctx, paretoInput)
	if err != nil {
		return nil, err
	}

	// Convert from ParetoResponseOutput to GenerateResponseOutput
	return &ports.GenerateResponseOutput{
		Message:        paretoOutput.Message,
		Sentences:      paretoOutput.Sentences,
		ToolUses:       paretoOutput.ToolUses,
		ReasoningSteps: paretoOutput.ReasoningSteps,
		StreamChannel:  paretoOutput.StreamChannel,
	}, nil
}

// Ensure ParetoGenerateResponseAdapter implements ports.GenerateResponseUseCase
var _ ports.GenerateResponseUseCase = (*ParetoGenerateResponseAdapter)(nil)
