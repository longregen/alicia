// Package tools provides native tool implementations for Alicia.
// These tools are registered directly with the ToolService without going through MCP.
package tools

import (
	"context"

	"github.com/longregen/alicia/internal/ports"
)

// NativeTool defines the interface for a native tool implementation.
type NativeTool interface {
	// Name returns the tool's unique identifier
	Name() string
	// Description returns a human-readable description of what the tool does
	Description() string
	// Schema returns the JSON Schema for the tool's input parameters
	Schema() map[string]any
	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]any) (any, error)
}

// Registry manages the collection of native tools
type Registry struct {
	tools []NativeTool
}

// NewRegistry creates a new tool registry with all native tools
func NewRegistry() *Registry {
	return &Registry{
		tools: []NativeTool{
			// Web tools
			NewWebReadTool(),
			NewWebFetchRawTool(),
			NewWebFetchStructuredTool(),
			NewWebSearchTool(),
			NewWebExtractLinksTool(),
			NewWebExtractMetadataTool(),
			NewWebScreenshotTool(),
			// Garden tools will be added conditionally based on database config
		},
	}
}

// NewRegistryWithGarden creates a registry with all tools including garden tools
func NewRegistryWithGarden(gardenDB GardenDB) *Registry {
	r := NewRegistry()
	r.tools = append(r.tools,
		NewGardenDescribeTableTool(gardenDB),
		NewGardenExecuteSQLTool(gardenDB),
		NewGardenSchemaExploreTool(gardenDB),
	)
	return r
}

// RegisterAll registers all tools with the given tool service
func (r *Registry) RegisterAll(ctx context.Context, toolService ports.ToolService) error {
	for _, tool := range r.tools {
		// Register tool definition (idempotent)
		if _, err := toolService.EnsureTool(ctx, tool.Name(), tool.Description(), tool.Schema()); err != nil {
			return err
		}

		// Register executor
		executor := createExecutor(tool)
		if err := toolService.RegisterExecutor(tool.Name(), executor); err != nil {
			return err
		}
	}
	return nil
}

// createExecutor wraps a NativeTool in a ToolService-compatible executor
func createExecutor(tool NativeTool) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return tool.Execute(ctx, args)
	}
}

// Tools returns all registered tools
func (r *Registry) Tools() []NativeTool {
	return r.tools
}
