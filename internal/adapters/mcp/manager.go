package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ServerConfig represents configuration for an MCP server
type ServerConfig struct {
	Name           string        `json:"name"`
	Transport      string        `json:"transport"` // "stdio" or "http"
	Command        string        `json:"command,omitempty"`
	Args           []string      `json:"args,omitempty"`
	Env            []string      `json:"env,omitempty"`
	URL            string        `json:"url,omitempty"`
	APIKey         string        `json:"api_key,omitempty"`
	AutoReconnect  bool          `json:"auto_reconnect"`
	ReconnectDelay time.Duration `json:"reconnect_delay,omitempty"`
}

// ConnectionCallback is called when a server's connection status changes
type ConnectionCallback func(serverName string, connected bool)

// Manager manages multiple MCP server connections
type Manager struct {
	servers            map[string]*ManagedClient
	mu                 sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	connectionCallback ConnectionCallback
}

// ManagedClient wraps a client with reconnection logic
type ManagedClient struct {
	config       *ServerConfig
	client       *Client
	transport    Transport
	mu           sync.RWMutex
	connected    bool
	reconnecting bool
	stopCh       chan struct{}
	manager      *Manager // Reference to parent manager for callbacks
}

// NewManager creates a new MCP manager
func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)
	return &Manager{
		servers: make(map[string]*ManagedClient),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetConnectionCallback sets a callback that will be called when server connection status changes.
// The callback receives the server name and the new connection status (true = connected, false = disconnected).
func (m *Manager) SetConnectionCallback(callback ConnectionCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionCallback = callback
}

// notifyConnectionChange calls the connection callback if set
func (m *Manager) notifyConnectionChange(serverName string, connected bool) {
	m.mu.RLock()
	callback := m.connectionCallback
	m.mu.RUnlock()

	if callback != nil {
		callback(serverName, connected)
	}
}

// AddServer adds and connects to an MCP server
func (m *Manager) AddServer(config *ServerConfig) error {
	m.mu.Lock()

	if _, exists := m.servers[config.Name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("server %s already exists", config.Name)
	}

	managed := &ManagedClient{
		config:  config,
		stopCh:  make(chan struct{}),
		manager: m, // Set reference to parent manager for callbacks
	}

	// Set default reconnect delay
	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 5 * time.Second
	}

	// Initial connection
	if err := managed.connect(m.ctx); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to connect to server %s: %w", config.Name, err)
	}

	m.servers[config.Name] = managed
	autoReconnect := config.AutoReconnect
	serverName := config.Name
	m.mu.Unlock()

	// Notify that server is now connected (must be called outside the lock to avoid deadlock)
	m.notifyConnectionChange(serverName, true)

	// Start reconnection monitor if enabled
	if autoReconnect {
		go managed.monitorConnection(m.ctx)
	}

	return nil
}

// RemoveServer removes and disconnects from an MCP server
func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	managed, exists := m.servers[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("server %s not found", name)
	}
	delete(m.servers, name)
	m.mu.Unlock()

	return managed.close()
}

// GetClient returns the client for a specific server
func (m *Manager) GetClient(name string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	managed, exists := m.servers[name]
	if !exists {
		return nil, fmt.Errorf("server %s not found", name)
	}

	managed.mu.RLock()
	defer managed.mu.RUnlock()

	if !managed.connected {
		return nil, fmt.Errorf("server %s not connected", name)
	}

	return managed.client, nil
}

// ListServers returns a list of all server names
func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

// GetServerStatus returns the connection status for a server
func (m *Manager) GetServerStatus(name string) (bool, error) {
	m.mu.RLock()
	managed, exists := m.servers[name]
	m.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("server %s not found", name)
	}

	managed.mu.RLock()
	defer managed.mu.RUnlock()
	return managed.connected, nil
}

