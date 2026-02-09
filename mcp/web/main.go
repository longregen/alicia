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

	"github.com/longregen/alicia/mcp/web/tools"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/mcp"
)

func main() {
	// Initialize OpenTelemetry (provides tee'd logger: stderr JSON + OTLP export)
	otelEndpoint := config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://alicia-data.hjkl.lol")
	result, err := otel.Init(otel.Config{
		ServiceName:  "mcp-web",
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

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and run server
	server := NewServer(slog.Default())

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
	logger   *slog.Logger
	registry *tools.Registry
}

func NewServer(logger *slog.Logger) *Server {
	registry := tools.NewRegistry()

	// Register all tools
	registry.Register(tools.NewReadTool())
	registry.Register(tools.NewFetchRawTool())
	registry.Register(tools.NewFetchStructuredTool())
	registry.Register(tools.NewSearchTool())
	registry.Register(tools.NewExtractLinksTool())
	registry.Register(tools.NewExtractMetadataTool())
	registry.Register(tools.NewScreenshotTool())

	return &Server{
		logger:   logger,
		registry: registry,
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
		if response == nil {
			continue // Notification, no response needed
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
		return s.handleInitialize(req)
	case "initialized":
		return nil // Notification, no response
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "ping":
		return mcp.NewResponse(req.ID, map[string]any{})
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
			Name:    "mcp-web",
			Version: "1.0.0",
		},
	})
}

func (s *Server) handleToolsList(req *mcp.Request) *mcp.Response {
	return mcp.NewResponse(req.ID, map[string]any{
		"tools": s.registry.ListTools(),
	})
}

func (s *Server) handleToolsCall(ctx context.Context, req *mcp.Request) *mcp.Response {
	params, err := mcp.DecodeParams(req.Params)
	if err != nil {
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
	}

	// Start span with trace context from _meta
	ctx, span := otel.StartMCPToolSpan(ctx, "mcp-web", "web", params.Name, params.Meta)
	defer span.End()

	tool, exists := s.registry.Get(params.Name)
	if !exists {
		otel.RecordToolError(span, fmt.Errorf("unknown tool: %s", params.Name))
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	result, err := tool.Execute(ctx, params.Arguments)
	isError := err != nil
	var content string
	if err != nil {
		content = fmt.Sprintf("Error: %v", err)
		span.RecordError(err)
	} else {
		content = result
	}

	otel.EndMCPToolSpan(span, isError, len(content))

	if isError {
		return mcp.NewResponse(req.ID, mcp.NewToolError(content))
	}
	return mcp.NewResponse(req.ID, mcp.NewToolResult(content))
}
