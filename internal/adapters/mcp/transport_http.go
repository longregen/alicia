package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPSSETransport implements Transport using HTTP and Server-Sent Events
type HTTPSSETransport struct {
	baseURL   string
	apiKey    string
	client    *http.Client
	receiveCh chan Message
	closeCh   chan struct{}
	closeOnce sync.Once
	mu        sync.RWMutex
	connected bool
	sessionID string
}

// NewHTTPSSETransport creates a new HTTP/SSE transport
func NewHTTPSSETransport(baseURL, apiKey string) (*HTTPSSETransport, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")

	transport := &HTTPSSETransport{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		receiveCh: make(chan Message, 10),
		closeCh:   make(chan struct{}),
		connected: false,
	}

	return transport, nil
}

// Connect establishes the SSE connection
func (t *HTTPSSETransport) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/sse", nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("MCP: error reading error response body: %v", err)
			resp.Body.Close()
			return fmt.Errorf("SSE connection failed: %s (error reading response body)", resp.Status)
		}
		resp.Body.Close()
		return fmt.Errorf("SSE connection failed: %s - %s", resp.Status, string(body))
	}

	t.mu.Lock()
	t.connected = true
	t.mu.Unlock()

	// Start reading SSE events
	go t.readSSE(resp.Body)

	return nil
}

// Send sends a message to the MCP server via HTTP POST
func (t *HTTPSSETransport) Send(ctx context.Context, message any) error {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return fmt.Errorf("transport not connected")
	}
	t.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+"/message", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	if t.sessionID != "" {
		req.Header.Set("X-Session-ID", t.sessionID)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("MCP: error reading error response body: %v", err)
			return fmt.Errorf("server error: %s (error reading response body)", resp.Status)
		}
		return fmt.Errorf("server error: %s - %s", resp.Status, string(body))
	}

	// Extract session ID from response if present
	if sessionID := resp.Header.Get("X-Session-ID"); sessionID != "" {
		t.mu.Lock()
		t.sessionID = sessionID
		t.mu.Unlock()
	}

	return nil
}

// Receive returns a channel for receiving messages
func (t *HTTPSSETransport) Receive() <-chan Message {
	return t.receiveCh
}

// Close closes the transport
func (t *HTTPSSETransport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.closeCh)

		t.mu.Lock()
		t.connected = false
		t.mu.Unlock()

		close(t.receiveCh)
	})
	return err
}

// IsConnected returns true if the transport is connected
func (t *HTTPSSETransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// readSSE reads Server-Sent Events from the response body
func (t *HTTPSSETransport) readSSE(body io.ReadCloser) {
	defer body.Close()

	reader := bufio.NewReader(body)
	var eventType string
	var eventData []string

	for {
		select {
		case <-t.closeCh:
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					select {
					case t.receiveCh <- Message{Error: fmt.Errorf("SSE read error: %w", err)}:
					case <-t.closeCh:
					}
				}
				t.mu.Lock()
				t.connected = false
				t.mu.Unlock()
				return
			}

			line = strings.TrimRight(line, "\r\n")

			// Empty line indicates end of event
			if line == "" {
				if len(eventData) > 0 {
					t.processSSEEvent(eventType, eventData)
					eventType = ""
					eventData = nil
				}
				continue
			}

			// Parse SSE field
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(line[6:])
			} else if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(line[5:])
				eventData = append(eventData, data)
			}
			// Ignore other fields like id:, retry:, etc.
		}
	}
}

// processSSEEvent processes a complete SSE event
func (t *HTTPSSETransport) processSSEEvent(eventType string, eventData []string) {
	// Join multi-line data
	data := strings.Join(eventData, "\n")

	// For MCP, we expect JSON-RPC messages in the data field
	select {
	case t.receiveCh <- Message{Data: []byte(data)}:
	case <-t.closeCh:
	}
}
