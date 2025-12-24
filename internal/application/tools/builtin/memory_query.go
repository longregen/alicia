package builtin

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// RegisterMemoryQuery registers the memory query tool with the tool service
func RegisterMemoryQuery(
	ctx context.Context,
	toolService ports.ToolService,
	memoryRepo ports.MemoryRepository,
	embeddingService ports.EmbeddingService,
) error {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The query to search for in memory",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of memories to return (default: 5)",
				"default":     5,
			},
		},
		"required": []string{"query"},
	}

	tool, err := toolService.RegisterTool(
		ctx,
		"memory_query",
		"Searches through conversation memories to retrieve relevant past information.",
		schema,
	)
	if err != nil {
		return fmt.Errorf("failed to register memory query tool: %w", err)
	}

	// Register the executor
	err = toolService.RegisterExecutor("memory_query", func(ctx context.Context, arguments map[string]any) (any, error) {
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("query must be a string")
		}

		limit := 5
		if l, ok := arguments["limit"]; ok {
			switch v := l.(type) {
			case float64:
				limit = int(v)
			case int:
				limit = v
			}
		}

		// Generate embedding for the query
		embeddingResult, err := embeddingService.Embed(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}

		// Search memories
		results, err := memoryRepo.SearchMemories(ctx, ports.MemorySearchOptions{
			Embedding:     embeddingResult.Embedding,
			Limit:         limit,
			IncludeScores: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search memories: %w", err)
		}

		// Format results
		memories := make([]map[string]any, 0, len(results))
		for _, result := range results {
			memories = append(memories, map[string]any{
				"id":         result.Memory.ID,
				"content":    result.Memory.Content,
				"similarity": result.Similarity,
				"tags":       result.Memory.Tags,
			})
		}

		return map[string]any{
			"query":    query,
			"count":    len(memories),
			"memories": memories,
		}, nil
	})

	if err != nil {
		return fmt.Errorf("failed to register memory query executor: %w", err)
	}

	log.Printf("Registered memory query tool: %s", tool.ID)
	return nil
}

// GetMemoryQueryTool returns the memory query tool definition
func GetMemoryQueryTool() *models.Tool {
	return &models.Tool{
		Name:        "memory_query",
		Description: "Searches through conversation memories",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The query to search for",
				},
				"limit": map[string]any{
					"type":    "integer",
					"default": 5,
				},
			},
			"required": []string{"query"},
		},
		Enabled: true,
	}
}
