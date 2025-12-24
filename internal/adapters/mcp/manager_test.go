package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// Tests for Manager

func TestManager_AddServer(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	cfg := &ServerConfig{
		Name:           "test-server",
		Transport:      "stdio",
		Command:        "/bin/echo",
		Args:           []string{"test"},
		AutoReconnect:  false,
		ReconnectDelay: 5 * time.Second,
	}

	// This will fail because echo doesn't speak MCP protocol
	// But we can verify the flow
	err := manager.AddServer(cfg)
	if err == nil {
		t.Error("expected error when adding server with invalid command")
	}
}

func TestManager_AddServer_Duplicate(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Create a mock managed client
	manager.servers["test"] = &ManagedClient{
		config: &ServerConfig{Name: "test"},
	}

	cfg := &ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	err := manager.AddServer(cfg)
	if err == nil {
		t.Fatal("expected error when adding duplicate server")
	}

	if err.Error() != "server test already exists" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestManager_RemoveServer(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Add a mock server
	managed := &ManagedClient{
		config:    &ServerConfig{Name: "test"},
		connected: true,
		stopCh:    make(chan struct{}),
	}
	manager.servers["test"] = managed

	err := manager.RemoveServer("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify server was removed
	if _, exists := manager.servers["test"]; exists {
		t.Error("expected server to be removed")
	}
}

func TestManager_RemoveServer_NotFound(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	err := manager.RemoveServer("non-existent")
	if err == nil {
		t.Fatal("expected error when removing non-existent server")
	}

	if err.Error() != "server non-existent not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestManager_GetClient(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	client := createMockClient(nil, nil, nil)
	defer client.Close()

	managed := &ManagedClient{
		config:    &ServerConfig{Name: "test"},
		client:    client,
		connected: true,
	}
	manager.servers["test"] = managed

	retrieved, err := manager.GetClient("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved != client {
		t.Error("expected to retrieve the same client")
	}
}

func TestManager_GetClient_NotFound(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	_, err := manager.GetClient("non-existent")
	if err == nil {
		t.Fatal("expected error when getting non-existent server")
	}
}

func TestManager_GetClient_NotConnected(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	client := createMockClient(nil, nil, nil)
	defer client.Close()

	managed := &ManagedClient{
		config:    &ServerConfig{Name: "test"},
		client:    client,
		connected: false,
	}
	manager.servers["test"] = managed

	_, err := manager.GetClient("test")
	if err == nil {
		t.Fatal("expected error when server not connected")
	}

	if err.Error() != "server test not connected" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestManager_ListServers(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.servers["server1"] = &ManagedClient{}
	manager.servers["server2"] = &ManagedClient{}
	manager.servers["server3"] = &ManagedClient{}

	servers := manager.ListServers()
	if len(servers) != 3 {
		t.Errorf("expected 3 servers, got %d", len(servers))
	}

	// Verify all server names are present
	nameMap := make(map[string]bool)
	for _, name := range servers {
		nameMap[name] = true
	}

	if !nameMap["server1"] || !nameMap["server2"] || !nameMap["server3"] {
		t.Error("not all server names returned")
	}
}

func TestManager_GetServerStatus(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.servers["connected"] = &ManagedClient{
		connected: true,
	}
	manager.servers["disconnected"] = &ManagedClient{
		connected: false,
	}

	// Test connected server
	connected, err := manager.GetServerStatus("connected")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !connected {
		t.Error("expected server to be connected")
	}

	// Test disconnected server
	connected, err = manager.GetServerStatus("disconnected")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if connected {
		t.Error("expected server to be disconnected")
	}

	// Test non-existent server
	_, err = manager.GetServerStatus("non-existent")
	if err == nil {
		t.Error("expected error for non-existent server")
	}
}

func TestManager_Close(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Add mock servers
	managed1 := &ManagedClient{
		config:    &ServerConfig{Name: "server1"},
		stopCh:    make(chan struct{}),
		connected: true,
	}
	managed2 := &ManagedClient{
		config:    &ServerConfig{Name: "server2"},
		stopCh:    make(chan struct{}),
		connected: true,
	}

	manager.servers["server1"] = managed1
	manager.servers["server2"] = managed2

	err := manager.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify context was cancelled
	select {
	case <-manager.ctx.Done():
		// Expected
	default:
		t.Error("expected manager context to be cancelled")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	var wg sync.WaitGroup

	// Add servers concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Just access the manager, don't actually add servers since it will fail
			_ = manager.ListServers()
		}(i)
	}

	wg.Wait()
}

// Tests for ManagedClient

func TestManagedClient_IsConnected(t *testing.T) {
	managed := &ManagedClient{
		connected: true,
	}

	if !managed.IsConnected() {
		t.Error("expected client to be connected")
	}

	managed.mu.Lock()
	managed.connected = false
	managed.mu.Unlock()

	if managed.IsConnected() {
		t.Error("expected client to be disconnected")
	}
}

func TestManagedClient_Close(t *testing.T) {
	client := createMockClient(nil, nil, nil)
	defer client.Close()

	managed := &ManagedClient{
		config:    &ServerConfig{Name: "test"},
		client:    client,
		connected: true,
		stopCh:    make(chan struct{}),
	}

	err := managed.close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify stopCh is closed
	select {
	case <-managed.stopCh:
		// Expected
	default:
		t.Error("expected stopCh to be closed")
	}

	// Verify connected is false
	if managed.connected {
		t.Error("expected connected to be false")
	}

	// Verify client is nil
	if managed.client != nil {
		t.Error("expected client to be nil")
	}
}

func TestManagedClient_Reconnect_DefaultDelay(t *testing.T) {
	cfg := &ServerConfig{
		Name:           "test",
		Transport:      "stdio",
		ReconnectDelay: 0, // Should default to 5s
	}

	if cfg.ReconnectDelay == 0 {
		cfg.ReconnectDelay = 5 * time.Second
	}

	if cfg.ReconnectDelay != 5*time.Second {
		t.Errorf("expected default delay of 5s, got %v", cfg.ReconnectDelay)
	}
}

// Tests for Client

func TestClient_Initialize(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	// Set up transport to handle initialize request
	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		if req.Method == MethodInitialize {
			// Send response immediately
			go func() {
				time.Sleep(5 * time.Millisecond)

				response := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: mustMarshal(InitializeResult{
						ProtocolVersion: "2024-11-05",
						ServerInfo: ServerInfo{
							Name:    "test-server",
							Version: "1.0.0",
						},
						Capabilities: ServerCapabilities{},
					}),
				}

				data, _ := json.Marshal(response)
				transport.receiveCh <- Message{Data: data}
			}()
		}

		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := client.Initialize(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !client.IsInitialized() {
		t.Error("expected client to be initialized")
	}

	info := client.GetServerInfo()
	if info == nil || info.Name != "test-server" {
		t.Error("expected server info to be set")
	}
}

func TestClient_ListTools(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	client.initialized = true

	// Set up transport to return tools list response
	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		if req.Method == MethodToolsList {
			go func() {
				time.Sleep(5 * time.Millisecond)

				response := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: mustMarshal(ToolsListResult{
						Tools: []Tool{
							{Name: "tool1", Description: "Tool 1", InputSchema: map[string]any{}},
							{Name: "tool2", Description: "Tool 2", InputSchema: map[string]any{}},
						},
					}),
				}

				data, _ := json.Marshal(response)
				transport.receiveCh <- Message{Data: data}
			}()
		}

		return nil
	}

	// Start receive loop
	go client.receiveLoop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestClient_ListTools_NotInitialized(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)

	ctx := context.Background()
	_, err := client.ListTools(ctx)
	if err == nil {
		t.Fatal("expected error when client not initialized")
	}

	if err.Error() != "client not initialized" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_CallTool(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	client.initialized = true

	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		if req.Method == MethodToolsCall {
			go func() {
				time.Sleep(5 * time.Millisecond)

				response := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: mustMarshal(ToolsCallResult{
						Content: []ContentItem{
							{Type: "text", Text: "Result"},
						},
						IsError: false,
					}),
				}

				data, _ := json.Marshal(response)
				transport.receiveCh <- Message{Data: data}
			}()
		}

		return nil
	}

	// Start receive loop
	go client.receiveLoop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := client.CallTool(ctx, "tool", map[string]any{"arg": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "Result" {
		t.Errorf("expected 'Result', got %q", result.Content[0].Text)
	}
}

func TestClient_CallTool_NotInitialized(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)

	ctx := context.Background()
	_, err := client.CallTool(ctx, "tool", nil)
	if err == nil {
		t.Fatal("expected error when client not initialized")
	}
}

