package tools

import (
	"context"

	"github.com/longregen/alicia/internal/ports"
)

type NativeTool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, args map[string]any) (any, error)
}

type Registry struct {
	tools []NativeTool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: []NativeTool{
			NewWebReadTool(),
			NewWebFetchRawTool(),
			NewWebFetchStructuredTool(),
			NewWebSearchTool(),
			NewWebExtractLinksTool(),
			NewWebExtractMetadataTool(),
			NewWebScreenshotTool(),
		},
	}
}

func NewRegistryWithGarden(gardenDB GardenDB) *Registry {
	r := NewRegistry()
	r.tools = append(r.tools,
		NewGardenDescribeTableTool(gardenDB),
		NewGardenExecuteSQLTool(gardenDB),
		NewGardenSchemaExploreTool(gardenDB),
	)
	return r
}

func (r *Registry) RegisterAll(ctx context.Context, toolService ports.ToolService) error {
	for _, tool := range r.tools {
		if _, err := toolService.EnsureTool(ctx, tool.Name(), tool.Description(), tool.Schema()); err != nil {
			return err
		}

		executor := createExecutor(tool)
		if err := toolService.RegisterExecutor(tool.Name(), executor); err != nil {
			return err
		}
	}
	return nil
}

func createExecutor(tool NativeTool) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return tool.Execute(ctx, args)
	}
}

func (r *Registry) Tools() []NativeTool {
	return r.tools
}
