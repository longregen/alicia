package builtin

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/ports"
)

// RegisterAllBuiltinTools registers all built-in tools with the tool service
func RegisterAllBuiltinTools(
	ctx context.Context,
	toolService ports.ToolService,
	memoryRepo ports.MemoryRepository,
	embeddingService ports.EmbeddingService,
) error {
	// Register calculator tool
	if err := RegisterCalculator(ctx, toolService); err != nil {
		return fmt.Errorf("failed to register calculator: %w", err)
	}

	// Register web search tool (DuckDuckGo HTML search)
	if err := RegisterWebSearch(ctx, toolService); err != nil {
		return fmt.Errorf("failed to register web search: %w", err)
	}

	// Register memory query tool (if services are provided)
	if memoryRepo != nil && embeddingService != nil {
		if err := RegisterMemoryQuery(ctx, toolService, memoryRepo, embeddingService); err != nil {
			return fmt.Errorf("failed to register memory query: %w", err)
		}
	}

	return nil
}
