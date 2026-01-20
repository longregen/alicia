package tools

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/ports"
)

// BootstrapOptions configures the native tools bootstrap
type BootstrapOptions struct {
	// EnableWeb enables web tools (web_read, web_search, etc.)
	EnableWeb bool
	// EnableGarden enables garden tools (requires DatabasePool)
	EnableGarden bool
	// DatabasePool is required for garden tools
	DatabasePool *pgxpool.Pool
}

// DefaultOptions returns default bootstrap options
func DefaultOptions() BootstrapOptions {
	return BootstrapOptions{
		EnableWeb:    true,
		EnableGarden: os.Getenv("GARDEN_DATABASE_URL") != "" || os.Getenv("DATABASE_URL") != "",
	}
}

// Bootstrap registers all native tools with the tool service
func Bootstrap(ctx context.Context, toolService ports.ToolService, opts BootstrapOptions) error {
	log.Println("Bootstrapping native tools...")

	var registry *Registry

	if opts.EnableGarden && opts.DatabasePool != nil {
		// Create registry with garden tools
		gardenDB := NewPgxGardenDB(opts.DatabasePool)
		registry = NewRegistryWithGarden(gardenDB)
		log.Println("Garden tools enabled")
	} else {
		// Create registry without garden tools
		registry = NewRegistry()
		if opts.EnableGarden {
			log.Println("Garden tools requested but no database pool provided, skipping")
		}
	}

	// Register all tools
	if err := registry.RegisterAll(ctx, toolService); err != nil {
		return err
	}

	log.Printf("Native tools bootstrap complete: %d tools registered", len(registry.Tools()))

	// Log registered tools
	for _, tool := range registry.Tools() {
		log.Printf("  - %s: %s", tool.Name(), truncateDescription(tool.Description(), 60))
	}

	return nil
}

// truncateDescription truncates a description string with ellipsis
func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// BootstrapWithDefaults bootstraps native tools with default options
func BootstrapWithDefaults(ctx context.Context, toolService ports.ToolService, pool *pgxpool.Pool) error {
	opts := DefaultOptions()
	opts.DatabasePool = pool
	return Bootstrap(ctx, toolService, opts)
}
