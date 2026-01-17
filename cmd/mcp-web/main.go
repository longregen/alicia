package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/longregen/alicia/cmd/mcp-web/tools"
)

func main() {
	// Setup logging to stderr (stdout is for MCP protocol)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and run server
	server := NewServer(logger)

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

		var request JSONRPCRequest
		if err := decoder.Decode(&request); err != nil {
			if err.Error() == "EOF" {
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

func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
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
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		}
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{
					ListChanged: false,
				},
			},
			ServerInfo: ServerInfo{
				Name:    "mcp-web",
				Version: "1.0.0",
			},
		},
	}
}

func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolsListResult{
			Tools: s.registry.ListTools(),
		},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := mapToStruct(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("invalid params: %v", err),
			},
		}
	}

	tool, exists := s.registry.Get(params.Name)
	if !exists {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("unknown tool: %s", params.Name),
			},
		}
	}

	result, err := tool.Execute(ctx, params.Arguments)
	isError := err != nil
	var content string
	if err != nil {
		content = fmt.Sprintf("Error: %v", err)
	} else {
		content = result
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolCallResult{
			Content: []ContentBlock{
				{
					Type: "text",
					Text: content,
				},
			},
			IsError: isError,
		},
	}
}

func mapToStruct(m any, v any) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
