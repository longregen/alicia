package main

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds the MCP server configuration
type Config struct {
	// DatabaseURL is the PostgreSQL connection string
	DatabaseURL string

	// SchemaDoc is the contents of the database documentation file
	SchemaDoc string

	// MaxResponseSize is the maximum character length for tool responses
	MaxResponseSize int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := &Config{}

	// Database URL
	cfg.DatabaseURL = os.Getenv("GARDEN_DATABASE_URL")
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	}

	// Schema documentation file path
	if docPath := os.Getenv("DATABASE_DOC_PATH"); docPath != "" {
		if content, err := os.ReadFile(docPath); err == nil {
			cfg.SchemaDoc = string(content)
			slog.Info("loaded database documentation", "path", docPath)
		} else {
			slog.Warn("failed to load database documentation", "path", docPath, "error", err)
		}
	}

	// Max response size (default 10k)
	cfg.MaxResponseSize = 10000
	if v := os.Getenv("MCP_MAX_CHARACTER_RESPONSE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxResponseSize = n
		}
	}

	return cfg
}

// GetSchemaContext returns the schema documentation
func (c *Config) GetSchemaContext() string {
	return c.SchemaDoc
}
