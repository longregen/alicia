package main

import (
	"fmt"
	"os"

	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/llm"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "alicia",
		Short: "Alicia - AI Voice Assistant CLI",
		Long: `Alicia is a self-hosted personal AI assistant.
This CLI provides text-based interaction with Alicia.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			llmClient = llm.NewClient(
				cfg.LLM.URL,
				cfg.LLM.APIKey,
				cfg.LLM.Model,
				cfg.LLM.MaxTokens,
				cfg.LLM.Temperature,
			)

			return nil
		},
	}

	rootCmd.AddCommand(
		chatCmd(),
		newCmd(),
		listCmd(),
		showCmd(),
		deleteCmd(),
		configCmd(),
		serveCmd(),
		agentCmd(),
		optimizeCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// configCmd shows current configuration
func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Current configuration:")
			fmt.Println()

			fmt.Println("LLM:")
			fmt.Printf("  URL:         %s\n", cfg.LLM.URL)
			fmt.Printf("  Model:       %s\n", cfg.LLM.Model)
			fmt.Printf("  Max Tokens:  %d\n", cfg.LLM.MaxTokens)
			fmt.Printf("  Temperature: %.2f\n", cfg.LLM.Temperature)
			fmt.Printf("  API Key:     %s\n", maskSecret(cfg.LLM.APIKey))
			fmt.Println()

			fmt.Println("LiveKit:")
			fmt.Printf("  URL:        %s\n", cfg.LiveKit.URL)
			fmt.Printf("  API Key:    %s\n", maskSecret(cfg.LiveKit.APIKey))
			fmt.Printf("  API Secret: %s\n", maskSecret(cfg.LiveKit.APISecret))
			fmt.Printf("  Status:     %s\n", boolStatus(cfg.IsLiveKitConfigured()))
			fmt.Println()

			fmt.Println("ASR (Speech Recognition):")
			fmt.Printf("  URL:     %s\n", cfg.ASR.URL)
			fmt.Printf("  Model:   %s\n", cfg.ASR.Model)
			fmt.Printf("  API Key: %s\n", maskSecret(cfg.ASR.APIKey))
			fmt.Printf("  Status:  %s\n", boolStatus(cfg.IsASRConfigured()))
			fmt.Println()

			fmt.Println("TTS (Text-to-Speech):")
			fmt.Printf("  URL:     %s\n", cfg.TTS.URL)
			fmt.Printf("  Model:   %s\n", cfg.TTS.Model)
			fmt.Printf("  Voice:   %s\n", cfg.TTS.Voice)
			fmt.Printf("  API Key: %s\n", maskSecret(cfg.TTS.APIKey))
			fmt.Printf("  Status:  %s\n", boolStatus(cfg.IsTTSConfigured()))
			fmt.Println()

			fmt.Println("Database:")
			fmt.Printf("  SQLite Path:   %s\n", cfg.Database.Path)
			fmt.Printf("  PostgreSQL:    %s\n", maskSecret(cfg.Database.PostgresURL))
			fmt.Println()

			fmt.Println("Environment variables:")
			fmt.Println("  ALICIA_LLM_URL, ALICIA_LLM_API_KEY, ALICIA_LLM_MODEL")
			fmt.Println("  ALICIA_LIVEKIT_URL, ALICIA_LIVEKIT_API_KEY, ALICIA_LIVEKIT_API_SECRET")
			fmt.Println("  ALICIA_ASR_URL, ALICIA_ASR_API_KEY, ALICIA_ASR_MODEL")
			fmt.Println("  ALICIA_TTS_URL, ALICIA_TTS_API_KEY, ALICIA_TTS_MODEL, ALICIA_TTS_VOICE")
			fmt.Println("  ALICIA_DB_PATH, ALICIA_POSTGRES_URL")

			return nil
		},
	}
}

// versionCmd shows version information
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Alicia %s\n", version)
			fmt.Printf("  Commit:     %s\n", commit)
			fmt.Printf("  Build Date: %s\n", buildDate)
		},
	}
}
