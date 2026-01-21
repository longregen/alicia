package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

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

func NewClient(name string, transport Transport) *Client {
	return &Client{
		name:         name,
		transport:    transport,
		pendingCalls: make(map[any]chan *JSONRPCResponse),
		closeCh:      make(chan struct{}),
	}
}

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

	go c.receiveLoop()

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

	if err := c.notify(ctx, MethodInitialized, nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	var allTools []Tool
	var cursor *string

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

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.call(ctx, MethodPing, map[string]any{})
	return err
}

func (c *Client) GetServerInfo() *ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

func (c *Client) GetCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closeCh)

		c.mu.Lock()
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

func (c *Client) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	req := NewJSONRPCRequest(id, method, params)

	respCh := make(chan *JSONRPCResponse, 1)
	c.mu.Lock()
	c.pendingCalls[id] = respCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pendingCalls, id)
		c.mu.Unlock()
	}()

	if err := c.transport.Send(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

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

func (c *Client) notify(ctx context.Context, method string, params map[string]any) error {
	notif := NewJSONRPCNotification(method, params)
	return c.transport.Send(ctx, notif)
}

func (c *Client) receiveLoop() {
	for {
		select {
		case <-c.closeCh:
			return
		case msg := <-c.transport.Receive():
			if msg.Error != nil {
				continue
			}

			c.handleMessage(msg.Data)
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err == nil && resp.ID != nil {
		c.handleResponse(&resp)
		return
	}

	var notif JSONRPCNotification
	if err := json.Unmarshal(data, &notif); err == nil {
		c.handleNotification(&notif)
		return
	}
}

func (c *Client) handleResponse(resp *JSONRPCResponse) {
	// JSON unmarshaling converts numbers to float64, but we use int64 for IDs
	id := resp.ID
	if f, ok := id.(float64); ok {
		id = int64(f)
	}

	c.mu.RLock()
	ch, exists := c.pendingCalls[id]
	c.mu.RUnlock()

	if !exists {
		return
	}

	select {
	case ch <- resp:
	default:
	}
}

func (c *Client) handleNotification(notif *JSONRPCNotification) {
	switch notif.Method {
	case MethodProgress:
	case MethodCancelled:
	default:
	}
}
