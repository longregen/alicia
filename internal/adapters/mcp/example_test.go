package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/adapters/mcp"
)

// Example_stdioTransport demonstrates using the stdio transport
func Example_stdioTransport() {
	ctx := context.Background()

	// Create a stdio transport
	transport, err := mcp.NewStdioTransport(
		"npx",
		[]string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		nil,
	)
	if err != nil {
		fmt.Printf("Failed to create transport: %v\n", err)
		return
	}
	defer transport.Close()

	// Create a client
	client := mcp.NewClient("filesystem", transport)
	defer client.Close()

	// Initialize
	if err := client.Initialize(ctx); err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		return
	}

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		fmt.Printf("Failed to list tools: %v\n", err)
		return
	}

	fmt.Printf("Discovered %d tools\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
}

// Example_httpTransport demonstrates using the HTTP/SSE transport
func Example_httpTransport() {
	ctx := context.Background()

	// Create HTTP transport
	transport, err := mcp.NewHTTPSSETransport("http://localhost:3000", "")
	if err != nil {
		fmt.Printf("Failed to create transport: %v\n", err)
		return
	}
	defer transport.Close()

	// Connect to SSE endpoint
	if err := transport.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Create client
	client := mcp.NewClient("remote", transport)
	defer client.Close()

	// Initialize
	if err := client.Initialize(ctx); err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		return
	}

	// Get server info
	if info := client.GetServerInfo(); info != nil {
		fmt.Printf("Connected to %s v%s\n", info.Name, info.Version)
	}
}

// Example_manager demonstrates using the connection manager
func Example_manager() {
	ctx := context.Background()

	// Create manager
	manager := mcp.NewManager(ctx)
	defer manager.Close()

	// Add a server
	config := &mcp.ServerConfig{
		Name:           "test",
		Transport:      "stdio",
		Command:        "npx",
		Args:           []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		AutoReconnect:  true,
		ReconnectDelay: 5 * time.Second,
	}

	if err := manager.AddServer(config); err != nil {
		fmt.Printf("Failed to add server: %v\n", err)
		return
	}

	// Get client
	client, err := manager.GetClient("test")
	if err != nil {
		fmt.Printf("Failed to get client: %v\n", err)
		return
	}

	// Use client
	tools, err := client.ListTools(ctx)
	if err != nil {
		fmt.Printf("Failed to list tools: %v\n", err)
		return
	}

	fmt.Printf("Server has %d tools\n", len(tools))
}

// TestProtocolSerialization tests JSON-RPC message serialization
func TestProtocolSerialization(t *testing.T) {
	// Test request serialization
	req := mcp.NewJSONRPCRequest(1, "tools/list", map[string]any{
		"cursor": "next",
	})

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded mcp.JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Method != "tools/list" {
		t.Errorf("Expected method 'tools/list', got '%s'", decoded.Method)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got '%s'", decoded.JSONRPC)
	}
}

// TestToolCallParams tests tool call parameter serialization
func TestToolCallParams(t *testing.T) {
	params := &mcp.ToolsCallParams{
		Name: "read_file",
		Arguments: map[string]any{
			"path": "/tmp/test.txt",
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	var decoded mcp.ToolsCallParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal params: %v", err)
	}

	if decoded.Name != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", decoded.Name)
	}

	path, ok := decoded.Arguments["path"].(string)
	if !ok || path != "/tmp/test.txt" {
		t.Errorf("Expected path '/tmp/test.txt', got '%v'", decoded.Arguments["path"])
	}
}

// BenchmarkJSONRPCSerialization benchmarks JSON-RPC message serialization
func BenchmarkJSONRPCSerialization(b *testing.B) {
	req := mcp.NewJSONRPCRequest(1, "tools/call", map[string]any{
		"name": "test_tool",
		"arguments": map[string]any{
			"arg1": "value1",
			"arg2": 42,
			"arg3": true,
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
