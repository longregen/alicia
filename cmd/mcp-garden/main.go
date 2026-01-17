package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Setup logging to stderr (stdout is for MCP protocol)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := LoadConfig()

	if cfg.DatabaseURL == "" {
		slog.Error("GARDEN_DATABASE_URL or DATABASE_URL environment variable required")
		os.Exit(1)
	}

	// Connect to database
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	slog.Info("connected to garden database")

	// Initialize LLM client
	llmClient := NewLLMClient()
	if llmClient.IsConfigured() {
		slog.Info("LLM client configured", "model", llmClient.model)
	} else {
		slog.Info("LLM not configured, using fallback hints (set LLM_API_KEY to enable)")
	}

	// Create and run server
	server := NewServer(pool, cfg, llmClient, logger)

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

		var request JSONRPCRequest
		if err := decoder.Decode(&request); err != nil {
			if err.Error() == "EOF" {
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
				Name:    "garden-db",
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
			Tools: s.getTools(),
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
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("unknown tool: %s", params.Name),
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolCallResult{
			Content: []ContentBlock{
				{
					Type: "text",
					Text: result,
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
