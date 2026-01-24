package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/pkg/langfuse"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

var paretoMode = flag.Bool("pareto", false, "Enable pareto-efficient exploration")

func main() {
	flag.Parse()

	cfg := struct {
		DatabaseURL    string
		ServerURL      string
		LLMAPIKey      string
		LLMURL         string
		LLMModel       string
		EmbeddingModel string
		ParetoMode     bool
		OTLPEndpoint   string
		Environment    string
	}{
		DatabaseURL:    config.MustEnv("DATABASE_URL"),
		ServerURL:      config.MustEnv("SERVER_URL"),
		LLMURL:         config.MustEnv("LLM_URL"),
		LLMAPIKey:      config.MustEnv("LLM_API_KEY"),
		LLMModel:       config.GetEnv("LLM_MODEL", "gpt-4"),
		EmbeddingModel: config.GetEnv("EMBEDDING_MODEL", "text-embedding-3-small"),
		ParetoMode:     *paretoMode || os.Getenv("PARETO_MODE") == "true",
		OTLPEndpoint:   config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		Environment:    config.GetEnv("ENVIRONMENT", "development"),
	}

	if cfg.OTLPEndpoint != "" {
		result, err := otel.Init(otel.Config{
			ServiceName:  "alicia-agent",
			Environment:  cfg.Environment,
			OTLPEndpoint: cfg.OTLPEndpoint,
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
			slog.Info("opentelemetry initialized", "endpoint", cfg.OTLPEndpoint)
		}
	} else {
		slog.SetDefault(slog.New(otel.NewPrettyHandler()))
		slog.Info("opentelemetry not configured, OTEL_EXPORTER_OTLP_ENDPOINT not set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := ConnectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	llm := NewLLMClient(cfg.LLMURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.EmbeddingModel, 4096)
	slog.Info("llm client created")

	mcpServers, err := LoadEnabledMCPServers(ctx, db)
	if err != nil {
		slog.Error("failed to load mcp servers from database", "error", err)
	}

	var mcp *MCPManager
	if len(mcpServers) > 0 {
		mcp, err = NewMCPManager(mcpServers)
		if err != nil {
			slog.Warn("mcp manager failed, continuing without tools", "error", err)
		} else {
			defer mcp.Close()
			slog.Info("mcp manager started", "tool_count", len(mcp.Tools()), "server_count", len(mcpServers))
		}
	} else {
		slog.Info("no mcp servers configured in database, running without tools")
	}

	prefs := NewPreferencesStore()
	deps := AgentDeps{DB: db, LLM: llm, MCP: mcp, Prefs: prefs, ParetoMode: cfg.ParetoMode}

	if cfg.ParetoMode {
		slog.Info("pareto mode enabled")
	}

	go initLangfuseScoreConfigs()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := runAgentLoop(ctx, cfg.ServerURL, deps); err != nil {
				slog.Error("agent loop error", "error", err)
			}
			slog.Info("reconnecting in 5 seconds")
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down")
	cancel()
}

func runAgentLoop(ctx context.Context, serverURL string, deps AgentDeps) error {
	slog.Info("connecting to server", "url", serverURL)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, serverURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	slog.Info("connected to server")

	if err := subscribeAsAgent(conn); err != nil {
		return err
	}
	slog.Info("registered as agent")

	notifier := NewWSNotifier(conn)
	deps.Notifier = notifier

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var envelope protocol.Envelope
		if err := msgpack.Unmarshal(data, &envelope); err != nil {
			slog.Error("decode error", "error", err)
			continue
		}

		switch envelope.Type {
		case protocol.TypePreferencesUpdate:
			var update protocol.PreferencesUpdate
			bodyBytes, _ := msgpack.Marshal(envelope.Body)
			if err := msgpack.Unmarshal(bodyBytes, &update); err != nil {
				slog.Error("preferences decode error", "error", err)
				continue
			}
			deps.Prefs.Update(update)
			slog.Info("updated preferences", "user_id", update.UserID)

		case protocol.TypeGenRequest:
			var req ResponseGenerationRequest
			bodyBytes, _ := msgpack.Marshal(envelope.Body)
			if err := msgpack.Unmarshal(bodyBytes, &req); err != nil {
				slog.Error("request decode error", "error", err)
				continue
			}

			slog.Info("request received", "type", req.RequestType, "conversation_id", req.ConversationID, "message_id", req.MessageID)
			notifier.SetConversationID(req.ConversationID)

			reqCtx := otel.WithSessionID(ctx, req.ConversationID)
			if envelope.UserID != "" {
				reqCtx = otel.WithUserID(reqCtx, envelope.UserID)
			}
			if envelope.HasTraceContext() {
				reqCtx = otel.ExtractFromTraceContext(reqCtx, otel.TraceContext{
					TraceID:    envelope.TraceID,
					SpanID:     envelope.SpanID,
					TraceFlags: envelope.TraceFlags,
					SessionID:  envelope.SessionID,
					UserID:     envelope.UserID,
				})
			}

			// Create deps with user ID for this request
			reqDeps := deps
			reqDeps.UserID = envelope.UserID

			go func(reqCtx context.Context, req ResponseGenerationRequest, reqDeps AgentDeps) {
				var err error
				switch req.RequestType {
				case "send":
					err = HandleSend(reqCtx, req, reqDeps)
				case "regenerate":
					err = HandleRegenerate(reqCtx, req, reqDeps)
				case "continue":
					err = HandleContinue(reqCtx, req, reqDeps)
				case "edit":
					err = HandleEdit(reqCtx, req, reqDeps)
				default:
					slog.Warn("unknown request type", "type", req.RequestType)
					return
				}
				if err != nil {
					slog.Error("handler error", "type", req.RequestType, "error", err)
				}
			}(reqCtx, req, reqDeps)
		}
	}
}

func subscribeAsAgent(conn *websocket.Conn) error {
	data, _ := msgpack.Marshal(protocol.Envelope{Type: protocol.TypeSubscribe, Body: map[string]any{"agentMode": true}})
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return err
	}

	_, ackData, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	var ack protocol.Envelope
	if err := msgpack.Unmarshal(ackData, &ack); err != nil {
		return fmt.Errorf("unmarshal subscribe ack: %w", err)
	}
	slog.Info("subscribe ack received", "type", ack.Type)
	return nil
}

// initLangfuseScoreConfigs initializes Langfuse evaluators, score configurations, and datasets.
// This runs in a goroutine and logs errors but does not fail startup.
func initLangfuseScoreConfigs() {
	client := getLangfuseClient()
	if client == nil {
		slog.Info("langfuse not configured, skipping evaluator and score config setup")
		return
	}

	setupCtx, setupCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer setupCancel()
	if err := langfuse.SetupAll(setupCtx, client); err != nil {
		slog.Error("failed to setup langfuse evaluators/score configs", "error", err)
	} else {
		slog.Info("langfuse evaluators and score configs initialized")
	}

	dsCtx, dsCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dsCancel()
	if err := langfuse.EnsureGoldenDataset(dsCtx, client); err != nil {
		slog.Error("failed to ensure golden dataset", "error", err)
	} else {
		slog.Info("langfuse golden dataset initialized")
	}
}

