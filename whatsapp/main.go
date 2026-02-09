package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
)

type Config struct {
	BackendWSURL   string
	AgentSecret    string
	AliciaAPIURL   string
	AliciaUserID   string
	ReaderDBPath   string
	AliciaDBPath   string
	ArchiveDBPath  string
	ResponsePrefix string
	AllowedJIDs    []string
}

func LoadConfig() *Config {
	allowedRaw := config.GetEnv("WHATSAPP_ALLOWED_JIDS", "")
	var allowedJIDs []string
	if allowedRaw != "" {
		for _, jid := range strings.Split(allowedRaw, ",") {
			jid = strings.TrimSpace(jid)
			if jid != "" {
				allowedJIDs = append(allowedJIDs, jid)
			}
		}
	}

	return &Config{
		BackendWSURL:   config.GetEnv("BACKEND_WS_URL", "ws://localhost:8080/ws"),
		AgentSecret:    config.GetEnv("AGENT_SECRET", ""),
		AliciaAPIURL:   config.GetEnv("ALICIA_API_URL", "http://localhost:8090/api/v1"),
		AliciaUserID:   config.GetEnv("ALICIA_USER_ID", "default_user"),
		ReaderDBPath:   config.GetEnv("WHATSAPP_READER_DB_PATH", "whatsapp-reader-session.db"),
		AliciaDBPath:   config.GetEnv("WHATSAPP_ALICIA_DB_PATH", "whatsapp-alicia-session.db"),
		ArchiveDBPath:  config.GetEnv("WHATSAPP_ARCHIVE_DB_PATH", "whatsapp-archive.db"),
		ResponsePrefix: config.GetEnv("WHATSAPP_RESPONSE_PREFIX", ""),
		AllowedJIDs:    allowedJIDs,
	}
}

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	result, err := otel.Init(otel.Config{
		ServiceName:  "alicia-whatsapp",
		Environment:  config.GetEnv("ENVIRONMENT", "development"),
		OTLPEndpoint: config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://alicia-data.hjkl.lol/otlp"),
	})
	if err != nil {
		slog.SetDefault(slog.New(otel.NewPrettyHandler()))
		slog.Warn("otel init failed, using stderr-only logger", "error", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			result.Shutdown(shutdownCtx)
		}()
		slog.SetDefault(result.Logger)
	}

	slog.Info("starting alicia whatsapp adapter")

	cfg := LoadConfig()
	logConfig(cfg)

	if len(cfg.AllowedJIDs) == 0 {
		slog.Warn("WHATSAPP_ALLOWED_JIDS is empty, alicia client will not respond to anyone")
	}

	archive, err := NewArchive(cfg.ArchiveDBPath)
	if err != nil {
		slog.Error("failed to open archive", "error", err)
		os.Exit(1)
	}
	defer archive.Close()

	bridge := NewBridge(cfg, archive)
	ws := NewWSClient(cfg)

	readerClient := NewWhatsAppClient(cfg, "reader", cfg.ReaderDBPath, ws, nil, archive)
	aliciaClient := NewWhatsAppClient(cfg, "alicia", cfg.AliciaDBPath, ws, bridge, archive)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ws.onPairRequest = func(role string) {
		switch role {
		case "reader":
			go readerClient.StartPairing(ctx)
		case "alicia":
			go aliciaClient.StartPairing(ctx)
		default:
			slog.Warn("ws: unknown pair request role", "role", role)
		}
	}

	if err := ws.Connect(ctx); err != nil {
		slog.Error("failed to connect to hub", "error", err)
		os.Exit(1)
	}

	if err := readerClient.Init(ctx); err != nil {
		slog.Error("failed to init reader whatsapp client", "error", err)
		os.Exit(1)
	}

	if err := aliciaClient.Init(ctx); err != nil {
		slog.Error("failed to init alicia whatsapp client", "error", err)
		os.Exit(1)
	}

	slog.Info("whatsapp adapter is running (reader + alicia)")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down")
	cancel()
	readerClient.Close()
	aliciaClient.Close()
	ws.Disconnect()
	slog.Info("whatsapp adapter stopped")
}

func printHelp() {
	fmt.Println(`Alicia WhatsApp Adapter

Bridges WhatsApp messages to the Alicia AI assistant using two connections:
  - Reader: Linked to your personal WhatsApp. Archives all messages passively.
  - Alicia: Linked to a separate WhatsApp number. Responds to allowlisted contacts.

Environment Variables:
  Backend Connection:
    BACKEND_WS_URL              Hub WebSocket URL (default: ws://localhost:8080/ws)
    AGENT_SECRET                Secret for Hub authentication (default: "")

  Alicia API:
    ALICIA_API_URL              REST API base URL (default: http://localhost:8090/api/v1)
    ALICIA_USER_ID              User ID for API requests (default: default_user)

  WhatsApp:
    WHATSAPP_READER_DB_PATH     Reader whatsmeow session DB (default: whatsapp-reader-session.db)
    WHATSAPP_ALICIA_DB_PATH     Alicia whatsmeow session DB (default: whatsapp-alicia-session.db)
    WHATSAPP_ARCHIVE_DB_PATH    Shared archive SQLite DB (default: whatsapp-archive.db)
    WHATSAPP_ALLOWED_JIDS       Comma-separated JIDs allowed to chat with Alicia
    WHATSAPP_RESPONSE_PREFIX    Optional prefix for responses (default: "")

  Telemetry:
    OTEL_EXPORTER_OTLP_ENDPOINT  OTLP endpoint (default: https://alicia-data.hjkl.lol/otlp)
    ENVIRONMENT                   Environment name (default: development)

Usage:
  whatsapp [flags]

Flags:
  -h, -help  Show this help message`)
}

func logConfig(cfg *Config) {
	slog.Info("configuration",
		"backend_ws_url", cfg.BackendWSURL,
		"agent_secret", maskSecret(cfg.AgentSecret),
		"alicia_api_url", cfg.AliciaAPIURL,
		"alicia_user_id", cfg.AliciaUserID,
		"reader_db_path", cfg.ReaderDBPath,
		"alicia_db_path", cfg.AliciaDBPath,
		"archive_db_path", cfg.ArchiveDBPath,
		"allowed_jids", cfg.AllowedJIDs,
		"response_prefix", cfg.ResponsePrefix,
	)
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