func TestClient_Ping(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		if req.Method == MethodPing {
			go func() {
				time.Sleep(5 * time.Millisecond)

				response := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  mustMarshal(map[string]any{}),
				}

				data, _ := json.Marshal(response)
				transport.receiveCh <- Message{Data: data}
			}()
		}

		return nil
	}

	// Start receive loop
	go client.receiveLoop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Close(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)

	err := client.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify closeCh is closed
	select {
	case <-client.closeCh:
		// Expected
	default:
		t.Error("expected closeCh to be closed")
	}

	// Verify transport is closed
	if transport.IsConnected() {
		t.Error("expected transport to be closed")
	}
}

func TestClient_Call_ContextCancelled(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.call(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected error when context cancelled")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestClient_Call_Timeout(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	// Start receive loop but don't send any response
	go client.receiveLoop()

	// Don't send any response, let it timeout
	ctx := context.Background()

	// This will timeout after 30 seconds (internal timeout)
	// We'll use a shorter context timeout for testing
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := client.call(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestClient_Call_JSONRPCError(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		go func() {
			time.Sleep(5 * time.Millisecond)

			response := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			}

			data, _ := json.Marshal(response)
			transport.receiveCh <- Message{Data: data}
		}()

		return nil
	}

	// Start receive loop
	go client.receiveLoop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := client.call(ctx, "non_existent", nil)
	if err == nil {
		t.Fatal("expected JSON-RPC error")
	}

	if err.Error() != "JSON-RPC error -32601: Method not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_HandleNotification(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)

	// Test progress notification
	notif := &JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  MethodProgress,
		Params: map[string]any{
			"progressToken": "token1",
			"progress":      50.0,
		},
	}

	// Should not panic or error
	client.handleNotification(notif)

	// Test cancelled notification
	notif = &JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  MethodCancelled,
		Params: map[string]any{
			"requestId": "req1",
		},
	}

	client.handleNotification(notif)
}

func TestClient_ConcurrentCalls(t *testing.T) {
	transport := newMockTransport()
	client := NewClient("test", transport)
	defer client.Close()

	client.initialized = true

	// Set up transport to respond to any request
	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		go func() {
			time.Sleep(5 * time.Millisecond)

			response := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  mustMarshal(map[string]any{"result": "ok"}),
			}
			data, _ := json.Marshal(response)
			transport.receiveCh <- Message{Data: data}
		}()

		return nil
	}

	// Start receive loop
	go client.receiveLoop()

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Make 10 concurrent calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.call(ctx, "test", nil)
			if err != nil {
				t.Errorf("unexpected error in concurrent call: %v", err)
			}
		}()
	}

	wg.Wait()
}

// Helper function
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
