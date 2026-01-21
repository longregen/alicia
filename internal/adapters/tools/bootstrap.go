package tools

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/ports"
)

type BootstrapOptions struct {
	EnableWeb    bool
	EnableGarden bool
	DatabasePool *pgxpool.Pool
}

func DefaultOptions() BootstrapOptions {
	return BootstrapOptions{
		EnableWeb:    true,
		EnableGarden: os.Getenv("GARDEN_DATABASE_URL") != "" || os.Getenv("DATABASE_URL") != "",
	}
}

func Bootstrap(ctx context.Context, toolService ports.ToolService, opts BootstrapOptions) error {
	log.Println("Bootstrapping native tools...")

	var registry *Registry

	if opts.EnableGarden && opts.DatabasePool != nil {
		gardenDB := NewPgxGardenDB(opts.DatabasePool)
		registry = NewRegistryWithGarden(gardenDB)
		log.Println("Garden tools enabled")
	} else {
		registry = NewRegistry()
		if opts.EnableGarden {
			log.Println("Garden tools requested but no database pool provided, skipping")
		}
	}

	if err := registry.RegisterAll(ctx, toolService); err != nil {
		return err
	}

	log.Printf("Native tools bootstrap complete: %d tools registered", len(registry.Tools()))

	for _, tool := range registry.Tools() {
		log.Printf("  - %s: %s", tool.Name(), truncateDescription(tool.Description(), 60))
	}

	return nil
}

func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func BootstrapWithDefaults(ctx context.Context, toolService ports.ToolService, pool *pgxpool.Pool) error {
	opts := DefaultOptions()
	opts.DatabasePool = pool
	return Bootstrap(ctx, toolService, opts)
}
