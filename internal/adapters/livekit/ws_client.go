package livekit

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

// WSClientConfig contains configuration for the WebSocket client
type WSClientConfig struct {
	// URL is the WebSocket URL to connect to (e.g., ws://localhost:8000/api/v1/ws)
	URL string
	// ReconnectInterval is the time to wait before attempting to reconnect
	ReconnectInterval time.Duration
	// PingInterval is the interval for sending ping messages
	PingInterval time.Duration
	// ReadTimeout is the read deadline for the WebSocket connection
	ReadTimeout time.Duration
	// WriteTimeout is the write deadline for the WebSocket connection
	WriteTimeout time.Duration
}

// DefaultWSClientConfig returns the default WebSocket client configuration
func DefaultWSClientConfig() *WSClientConfig {
	return &WSClientConfig{
		URL:               "ws://localhost:8000/api/v1/ws",
		ReconnectInterval: 5 * time.Second,
		PingInterval:      30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
}

// WSClientCallbacks defines callbacks for WebSocket client events
type WSClientCallbacks interface {
	// OnResponseGenerationRequest is called when a ResponseGenerationRequest is received
	OnResponseGenerationRequest(ctx context.Context, req *protocol.ResponseGenerationRequest) error
	// OnConnected is called when the WebSocket connection is established
	OnConnected()
	// OnDisconnected is called when the WebSocket connection is lost
	OnDisconnected(err error)
}

// WSClient manages a WebSocket connection to the serve process
type WSClient struct {
	config    *WSClientConfig
	callbacks WSClientCallbacks

	mu           sync.RWMutex
	conn         *websocket.Conn
	connected    bool
	reconnecting bool
	stanzaID     int32 // Client stanza ID counter (positive, incrementing)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// reconnectChan signals when reconnection should be attempted
	reconnectChan chan struct{}
}

// NewWSClient creates a new WebSocket client
func NewWSClient(config *WSClientConfig, callbacks WSClientCallbacks) *WSClient {
	if config == nil {
		config = DefaultWSClientConfig()
	}
	return &WSClient{
		config:        config,
		callbacks:     callbacks,
		reconnectChan: make(chan struct{}, 1),
	}
}

// Connect establishes the WebSocket connection and starts the read/write pumps.
// It also starts a reconnection loop that automatically reconnects on disconnection.
func (c *WSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected || c.reconnecting {
		c.mu.Unlock()
		return nil
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.mu.Unlock()

	// Start the reconnection loop
	c.wg.Add(1)
	go c.reconnectLoop()

	// Attempt initial connection
	return c.connect()
}

func (c *WSClient) connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(c.config.URL, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.stanzaID = 0
	c.mu.Unlock()

	log.Printf("WSClient: Connected to %s", c.config.URL)

	// Subscribe as agent
	if err := c.subscribeAsAgent(); err != nil {
		c.conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.connected = false
		c.mu.Unlock()
		return err
	}

	if c.callbacks != nil {
		c.callbacks.OnConnected()
	}

	// Start read and write pumps
	c.wg.Add(2)
	go c.readPump()
	go c.writePump()

	return nil
}

// subscribeAsAgent sends a subscription request with AgentMode=true
func (c *WSClient) subscribeAsAgent() error {
	c.mu.Lock()
	c.stanzaID++
	stanzaID := c.stanzaID
	c.mu.Unlock()

	req := dto.SubscribeRequest{
		AgentMode: true,
	}

	envelope := protocol.NewEnvelope(stanzaID, "", protocol.TypeSubscribe, req)
	data, err := msgpack.Marshal(envelope)
	if err != nil {
		return err
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil
	}

	conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return err
	}

	log.Printf("WSClient: Subscribed as agent")
	return nil
}

// reconnectLoop handles automatic reconnection with exponential backoff
func (c *WSClient) reconnectLoop() {
	defer c.wg.Done()

	backoff := c.config.ReconnectInterval
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.reconnectChan:
			c.mu.Lock()
			if c.connected {
				c.mu.Unlock()
				continue
			}
			c.reconnecting = true
			c.mu.Unlock()

			log.Printf("WSClient: Attempting to reconnect in %v...", backoff)

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(backoff):
			}

			err := c.connect()
			if err != nil {
				log.Printf("WSClient: Reconnection failed: %v", err)
				// Exponential backoff with max limit
				backoff = backoff * 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				// Trigger another reconnection attempt
				c.triggerReconnect()
			} else {
				// Reset backoff on successful connection
				backoff = c.config.ReconnectInterval
				c.mu.Lock()
				c.reconnecting = false
				c.mu.Unlock()
			}
		}
	}
}

