package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
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

// AllowedURLHosts is a list of explicitly allowed hosts for MCP connections.
// If non-empty, only these hosts are permitted. If empty, URL validation
// relies on blocking private/internal addresses.
var AllowedURLHosts []string

// isPrivateIP checks if an IP address is private, loopback, or otherwise internal
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for loopback (127.0.0.0/8 for IPv4, ::1 for IPv6)
	if ip.IsLoopback() {
		return true
	}

	// Check for private addresses
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local addresses (169.254.0.0/16 for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for unspecified addresses (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Check for multicast addresses
	if ip.IsMulticast() {
		return true
	}

	// Check for IPv4-mapped IPv6 addresses that wrap private IPs
	if ip4 := ip.To4(); ip4 != nil {
		return isPrivateIP(ip4)
	}

	return false
}

// validateURL validates that a URL is safe for server-side requests.
// It prevents SSRF attacks by blocking requests to internal/private networks.
func validateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http and https are allowed)", parsedURL.Scheme)
	}

	// Extract hostname (without port)
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Check against allowlist if configured
	if len(AllowedURLHosts) > 0 {
		allowed := false
		for _, allowedHost := range AllowedURLHosts {
			if strings.EqualFold(hostname, allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("hostname %q is not in the allowed hosts list", hostname)
		}
		return nil
	}

	// Block common internal hostnames
	lowerHostname := strings.ToLower(hostname)
	internalHostnames := []string{
		"localhost",
		"localhost.localdomain",
		"local",
		"internal",
		"metadata",                    // Cloud metadata services
		"metadata.google.internal",    // GCP metadata
		"instance-data",               // AWS metadata alias
		"169.254.169.254",             // AWS/GCP/Azure metadata IP
		"metadata.azure.com",          // Azure metadata
		"kubernetes",                  // Kubernetes services
		"kubernetes.default",          // Kubernetes default service
		"kubernetes.default.svc",      // Kubernetes default service
		"kubernetes.default.svc.cluster.local", // Kubernetes FQDN
	}

	for _, internal := range internalHostnames {
		if lowerHostname == internal || strings.HasSuffix(lowerHostname, "."+internal) {
			return fmt.Errorf("hostname %q is not allowed: internal/metadata hostname", hostname)
		}
	}

	// Resolve hostname to IP addresses and check each one
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If we can't resolve the hostname, we should be cautious
		// However, this could be a legitimate hostname that's temporarily unresolvable
		// For security, we reject it
		return fmt.Errorf("cannot resolve hostname %q: %w", hostname, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("hostname %q resolves to private/internal IP address %s", hostname, ip.String())
		}
	}

	return nil
}

// NewHTTPSSETransport creates a new HTTP/SSE transport
func NewHTTPSSETransport(baseURL, apiKey string) (*HTTPSSETransport, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Validate URL to prevent SSRF attacks
	if err := validateURL(baseURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

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
