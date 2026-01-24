package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/mcp"
)

func main() {
	// Initialize OpenTelemetry
	otelEndpoint := config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://alicia-data.hjkl.lol")
	result, err := otel.Init(otel.Config{
		ServiceName:  "mcp-assistant",
		Environment:  config.GetEnv("ENVIRONMENT", ""),
		OTLPEndpoint: otelEndpoint,
	})
	if err != nil {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create WebSocket bridge to hub
	wsURL := config.GetEnv("WS_URL", "")
	agentSecret := config.GetEnv("AGENT_SECRET", "")
	if wsURL == "" {
		slog.Error("WS_URL environment variable is required")
		os.Exit(1)
	}

	bridge := NewBridge(wsURL, agentSecret)
	if err := bridge.Connect(ctx); err != nil {
		slog.Error("failed to connect bridge", "error", err)
		os.Exit(1)
	}
	defer bridge.Close()

	server := NewServer(slog.Default(), bridge)

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

type Server struct {
	logger *slog.Logger
	bridge *Bridge
	tools  []mcp.Tool
}

func NewServer(logger *slog.Logger, bridge *Bridge) *Server {
	return &Server{
		logger: logger,
		bridge: bridge,
		tools:  AllTools(),
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
			if err.Error() == "EOF" {
				return nil
			}
			s.logger.Error("failed to decode request", "error", err)
			continue
		}

		response := s.handleRequest(ctx, &request)
		if response == nil {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			s.logger.Error("failed to encode response", "error", err)
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *mcp.Request) *mcp.Response {
	s.logger.Info("handling request", "method", req.Method, "id", req.ID)

	switch req.Method {
	case "initialize":
		return mcp.NewResponse(req.ID, mcp.InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: mcp.ServerCapabilities{
				Tools: &mcp.ToolsCapability{ListChanged: false},
			},
			ServerInfo: mcp.ServerInfo{
				Name:    "mcp-assistant",
				Version: "1.0.0",
			},
		})
	case "initialized":
		return nil
	case "tools/list":
		return mcp.NewResponse(req.ID, mcp.ToolsListResult{Tools: s.tools})
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "ping":
		return mcp.NewResponse(req.ID, map[string]any{})
	default:
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *mcp.Request) *mcp.Response {
	params, err := mcp.DecodeParams(req.Params)
	if err != nil {
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
	}

	ctx, span := otel.StartMCPToolSpan(ctx, "mcp-assistant", "assistant", params.Name, params.Meta)
	defer span.End()

	// Validate tool exists
	found := false
	for _, t := range s.tools {
		if t.Name == params.Name {
			found = true
			break
		}
	}
	if !found {
		otel.RecordToolError(span, fmt.Errorf("unknown tool: %s", params.Name))
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	// Forward to assistant device via WebSocket bridge
	result, err := s.bridge.SendToolRequest(ctx, params.Name, params.Arguments)
	if err != nil {
		content := fmt.Sprintf("Error: %v", err)
		span.RecordError(err)
		otel.EndMCPToolSpan(span, true, len(content))
		return mcp.NewResponse(req.ID, mcp.NewToolError(content))
	}

	otel.EndMCPToolSpan(span, false, len(result))
	return mcp.NewResponse(req.ID, mcp.NewToolResult(result))
}
