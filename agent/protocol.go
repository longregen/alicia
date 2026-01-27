package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

// WSNotifier sends protocol messages over WebSocket.
type WSNotifier struct {
	conn           *websocket.Conn
	conversationID string
	messageID      string
	previousID     string
	mu             sync.Mutex
}

func NewWSNotifier(conn *websocket.Conn) *WSNotifier {
	return &WSNotifier{conn: conn}
}

func (n *WSNotifier) SetConversationID(id string) {
	n.mu.Lock()
	n.conversationID = id
	n.mu.Unlock()
}

func (n *WSNotifier) SetMessageID(id string) {
	n.mu.Lock()
	n.messageID = id
	n.mu.Unlock()
}

func (n *WSNotifier) SetPreviousID(id string) {
	n.mu.Lock()
	n.previousID = id
	n.mu.Unlock()
}

func (n *WSNotifier) send(ctx context.Context, msgType protocol.MessageType, body any) {
	n.mu.Lock()
	defer n.mu.Unlock()

	env := protocol.Envelope{
		ConversationID: n.conversationID,
		Type:           msgType,
		Body:           body,
	}

	if ctx != nil {
		tc := otel.InjectToTraceContext(ctx, n.conversationID, otel.UserIDFromContext(ctx))
		env.TraceID = tc.TraceID
		env.SpanID = tc.SpanID
		env.TraceFlags = tc.TraceFlags
		env.SessionID = tc.SessionID
		env.UserID = tc.UserID
	}

	data, err := msgpack.Marshal(env)
	if err != nil {
		slog.Error("msgpack marshal error", "error", err)
		return
	}
	if err := n.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		slog.Error("websocket write error", "error", err)
	}
}

func (n *WSNotifier) SendStartAnswer(ctx context.Context, messageID string) {
	n.mu.Lock()
	convID := n.conversationID
	prevID := n.previousID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeStartAnswer, protocol.StartAnswer{
		MessageID:      messageID,
		ConversationID: convID,
		PreviousID:     prevID,
	})
}

func (n *WSNotifier) SendThinking(ctx context.Context, messageID, text string) {
	n.SendThinkingWithProgress(ctx, messageID, text, 0)
}

func (n *WSNotifier) SendThinkingWithProgress(ctx context.Context, messageID, text string, progress float32) {
	n.mu.Lock()
	convID := n.conversationID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeThinkingSummary, protocol.ThinkingSummary{
		ID:             NewThinkingID(),
		MessageID:      messageID,
		ConversationID: convID,
		Content:        text,
		Progress:       progress,
		Timestamp:      time.Now().UnixMilli(),
	})
}

func (n *WSNotifier) SendToolStart(ctx context.Context, id, name string, args map[string]any) {
	n.mu.Lock()
	convID := n.conversationID
	msgID := n.messageID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeToolUseRequest, protocol.ToolUseRequest{
		ID:             id,
		MessageID:      msgID,
		ConversationID: convID,
		ToolName:       name,
		Arguments:      args,
		Execution:      "server",
	})
}

func (n *WSNotifier) SendToolComplete(ctx context.Context, id string, success bool, result any, errMsg string) {
	n.mu.Lock()
	convID := n.conversationID
	msgID := n.messageID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeToolUseResult, protocol.ToolUseResult{
		ID:             NewToolUseID(),
		RequestID:      id,
		MessageID:      msgID,
		ConversationID: convID,
		Success:        success,
		Result:         result,
		Error:          errMsg,
	})
}

func (n *WSNotifier) SendComplete(ctx context.Context, messageID, content string) {
	n.mu.Lock()
	convID := n.conversationID
	prevID := n.previousID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeAssistantMsg, protocol.AssistantMessage{
		ID:             messageID,
		PreviousID:     prevID,
		ConversationID: convID,
		Content:        content,
		Timestamp:      time.Now().UnixMilli(),
	})
}

func (n *WSNotifier) SendError(ctx context.Context, messageID string, err error) {
	n.mu.Lock()
	convID := n.conversationID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeError, protocol.Error{
		Code:           "agent_error",
		Message:        err.Error(),
		MessageID:      messageID,
		ConversationID: convID,
	})
}

func (n *WSNotifier) SendTitleUpdate(ctx context.Context, title string) {
	n.mu.Lock()
	convID := n.conversationID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeTitleUpdate, protocol.TitleUpdate{
		ConversationID: convID,
		Title:          title,
	})
}

func (n *WSNotifier) SendMemoryTrace(ctx context.Context, messageID, memoryID, content string, relevance float32) {
	n.mu.Lock()
	convID := n.conversationID
	n.mu.Unlock()
	n.send(ctx, protocol.TypeMemoryTrace, protocol.MemoryTrace{
		ID:             NewMemoryTraceID(),
		MemoryID:       memoryID,
		MessageID:      messageID,
		ConversationID: convID,
		Content:        content,
		Relevance:      relevance,
	})
}
