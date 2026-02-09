package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/shared/backoff"
	"github.com/longregen/alicia/shared/protocol"
)

type WSClient struct {
	cfg  *Config
	conn *websocket.Conn

	connected    bool
	reconnecting bool
	mu           sync.RWMutex
	writeMu      sync.Mutex

	onPairRequest func(role string)
}

func NewWSClient(cfg *Config) *WSClient {
	return &WSClient{cfg: cfg}
}

func (c *WSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	slog.Info("ws: connecting", "url", c.cfg.BackendWSURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	if c.cfg.AgentSecret != "" {
		header.Set("Authorization", "Bearer "+c.cfg.AgentSecret)
	}

	conn, resp, err := dialer.DialContext(ctx, c.cfg.BackendWSURL, header)
	if err != nil {
		if resp != nil {
			slog.Error("ws: connection failed", "status", resp.StatusCode, "error", err)
		} else {
			slog.Error("ws: connection failed", "error", err)
		}
		return err
	}

	c.conn = conn
	c.connected = true

	if err := c.subscribeWhatsAppMode(); err != nil {
		conn.Close()
		c.conn = nil
		c.connected = false
		return err
	}

	go c.readMessages(ctx)

	slog.Info("ws: connected to hub")
	return nil
}

func (c *WSClient) subscribeWhatsAppMode() error {
	sub := protocol.Subscribe{
		WhatsAppMode: true,
	}

	env := protocol.NewEnvelope("", protocol.TypeSubscribe, sub)
	data, err := env.Encode()
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = c.conn.WriteMessage(websocket.BinaryMessage, data)
	c.writeMu.Unlock()
	if err != nil {
		return err
	}

	slog.Info("ws: subscribed whatsapp mode")
	return nil
}

func (c *WSClient) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.connected = false
	slog.Info("ws: disconnected from hub")
}

func (c *WSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *WSClient) readMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("ws: read error", "error", err)
			}
			c.mu.Lock()
			c.connected = false
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			shouldReconnect := !c.reconnecting
			c.reconnecting = true
			c.mu.Unlock()
			if shouldReconnect {
				go func() {
					if err := c.Reconnect(ctx); err != nil {
						slog.Error("ws: reconnect failed", "error", err)
					}
				}()
			}
			return
		}

		env, err := protocol.DecodeEnvelope(data)
		if err != nil {
			slog.Error("ws: decode error", "error", err)
			continue
		}

		c.handleMessage(env)
	}
}

func (c *WSClient) handleMessage(env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypeSubscribeAck:
		ack, err := protocol.DecodeBody[protocol.SubscribeAck](env)
		if err != nil {
			slog.Error("ws: decode subscribe ack error", "error", err)
			return
		}
		if !ack.Success {
			slog.Error("ws: subscribe failed", "error", ack.Error)
		} else {
			slog.Info("ws: subscribe acknowledged")
		}

	case protocol.TypeWhatsAppPairRequest:
		req, err := protocol.DecodeBody[protocol.WhatsAppPairRequest](env)
		if err != nil {
			slog.Error("ws: decode pair request error", "error", err)
			return
		}
		slog.Info("ws: received pair request", "role", req.Role)
		if c.onPairRequest != nil {
			c.onPairRequest(req.Role)
		}

	default:
		slog.Debug("ws: unhandled message", "type", env.Type)
	}
}

func (c *WSClient) writeEnvelope(env *protocol.Envelope) error {
	data, err := env.Encode()
	if err != nil {
		return err
	}

	c.mu.RLock()
	conn := c.conn
	connected := c.connected
	c.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}

	c.writeMu.Lock()
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = conn.WriteMessage(websocket.BinaryMessage, data)
	c.writeMu.Unlock()
	return err
}

func (c *WSClient) SendWhatsAppQR(code, event, role string) error {
	return c.writeEnvelope(protocol.NewEnvelope("", protocol.TypeWhatsAppQR, protocol.WhatsAppQR{
		Code:  code,
		Event: event,
		Role:  role,
	}))
}

func (c *WSClient) SendWhatsAppStatus(connected bool, phone, errMsg, role string) error {
	return c.writeEnvelope(protocol.NewEnvelope("", protocol.TypeWhatsAppStatus, protocol.WhatsAppStatus{
		Connected: connected,
		Phone:     phone,
		Error:     errMsg,
		Role:      role,
	}))
}

func (c *WSClient) SendWhatsAppDebug(role, event, detail string) {
	if err := c.writeEnvelope(protocol.NewEnvelope("", protocol.TypeWhatsAppDebug, protocol.WhatsAppDebug{
		Role:   role,
		Event:  event,
		Detail: detail,
	})); err != nil {
		slog.Debug("ws: failed to send debug event", "role", role, "event", event, "error", err)
	}
}

func (c *WSClient) Reconnect(ctx context.Context) error {
	c.Disconnect()

	defer func() {
		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
	}()

	return backoff.RetryWithCallback(ctx, backoff.Quick, func(ctx context.Context, attempt int) error {
		return c.Connect(ctx)
	}, func(attempt int, err error, delay time.Duration) {
		slog.Warn("ws: reconnect attempt failed", "attempt", attempt, "error", err, "retry_in", delay)
	})
}