// triggerReconnect signals that a reconnection attempt should be made
func (c *WSClient) triggerReconnect() {
	select {
	case c.reconnectChan <- struct{}{}:
	default:
		// Channel already has a pending reconnect signal
	}
}

// Disconnect closes the WebSocket connection
func (c *WSClient) Disconnect() {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return
	}
	c.connected = false
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	c.wg.Wait()
	log.Printf("WSClient: Disconnected")
}

// IsConnected returns true if the client is connected
func (c *WSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer c.wg.Done()
	defer func() {
		c.mu.Lock()
		wasConnected := c.connected
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		if c.callbacks != nil {
			c.callbacks.OnDisconnected(nil)
		}

		// Trigger reconnection if we were previously connected
		if wasConnected {
			c.triggerReconnect()
		}
	}()

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WSClient: Read error: %v", err)
			}
			return
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		// Decode envelope
		var envelope protocol.Envelope
		if err := msgpack.Unmarshal(data, &envelope); err != nil {
			log.Printf("WSClient: Failed to decode envelope: %v", err)
			continue
		}

		c.handleMessage(&envelope)
	}
}

// handleMessage processes incoming messages
func (c *WSClient) handleMessage(envelope *protocol.Envelope) {
	switch envelope.Type {
	case protocol.TypeSubscribeAck:
		log.Printf("WSClient: Received subscription acknowledgement")

	case protocol.TypeResponseGenerationRequest:
		// Decode body as ResponseGenerationRequest
		bodyBytes, err := msgpack.Marshal(envelope.Body)
		if err != nil {
			log.Printf("WSClient: Failed to encode body for ResponseGenerationRequest: %v", err)
			return
		}

		var req protocol.ResponseGenerationRequest
		if err := msgpack.Unmarshal(bodyBytes, &req); err != nil {
			log.Printf("WSClient: Failed to decode ResponseGenerationRequest: %v", err)
			return
		}

		log.Printf("WSClient: Received ResponseGenerationRequest (type: %s, messageID: %s, conversationID: %s)",
			req.RequestType, req.MessageID, req.ConversationID)

		if c.callbacks != nil {
			go func() {
				if err := c.callbacks.OnResponseGenerationRequest(c.ctx, &req); err != nil {
					log.Printf("WSClient: Error handling ResponseGenerationRequest: %v", err)
				}
			}()
		}

	default:
		log.Printf("WSClient: Received unknown message type: %d", envelope.Type)
	}
}

// writePump sends periodic ping messages
func (c *WSClient) writePump() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			connected := c.connected
			c.mu.RUnlock()

			if !connected || conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WSClient: Failed to send ping: %v", err)
				// Close the connection to unblock readPump
				c.mu.Lock()
				if c.conn != nil {
					c.conn.Close()
				}
				c.mu.Unlock()
				return
			}
		}
	}
}

// SendEnvelope sends an envelope message to the serve process
func (c *WSClient) SendEnvelope(envelope *protocol.Envelope) error {
	c.mu.RLock()
	conn := c.conn
	connected := c.connected
	c.mu.RUnlock()

	if !connected || conn == nil {
		return nil
	}

	// Set stanza ID for outgoing message
	c.mu.Lock()
	c.stanzaID++
	envelope.StanzaID = c.stanzaID
	c.mu.Unlock()

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		return err
	}

	c.mu.RLock()
	conn = c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil
	}

	conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

// SendAssistantMessage sends an assistant message response
func (c *WSClient) SendAssistantMessage(conversationID string, msg *protocol.AssistantMessage) error {
	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeAssistantMessage, msg)
	return c.SendEnvelope(envelope)
}

// SendAssistantSentence sends a streaming sentence chunk
func (c *WSClient) SendAssistantSentence(conversationID string, sentence *protocol.AssistantSentence) error {
	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeAssistantSentence, sentence)
	return c.SendEnvelope(envelope)
}

// SendToolUseRequest sends a tool use request
func (c *WSClient) SendToolUseRequest(conversationID string, req *protocol.ToolUseRequest) error {
	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeToolUseRequest, req)
	return c.SendEnvelope(envelope)
}

// SendMemoryTrace sends a memory trace message
func (c *WSClient) SendMemoryTrace(conversationID string, trace *protocol.MemoryTrace) error {
	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeMemoryTrace, trace)
	return c.SendEnvelope(envelope)
}

// SendReasoningStep sends a reasoning step
func (c *WSClient) SendReasoningStep(conversationID string, step *protocol.ReasoningStep) error {
	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeReasoningStep, step)
	return c.SendEnvelope(envelope)
}

// SendStartAnswer sends a start answer message
func (c *WSClient) SendStartAnswer(conversationID string, start *protocol.StartAnswer) error {
	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeStartAnswer, start)
	return c.SendEnvelope(envelope)
}