// Close closes all server connections
func (m *Manager) Close() error {
	m.cancel()

	// Collect all managed clients under lock, then close them outside the lock
	// to avoid deadlock (close() calls notifyConnectionChange which needs the lock)
	m.mu.Lock()
	clients := make([]*ManagedClient, 0, len(m.servers))
	for _, managed := range m.servers {
		clients = append(clients, managed)
	}
	m.mu.Unlock()

	var lastErr error
	for _, managed := range clients {
		if err := managed.close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// connect establishes a connection to the MCP server
func (mc *ManagedClient) connect(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Create transport based on config
	var transport Transport
	var err error

	switch mc.config.Transport {
	case "stdio":
		transport, err = NewStdioTransport(mc.config.Command, mc.config.Args, mc.config.Env)
		if err != nil {
			return fmt.Errorf("failed to create stdio transport: %w", err)
		}

	case "http", "sse":
		httpTransport, err := NewHTTPSSETransport(mc.config.URL, mc.config.APIKey)
		if err != nil {
			return fmt.Errorf("failed to create HTTP transport: %w", err)
		}

		// Connect to SSE endpoint
		if err := httpTransport.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect HTTP transport: %w", err)
		}

		transport = httpTransport

	default:
		return fmt.Errorf("unsupported transport type: %s", mc.config.Transport)
	}

	// Create client
	client := NewClient(mc.config.Name, transport)

	// Initialize the client
	if err := client.Initialize(ctx); err != nil {
		transport.Close()
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	mc.transport = transport
	mc.client = client
	mc.connected = true

	log.Printf("Connected to MCP server: %s", mc.config.Name)
	return nil
}

// close closes the connection to the MCP server
func (mc *ManagedClient) close() error {
	close(mc.stopCh)

	mc.mu.Lock()
	wasConnected := mc.connected
	mc.connected = false

	var err error
	if mc.client != nil {
		err = mc.client.Close()
		mc.client = nil
	}

	if mc.transport != nil {
		mc.transport = nil
	}
	mc.mu.Unlock()

	// Notify about disconnection if we were previously connected
	if wasConnected && mc.manager != nil {
		mc.manager.notifyConnectionChange(mc.config.Name, false)
	}

	return err
}

// monitorConnection monitors the connection and reconnects if necessary
func (mc *ManagedClient) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopCh:
			return
		case <-ticker.C:
			mc.mu.RLock()
			connected := mc.connected
			reconnecting := mc.reconnecting
			mc.mu.RUnlock()

			if !connected && !reconnecting {
				go mc.reconnect(ctx)
			}
		}
	}
}

// reconnect attempts to reconnect to the server
func (mc *ManagedClient) reconnect(ctx context.Context) {
	mc.mu.Lock()
	if mc.reconnecting {
		mc.mu.Unlock()
		return
	}
	mc.reconnecting = true
	mc.mu.Unlock()

	defer func() {
		mc.mu.Lock()
		mc.reconnecting = false
		mc.mu.Unlock()
	}()

	log.Printf("Attempting to reconnect to MCP server: %s", mc.config.Name)

	// Exponential backoff
	backoff := mc.config.ReconnectDelay
	maxBackoff := 60 * time.Second
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopCh:
			return
		case <-time.After(backoff):
			attempts++

			// Clean up old connection
			mc.mu.Lock()
			if mc.client != nil {
				mc.client.Close()
				mc.client = nil
			}
			if mc.transport != nil {
				mc.transport.Close()
				mc.transport = nil
			}
			mc.mu.Unlock()

			// Attempt to connect
			if err := mc.connect(ctx); err != nil {
				log.Printf("Reconnection attempt %d failed for %s: %v", attempts, mc.config.Name, err)

				// Increase backoff
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			log.Printf("Successfully reconnected to MCP server: %s after %d attempts", mc.config.Name, attempts)

			// Notify about successful reconnection
			if mc.manager != nil {
				mc.manager.notifyConnectionChange(mc.config.Name, true)
			}
			return
		}
	}
}

// IsConnected returns true if the client is connected
func (mc *ManagedClient) IsConnected() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.connected
}
