package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/db"
	"github.com/longregen/alicia/shared/mcp"
)

func main() {
	// Initialize OpenTelemetry (provides tee'd logger: stderr JSON + OTLP export)
	otelEndpoint := config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://alicia-data.hjkl.lol")
	result, err := otel.Init(otel.Config{
		ServiceName:  "mcp-garden",
		Environment:  config.GetEnv("ENVIRONMENT", ""),
		OTLPEndpoint: otelEndpoint,
	})
	if err != nil {
		// Fallback to plain stderr logger if OTel fails
		slog.SetDefault(slog.New(otel.NewPrettyHandler()))
		slog.Warn("otel init failed, continuing without export", "error", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			result.Shutdown(shutdownCtx)
		}()
		slog.SetDefault(result.Logger)
		slog.Info("otel initialized", "endpoint", otelEndpoint)
	}

	// Load configuration
	cfg := LoadConfig()

	if cfg.DatabaseURL == "" {
		slog.Error("GARDEN_DATABASE_URL or DATABASE_URL environment variable required")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.ConnectSimple(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("connected to garden database")

	// Initialize LLM client
	llmClient := NewLLMClient()
	if llmClient.IsConfigured() {
		slog.Info("LLM client configured", "model", llmClient.model)
	} else {
		slog.Info("LLM not configured, using fallback hints (set LLM_API_KEY to enable)")
	}

	// Create and run server
	server := NewServer(pool, cfg, llmClient, slog.Default())

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down")
		cancel()
	}()

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// Server implements MCP protocol over stdio
type Server struct {
	pool   *pgxpool.Pool
	config *Config
	llm    *LLMClient
	logger *slog.Logger
}

func NewServer(pool *pgxpool.Pool, cfg *Config, llm *LLMClient, logger *slog.Logger) *Server {
	return &Server{
		pool:   pool,
		config: cfg,
		llm:    llm,
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var request mcp.Request
		if err := decoder.Decode(&request); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			s.logger.Error("failed to decode request", "error", err)
			continue
		}

		response := s.handleRequest(ctx, &request)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				s.logger.Error("failed to encode response", "error", err)
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *mcp.Request) *mcp.Response {
	s.logger.Info("handling request", "method", req.Method, "id", req.ID)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return nil // Notification, no response
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *mcp.Request) *mcp.Response {
	return mcp.NewResponse(req.ID, mcp.InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: mcp.ServerCapabilities{
			Tools: &mcp.ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: mcp.ServerInfo{
			Name:    "garden-db",
			Version: "1.0.0",
		},
	})
}

func (s *Server) handleToolsList(req *mcp.Request) *mcp.Response {
	return mcp.NewResponse(req.ID, mcp.ToolsListResult{
		Tools: s.getTools(),
	})
}

func (s *Server) handleToolsCall(ctx context.Context, req *mcp.Request) *mcp.Response {
	params, err := mcp.DecodeParams(req.Params)
	if err != nil {
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
	}

	// Start span with trace context from _meta
	ctx, span := otel.StartMCPToolSpan(ctx, "mcp-garden", "garden", params.Name, params.Meta)
	defer span.End()

	var result string
	var isError bool

	switch params.Name {
	case "describe_table":
		result, isError = s.describeTable(ctx, params.Arguments)
	case "execute_sql":
		result, isError = s.executeSQL(ctx, params.Arguments)
	case "schema_explore":
		result, isError = s.schemaExplore(ctx, params.Arguments)
	default:
		otel.RecordToolError(span, fmt.Errorf("unknown tool: %s", params.Name))
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	otel.EndMCPToolSpan(span, isError, len(result))

	if isError {
		return mcp.NewResponse(req.ID, mcp.NewToolError(result))
	}
	return mcp.NewResponse(req.ID, mcp.NewToolResult(result))
}

