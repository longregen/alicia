package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/api/protocol"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/backoff"
)

type WSClient struct {
	cfg  *Config
	conn *websocket.Conn

	connected     bool
	subscriptions map[string]struct{}
	mu            sync.RWMutex
	writeMu       sync.Mutex

	onSentence          func(ctx context.Context, convID string, sentence *protocol.AssistantSentence)
	onGenerationStart   func(ctx context.Context, convID string, start *protocol.StartAnswer)
	onVoiceJoinRequest  func(req *protocol.VoiceJoinRequest)
	onVoiceLeaveRequest func(req *protocol.VoiceLeaveRequest)
	onPreferencesUpdate func(update *protocol.PreferencesUpdate)
}

func NewWSClient(cfg *Config) *WSClient {
	return &WSClient{
		cfg:           cfg,
		subscriptions: make(map[string]struct{}),
	}
}

func (c *WSClient) SetCallbacks(
	onSentence func(context.Context, string, *protocol.AssistantSentence),
	onGenerationStart func(context.Context, string, *protocol.StartAnswer),
	onVoiceJoinRequest func(*protocol.VoiceJoinRequest),
	onVoiceLeaveRequest func(*protocol.VoiceLeaveRequest),
) {
	c.onSentence = onSentence
	c.onGenerationStart = onGenerationStart
	c.onVoiceJoinRequest = onVoiceJoinRequest
	c.onVoiceLeaveRequest = onVoiceLeaveRequest
}

func (c *WSClient) SetPreferencesCallback(cb func(*protocol.PreferencesUpdate)) {
	c.onPreferencesUpdate = cb
}

func (c *WSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		slog.Info("ws: already connected")
		return nil
	}

	url := c.cfg.BackendWSURL
	if c.cfg.AgentSecret != "" {
		url += "?agent_secret=" + c.cfg.AgentSecret
	}

	slog.Info("ws: connecting", "url", c.cfg.BackendWSURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	if c.cfg.AgentSecret != "" {
		header.Set("Authorization", "Bearer "+c.cfg.AgentSecret)
		slog.Info("ws: using agent secret for auth")
	}

	conn, resp, err := dialer.DialContext(ctx, url, header)
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

	if err := c.subscribeVoiceMode(); err != nil {
		conn.Close()
		c.conn = nil
		c.connected = false
		return err
	}

	go c.readMessages(ctx)

	slog.Info("ws: connected to backend")
	return nil
}

func (c *WSClient) subscribeVoiceMode() error {
	sub := protocol.Subscribe{
		VoiceMode: true,
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

	slog.Info("ws: subscribed voice mode", "bytes", len(data))
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
	slog.Info("ws: disconnected from backend")
}

func (c *WSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *WSClient) Subscribe(convID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("cannot subscribe: not connected")
	}

	if _, ok := c.subscriptions[convID]; ok {
		return nil
	}

	sub := protocol.Subscribe{
		ConversationID: convID,
	}

	env := protocol.NewEnvelope(convID, protocol.TypeSubscribe, sub)
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

	c.subscriptions[convID] = struct{}{}
	return nil
}

