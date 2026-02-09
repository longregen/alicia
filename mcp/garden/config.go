package main

import (
	"log/slog"
	"os"

	"github.com/longregen/alicia/shared/config"
)

type Config struct {
	DatabaseURL     string
	SchemaDoc       string
	MaxResponseSize int
}

func LoadConfig() *Config {
	cfg := &Config{}

	// Database URL with fallback
	cfg.DatabaseURL = config.GetEnvWithFallback("GARDEN_DATABASE_URL", "DATABASE_URL", "")

	// Schema documentation file path
	if docPath := config.GetEnv("DATABASE_DOC_PATH", ""); docPath != "" {
		if content, err := os.ReadFile(docPath); err == nil {
			cfg.SchemaDoc = string(content)
			slog.Info("loaded database documentation", "path", docPath)
		} else {
			slog.Warn("failed to load database documentation", "path", docPath, "error", err)
		}
	}

	// Max response size (default 10k)
	cfg.MaxResponseSize = config.GetEnvInt("MCP_MAX_CHARACTER_RESPONSE_SIZE", 10000)

	return cfg
}

