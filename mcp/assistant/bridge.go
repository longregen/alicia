package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/longregen/alicia/shared/protocol"
)

const (
	ToolTimeout       = 30 * time.Second
	ReconnectDelayMin = 5 * time.Second
	ReconnectDelayMax = 60 * time.Second
	AssistantClientID = "__assistant__"
)

type Bridge struct {
	wsURL    string
	secret   string
	conn     *websocket.Conn
	connMu   sync.Mutex
	pending  map[string]chan *protocol.ToolUseResult
	mu       sync.Mutex
	done     chan struct{}
	doneOnce sync.Once
	closed   atomic.Bool
}

func NewBridge(wsURL, secret string) *Bridge {
	return &Bridge{
		wsURL:   wsURL,
		secret:  secret,
		pending: make(map[string]chan *protocol.ToolUseResult),
		done:    make(chan struct{}),
	}
}

func (b *Bridge) Connect(ctx context.Context) error {
	header := http.Header{}
	if b.secret != "" {
		header.Set("Authorization", "Bearer "+b.secret)
	}

	// Ensure URL uses WebSocket scheme
	wsURL := b.wsURL
	if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	b.connMu.Lock()
	if b.conn != nil {
		b.conn.Close()
	}
	b.conn = conn
	b.connMu.Unlock()

	// Subscribe as monitor to receive all messages (including ToolUseResult from assistant)
	sub := protocol.Subscribe{MonitorMode: true}
	env := protocol.NewEnvelope("", protocol.TypeSubscribe, sub)
	data, err := env.Encode()
	if err != nil {
		conn.Close()
		return fmt.Errorf("encode subscribe: %w", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		conn.Close()
		return fmt.Errorf("send subscribe: %w", err)
	}

	// Wait for ack
	_, ackData, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("read subscribe ack: %w", err)
	}
	ackEnv, err := protocol.DecodeEnvelope(ackData)
	if err != nil {
		conn.Close()
		return fmt.Errorf("decode subscribe ack: %w", err)
	}
	if ackEnv.Type == protocol.TypeSubscribeAck {
		ack, err := protocol.DecodeBody[protocol.SubscribeAck](ackEnv)
		if err != nil || !ack.Success {
			conn.Close()
			errMsg := "unknown"
			if ack != nil {
				errMsg = ack.Error
			}
			return fmt.Errorf("subscribe failed: %s", errMsg)
		}
	}

	slog.Info("bridge connected to hub")

	// Start read loop
	go b.readLoop()

	return nil
}

func (b *Bridge) closeDone() {
	b.doneOnce.Do(func() { close(b.done) })
}

func (b *Bridge) readLoop() {
	for {
		_, data, err := b.conn.ReadMessage()
		if err != nil {
			if b.closed.Load() {
				b.closeDone()
				return
			}
			slog.Error("bridge read error", "error", err)
			b.drainPending()
			b.reconnect()
			return
		}

		var frame struct {
			Src  string `msgpack:"src"`
			Dst  string `msgpack:"dst"`
			Data []byte `msgpack:"data"`
		}

		innerData := data
		if err := msgpack.Unmarshal(data, &frame); err == nil && len(frame.Data) > 0 {
			innerData = frame.Data
		}

		env, err := protocol.DecodeEnvelope(innerData)
		if err != nil {
			continue
		}

		if env.Type == protocol.TypeToolUseResult {
			result, err := protocol.DecodeBody[protocol.ToolUseResult](env)
			if err != nil {
				slog.Error("bridge decode tool result error", "error", err)
				continue
			}

			b.mu.Lock()
			ch, ok := b.pending[result.RequestID]
			b.mu.Unlock()

			if ok {
				select {
				case ch <- result:
				default:
				}
			}
		}
	}
}

func (b *Bridge) drainPending() {
	b.mu.Lock()
	for id, ch := range b.pending {
		select {
		case ch <- &protocol.ToolUseResult{Success: false, Error: "connection lost"}:
		default:
		}
		delete(b.pending, id)
	}
	b.mu.Unlock()
}

func (b *Bridge) reconnect() {
	delay := ReconnectDelayMin
	for {
		if b.closed.Load() {
			b.closeDone()
			return
		}
		slog.Info("bridge reconnecting", "delay", delay)
		time.Sleep(delay)
		if b.closed.Load() {
			b.closeDone()
			return
		}
		if err := b.Connect(context.Background()); err != nil {
			delay = min(delay*2, ReconnectDelayMax)
			continue
		}
		return
	}
}

func (b *Bridge) SendToolRequest(ctx context.Context, toolName string, args map[string]any) (string, error) {
	requestID := uuid.New().String()

	ch := make(chan *protocol.ToolUseResult, 1)
	b.mu.Lock()
	b.pending[requestID] = ch
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.pending, requestID)
		b.mu.Unlock()
	}()

	req := protocol.ToolUseRequest{
		ID:        requestID,
		ToolName:  toolName,
		Arguments: args,
		Execution: "client",
	}
	env := protocol.NewEnvelope(AssistantClientID, protocol.TypeToolUseRequest, req)
	data, err := env.Encode()
	if err != nil {
		return "", fmt.Errorf("encode tool request: %w", err)
	}

	b.connMu.Lock()
	err = b.conn.WriteMessage(websocket.BinaryMessage, data)
	b.connMu.Unlock()
	if err != nil {
		return "", fmt.Errorf("send tool request: %w", err)
	}

	slog.Info("bridge sent tool request", "tool", toolName, "request_id", requestID)

	select {
	case result := <-ch:
		if !result.Success {
			return "", fmt.Errorf("tool error: %s", result.Error)
		}
		// Serialize result to JSON string for MCP response
		resultJSON, err := json.Marshal(result.Result)
		if err != nil {
			return fmt.Sprintf("%v", result.Result), nil
		}
		return string(resultJSON), nil
	case <-time.After(ToolTimeout):
		return "", fmt.Errorf("tool request timed out after %v (assistant device may be offline)", ToolTimeout)
	case <-ctx.Done():
		return "", ctx.Err()
	case <-b.done:
		return "", fmt.Errorf("bridge connection closed")
	}
}

func (b *Bridge) Close() {
	b.closed.Store(true)
	if b.conn != nil {
		b.conn.Close()
	}
}
