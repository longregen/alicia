package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type ServerConfig struct {
	Name           string        `json:"name"`
	Transport      string        `json:"transport"`
	Command        string        `json:"command,omitempty"`
	Args           []string      `json:"args,omitempty"`
	Env            []string      `json:"env,omitempty"`
	URL            string        `json:"url,omitempty"`
	APIKey         string        `json:"api_key,omitempty"`
	AutoReconnect  bool          `json:"auto_reconnect"`
	ReconnectDelay time.Duration `json:"reconnect_delay,omitempty"`
}

type ConnectionCallback func(serverName string, connected bool)

type Manager struct {
	servers            map[string]*ManagedClient
	mu                 sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	connectionCallback ConnectionCallback
}

type ManagedClient struct {
	config       *ServerConfig
	client       *Client
	transport    Transport
	mu           sync.RWMutex
	connected    bool
	reconnecting bool
	stopCh       chan struct{}
	manager      *Manager
}

func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)
	return &Manager{
		servers: make(map[string]*ManagedClient),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (m *Manager) SetConnectionCallback(callback ConnectionCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionCallback = callback
}

func (m *Manager) notifyConnectionChange(serverName string, connected bool) {
	m.mu.RLock()
	callback := m.connectionCallback
	m.mu.RUnlock()

	if callback != nil {
		callback(serverName, connected)
	}
}

func (m *Manager) AddServer(config *ServerConfig) error {
	m.mu.Lock()

	if _, exists := m.servers[config.Name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("server %s already exists", config.Name)
	}

	managed := &ManagedClient{
		config:  config,
		stopCh:  make(chan struct{}),
		manager: m,
	}

	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 5 * time.Second
	}

	if err := managed.connect(m.ctx); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to connect to server %s: %w", config.Name, err)
	}

	m.servers[config.Name] = managed
	autoReconnect := config.AutoReconnect
	serverName := config.Name
	m.mu.Unlock()

	// Notify outside the lock to avoid deadlock
	m.notifyConnectionChange(serverName, true)

	if autoReconnect {
		go managed.monitorConnection(m.ctx)
	}

	return nil
}

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

func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

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

func (m *Manager) Close() error {
	m.cancel()

	// Close clients outside the lock to avoid deadlock (close() calls notifyConnectionChange)
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

func (mc *ManagedClient) connect(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

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

	client := NewClient(mc.config.Name, transport)

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

	if wasConnected && mc.manager != nil {
		mc.manager.notifyConnectionChange(mc.config.Name, false)
	}

	return err
}

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

			if err := mc.connect(ctx); err != nil {
				log.Printf("Reconnection attempt %d failed for %s: %v", attempts, mc.config.Name, err)

				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			log.Printf("Successfully reconnected to MCP server: %s after %d attempts", mc.config.Name, attempts)

			if mc.manager != nil {
				mc.manager.notifyConnectionChange(mc.config.Name, true)
			}
			return
		}
	}
}

func (mc *ManagedClient) IsConnected() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.connected
}
