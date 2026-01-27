package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/longregen/alicia/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MCPClient is a stdio JSON-RPC client for the Model Context Protocol.
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	tools  []Tool
	nextID int
	mu     sync.Mutex
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewMCPClient spawns an MCP server and performs the initialization handshake.
func NewMCPClient(command string, args []string, env []string) (*MCPClient, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("start process: %w", err)
	}

	c := &MCPClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		nextID: 1,
	}

	if err := c.initialize(); err != nil {
		c.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := c.listTools(); err != nil {
		c.Close()
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return c, nil
}

func (c *MCPClient) Tools() []Tool {
	return c.tools
}

func (c *MCPClient) Call(ctx context.Context, toolName string, args map[string]any) (any, error) {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "mcp.call_tool",
		trace.WithAttributes(
			attribute.String("mcp.tool_name", toolName),
		))
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	params := map[string]any{
		"name":      toolName,
		"arguments": args,
		"_meta":     otel.InjectMCPMeta(ctx),
	}

	result, err := c.request("tools/call", params)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	var callResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &callResult); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("parse call result: %w", err)
	}

	if len(callResult.Content) == 1 && callResult.Content[0].Type == "text" {
		span.SetAttributes(attribute.Int("mcp.result_length", len(callResult.Content[0].Text)))
		return callResult.Content[0].Text, nil
	}

	var raw any
	if err := json.Unmarshal(result, &raw); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("parse raw result: %w", err)
	}
	return raw, nil
}

func (c *MCPClient) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

func (c *MCPClient) initialize() error {
	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "agent-2",
			"version": "1.0",
		},
	}

	_, err := c.request("initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	notification := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}
	if err := c.send(notification); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}

	return nil
}

func (c *MCPClient) listTools() error {
	result, err := c.request("tools/list", map[string]any{})
	if err != nil {
		return err
	}

	var listResult struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(result, &listResult); err != nil {
		return fmt.Errorf("parse tools list: %w", err)
	}

	c.tools = make([]Tool, len(listResult.Tools))
	for i, t := range listResult.Tools {
		c.tools[i] = Tool{
			Name:        t.Name,
			Description: t.Description,
			Schema:      t.InputSchema,
		}
	}

	return nil
}

// request sends a JSON-RPC request and waits for the response. Caller must hold mu.
func (c *MCPClient) request(method string, params any) (json.RawMessage, error) {
	id := c.nextID
	c.nextID++

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.send(req); err != nil {
		return nil, err
	}

	return c.receive(id)
}

func (c *MCPClient) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (c *MCPClient) receive(expectedID int) (json.RawMessage, error) {
	for {
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			slog.Warn("mcp: invalid response", "raw", string(line))
			continue
		}

		if resp.ID == nil {
			continue // notification
		}
		if *resp.ID != expectedID {
			slog.Warn("mcp: unexpected response id", "got", *resp.ID, "expected", expectedID)
			continue
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		return resp.Result, nil
	}
}

// MCPManager manages multiple MCP clients with namespaced tools.
type MCPManager struct {
	clients map[string]*MCPClient
	tools   []Tool
	toolMap map[string]string // "server:tool" -> server name
	mu      sync.RWMutex
}

func NewMCPManager(servers []MCPServerConfig) (*MCPManager, error) {
	m := &MCPManager{
		clients: make(map[string]*MCPClient),
		toolMap: make(map[string]string),
	}

	baseEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}

	// Per-server env vars to forward from the host environment.
	serverEnvVars := map[string][]string{
		"garden":    {"GARDEN_DATABASE_URL"},
		"web":       {"KAGI_API_KEY"},
		"assistant": {"AGENT_SECRET", "OTEL_EXPORTER_OTLP_ENDPOINT"},
	}

	for _, srv := range servers {
		if srv.TransportType != "stdio" {
			slog.Warn("mcp: unsupported transport", "server", srv.Name, "type", srv.TransportType)
			continue
		}
		if srv.Command == "" {
			slog.Warn("mcp: no command", "server", srv.Name)
			continue
		}

		env := append([]string{}, baseEnv...)
		for _, key := range serverEnvVars[srv.Name] {
			if val := os.Getenv(key); val != "" {
				env = append(env, key+"="+val)
			}
		}
		// Special case: derive WS_URL from SERVER_URL
		if srv.Name == "assistant" {
			if serverURL := os.Getenv("SERVER_URL"); serverURL != "" {
				wsURL := strings.Replace(serverURL, "https://", "wss://", 1)
				wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
				env = append(env, "WS_URL="+wsURL)
			}
		}

		client, err := NewMCPClient(srv.Command, srv.Args, env)
		if err != nil {
			slog.Error("mcp server failed to start", "server", srv.Name, "error", err)
			continue
		}

		m.clients[srv.Name] = client
		for _, tool := range client.Tools() {
			prefixedName := srv.Name + ":" + tool.Name
			m.tools = append(m.tools, Tool{
				Name:        prefixedName,
				Description: fmt.Sprintf("[%s] %s", srv.Name, tool.Description),
				Schema:      tool.Schema,
			})
			m.toolMap[prefixedName] = srv.Name
		}

		slog.Info("mcp server started", "server", srv.Name, "tool_count", len(client.Tools()))
	}

	if len(m.clients) == 0 {
		return nil, fmt.Errorf("no MCP servers started")
	}

	return m, nil
}

func (m *MCPManager) Tools() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools
}

func (m *MCPManager) Call(ctx context.Context, toolName string, args map[string]any) (any, error) {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "mcp_manager.call",
		trace.WithAttributes(
			attribute.String("mcp.tool_name", toolName),
		))
	defer span.End()

	m.mu.RLock()
	serverName, ok := m.toolMap[toolName]
	client := m.clients[serverName]
	m.mu.RUnlock()

	if !ok {
		err := fmt.Errorf("unknown tool: %s", toolName)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.String("mcp.server_name", serverName))
	actualToolName := strings.TrimPrefix(toolName, serverName+":")
	span.SetAttributes(attribute.String("mcp.actual_tool_name", actualToolName))

	return client.Call(ctx, actualToolName, args)
}

func (m *MCPManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			slog.Error("error closing mcp server", "server", name, "error", err)
			lastErr = err
		}
	}
	return lastErr
}
