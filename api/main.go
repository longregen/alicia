package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/longregen/alicia/api/config"
	"github.com/longregen/alicia/api/livekit"
	"github.com/longregen/alicia/api/server"
	"github.com/longregen/alicia/api/services"
	"github.com/longregen/alicia/api/store"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/db"
)

func main() {
	cfg := config.Load()

	if cfg.Otel.Endpoint != "" {
		result, err := otel.Init(otel.Config{
			ServiceName:  "alicia-api",
			Environment:  cfg.Otel.Environment,
			OTLPEndpoint: cfg.Otel.Endpoint,
		})
		if err != nil {
			slog.Error("failed to initialize opentelemetry", "error", err)
		} else {
			defer func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				result.Shutdown(shutdownCtx)
			}()
			slog.SetDefault(result.Logger)
			slog.Info("opentelemetry initialized", "endpoint", cfg.Otel.Endpoint)
		}
	} else {
		slog.SetDefault(slog.New(otel.NewPrettyHandler()))
		slog.Info("opentelemetry not configured, OTEL_EXPORTER_OTLP_ENDPOINT not set")
	}

	slog.Info("starting alicia backend")
	slog.Info("server configured", "host", cfg.Server.Host, "port", cfg.Server.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info("connecting to database")
	pool, err := db.Connect(ctx, db.Config{URL: cfg.Database.URL, Timezone: "UTC"})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	s := store.New(pool)

	convSvc := services.NewConversationService(s)
	msgSvc := services.NewMessageService(s)
	memorySvc := services.NewMemoryService(s, nil)
	toolSvc := services.NewToolService(s)
	mcpSvc := services.NewMCPService(s)
	prefsSvc := services.NewPreferencesService(s)
	noteSvc := services.NewNoteService(s, nil)

	var lkSvc *livekit.Service
	if cfg.IsLiveKitConfigured() {
		lkSvc, err = livekit.NewService(cfg.LiveKit.URL, cfg.LiveKit.APIKey, cfg.LiveKit.APISecret)
		if err != nil {
			slog.Warn("failed to initialize livekit", "error", err)
		} else {
			slog.Info("livekit configured", "url", cfg.LiveKit.URL)
		}
	}

	srv := server.NewServer(cfg, s, convSvc, msgSvc, memorySvc, toolSvc, mcpSvc, prefsSvc, noteSvc, lkSvc)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "host", cfg.Server.Host, "port", cfg.Server.Port)
		errCh <- srv.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := srv.Stop(shutdownCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
		slog.Info("server stopped")
	}
}