func (c *WSClient) Unsubscribe(convID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if _, ok := c.subscriptions[convID]; !ok {
		return nil
	}

	unsub := protocol.Unsubscribe{
		ConversationID: convID,
	}

	env := protocol.NewEnvelope(convID, protocol.TypeUnsubscribe, unsub)
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

	delete(c.subscriptions, convID)
	return nil
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
			c.mu.Unlock()
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

func (c *WSClient) handleMessage(env *protocol.Envelope) {
	slog.Debug("ws: received message", "type", env.Type, "conversation_id", env.ConversationID)

	ctx := context.Background()
	if env.HasTraceContext() {
		ctx = otel.ExtractFromTraceContext(ctx, otel.TraceContext{
			TraceID:    env.TraceID,
			SpanID:     env.SpanID,
			TraceFlags: env.TraceFlags,
			SessionID:  env.SessionID,
			UserID:     env.UserID,
		})
	}

	switch env.Type {
	case protocol.TypeAssistantSentence:
		sentence, err := protocol.DecodeBody[protocol.AssistantSentence](env)
		if err != nil {
			slog.Error("ws: decode sentence error", "error", err)
			return
		}
		slog.Debug("ws: received sentence", "message_id", sentence.MessageID, "preview", truncateString(sentence.Text, 50))
		if c.onSentence != nil {
			c.onSentence(ctx, env.ConversationID, sentence)
		}

	case protocol.TypeStartAnswer:
		start, err := protocol.DecodeBody[protocol.StartAnswer](env)
		if err != nil {
			slog.Error("ws: decode start answer error", "error", err)
			return
		}
		slog.Debug("ws: received generation start", "message_id", start.MessageID)
		if c.onGenerationStart != nil {
			c.onGenerationStart(ctx, env.ConversationID, start)
		}

	case protocol.TypeSubscribeAck:
		ack, err := protocol.DecodeBody[protocol.SubscribeAck](env)
		if err != nil {
			slog.Error("ws: decode subscribe ack error", "error", err)
			return
		}
		if !ack.Success {
			slog.Error("ws: subscribe failed", "conversation_id", ack.ConversationID, "error", ack.Error)
		}

	case protocol.TypeVoiceJoinRequest:
		req, err := protocol.DecodeBody[protocol.VoiceJoinRequest](env)
		if err != nil {
			slog.Error("ws: decode voice join request error", "error", err)
			return
		}
		if req.UserID == "" && env.UserID != "" {
			req.UserID = env.UserID
		}
		if c.onVoiceJoinRequest != nil {
			c.onVoiceJoinRequest(req)
		}

	case protocol.TypeVoiceLeaveRequest:
		req, err := protocol.DecodeBody[protocol.VoiceLeaveRequest](env)
		if err != nil {
			slog.Error("ws: decode voice leave request error", "error", err)
			return
		}
		if c.onVoiceLeaveRequest != nil {
			c.onVoiceLeaveRequest(req)
		}

	case protocol.TypePreferencesUpdate:
		update, err := protocol.DecodeBody[protocol.PreferencesUpdate](env)
		if err != nil {
			slog.Error("ws: decode preferences update error", "error", err)
			return
		}
		if c.onPreferencesUpdate != nil {
			c.onPreferencesUpdate(update)
		}
	}
}

func (c *WSClient) Reconnect(ctx context.Context) error {
	c.Disconnect()

	err := backoff.RetryWithCallback(ctx, backoff.Quick, func(ctx context.Context, attempt int) error {
		if err := c.Connect(ctx); err != nil {
			return err
		}

		c.mu.Lock()
		subs := make([]string, 0, len(c.subscriptions))
		for convID := range c.subscriptions {
			subs = append(subs, convID)
		}
		c.subscriptions = make(map[string]struct{})
		c.mu.Unlock()

		for _, convID := range subs {
			if err := c.Subscribe(convID); err != nil {
				slog.Error("ws: failed to resubscribe", "conversation_id", convID, "error", err)
			}
		}

		return nil
	}, func(attempt int, err error, delay time.Duration) {
		slog.Warn("ws: reconnect attempt failed", "attempt", attempt, "error", err, "retry_in", delay)
	})

	return err
}

func (c *WSClient) SendUserMessage(convID, userID, text string) error {
	env := protocol.NewEnvelope(convID, protocol.TypeUserMessage, protocol.UserMessage{
		ConversationID: convID,
		Content:        text,
	})
	env.UserID = userID
	return c.writeEnvelope(env)
}

func (c *WSClient) SendVoiceJoinAck(convID string, success bool, errMsg string, sampleRate int) error {
	return c.writeEnvelope(protocol.NewEnvelope(convID, protocol.TypeVoiceJoinAck, protocol.VoiceJoinAck{
		ConversationID: convID,
		Success:        success,
		Error:          errMsg,
		SampleRate:     sampleRate,
	}))
}

func (c *WSClient) SendVoiceLeaveAck(convID string, success bool, errMsg string) error {
	return c.writeEnvelope(protocol.NewEnvelope(convID, protocol.TypeVoiceLeaveAck, protocol.VoiceLeaveAck{
		ConversationID: convID,
		Success:        success,
		Error:          errMsg,
	}))
}

func (c *WSClient) SendVoiceSpeaking(convID string, speaking *protocol.VoiceSpeaking) error {
	return c.writeEnvelope(protocol.NewEnvelope(convID, protocol.TypeVoiceSpeaking, speaking))
}

func (c *WSClient) SendVoiceStatus(convID string, status *protocol.VoiceStatus) error {
	return c.writeEnvelope(protocol.NewEnvelope(convID, protocol.TypeVoiceStatus, status))
}
