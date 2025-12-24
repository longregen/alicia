package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Client represents an MCP client that can connect to MCP servers
type Client struct {
	name         string
	transport    Transport
	mu           sync.RWMutex
	nextID       atomic.Int64
	pendingCalls map[any]chan *JSONRPCResponse
	initialized  bool
	serverInfo   *ServerInfo
	capabilities *ServerCapabilities
	closeCh      chan struct{}
	closeOnce    sync.Once
}

// NewClient creates a new MCP client with the given transport
func NewClient(name string, transport Transport) *Client {
	return &Client{
		name:         name,
		transport:    transport,
		pendingCalls: make(map[any]chan *JSONRPCResponse),
		closeCh:      make(chan struct{}),
	}
}

// Initialize performs the MCP initialization handshake
func (c *Client) Initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities: ClientCapabilities{
			Experimental: map[string]any{},
		},
		ClientInfo: ClientInfo{
			Name:    "alicia",
			Version: "0.1.0",
		},
	}

	paramsMap := map[string]any{
		"protocolVersion": params.ProtocolVersion,
		"capabilities":    params.Capabilities,
		"clientInfo":      params.ClientInfo,
	}

	// Start receiving messages
	go c.receiveLoop()

	// Send initialize request
	result, err := c.call(ctx, MethodInitialize, paramsMap)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	var initResult InitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	c.mu.Lock()
	c.initialized = true
	c.serverInfo = &initResult.ServerInfo
	c.capabilities = &initResult.Capabilities
	c.mu.Unlock()

	// Send initialized notification
	if err := c.notify(ctx, MethodInitialized, nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// ListTools retrieves all available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	var allTools []Tool
	var cursor *string

	// Handle pagination
	for {
		params := map[string]any{}
		if cursor != nil {
			params["cursor"] = *cursor
		}

		result, err := c.call(ctx, MethodToolsList, params)
		if err != nil {
			return nil, fmt.Errorf("tools/list failed: %w", err)
		}

		var listResult ToolsListResult
		if err := json.Unmarshal(result, &listResult); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tools/list result: %w", err)
		}

		allTools = append(allTools, listResult.Tools...)

		if listResult.NextCursor == nil {
			break
		}
		cursor = listResult.NextCursor
	}

	return allTools, nil
}

// CallTool executes a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (*ToolsCallResult, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	params := map[string]any{
		"name": name,
	}
	if arguments != nil {
		params["arguments"] = arguments
	}

	result, err := c.call(ctx, MethodToolsCall, params)
	if err != nil {
		return nil, fmt.Errorf("tools/call failed: %w", err)
	}

	var callResult ToolsCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools/call result: %w", err)
	}

	return &callResult, nil
}

// Ping sends a ping request to check if the server is alive
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.call(ctx, MethodPing, map[string]any{})
	return err
}

// GetServerInfo returns information about the connected MCP server
func (c *Client) GetServerInfo() *ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// GetCapabilities returns the server's capabilities
func (c *Client) GetCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// IsInitialized returns true if the client has been initialized
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// Close closes the client and its transport
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closeCh)

		c.mu.Lock()
		// Close all pending calls
		for _, ch := range c.pendingCalls {
			close(ch)
		}
		c.pendingCalls = make(map[any]chan *JSONRPCResponse)
		c.initialized = false
		c.mu.Unlock()

		if c.transport != nil {
			err = c.transport.Close()
		}
	})
	return err
}

// call makes a JSON-RPC call and waits for the response
func (c *Client) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	req := NewJSONRPCRequest(id, method, params)

	// Create response channel
	respCh := make(chan *JSONRPCResponse, 1)
	c.mu.Lock()
	c.pendingCalls[id] = respCh
	c.mu.Unlock()

	// Ensure cleanup
	defer func() {
		c.mu.Lock()
		delete(c.pendingCalls, id)
		c.mu.Unlock()
	}()

	// Send request
	if err := c.transport.Send(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-respCh:
		if !ok {
			return nil, fmt.Errorf("response channel closed")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// notify sends a JSON-RPC notification (no response expected)
func (c *Client) notify(ctx context.Context, method string, params map[string]any) error {
	notif := NewJSONRPCNotification(method, params)
	return c.transport.Send(ctx, notif)
}

// receiveLoop processes incoming messages from the transport
func (c *Client) receiveLoop() {
	for {
		select {
		case <-c.closeCh:
			return
		case msg := <-c.transport.Receive():
			if msg.Error != nil {
				// Log error and continue
				continue
			}

			c.handleMessage(msg.Data)
		}
	}
}

// handleMessage handles an incoming message
func (c *Client) handleMessage(data []byte) {
	// Try to parse as response first
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err == nil && resp.ID != nil {
		c.handleResponse(&resp)
		return
	}

	// Try to parse as notification
	var notif JSONRPCNotification
	if err := json.Unmarshal(data, &notif); err == nil {
		c.handleNotification(&notif)
		return
	}

	// Unknown message format
}

// handleResponse handles a JSON-RPC response
func (c *Client) handleResponse(resp *JSONRPCResponse) {
	// Normalize ID type: JSON unmarshaling converts numbers to float64,
	// but we use int64 for IDs. Convert float64 to int64 for map lookup.
	id := resp.ID
	if f, ok := id.(float64); ok {
		id = int64(f)
	}

	c.mu.RLock()
	ch, exists := c.pendingCalls[id]
	c.mu.RUnlock()

	if !exists {
		// Unexpected response
		return
	}

	select {
	case ch <- resp:
	default:
		// Channel full or closed
	}
}

// handleNotification handles a JSON-RPC notification
func (c *Client) handleNotification(notif *JSONRPCNotification) {
	// Handle different notification types
	switch notif.Method {
	case MethodProgress:
		// Handle progress notifications
		// Could emit events for progress tracking
	case MethodCancelled:
		// Handle cancellation notifications
	default:
		// Unknown notification
	}
}
