package livekit

import (
	"log"
	"time"

	"github.com/longregen/alicia/pkg/protocol"
)

// WSNotifier implements ports.GenerationNotifier by sending events through a WebSocket client
type WSNotifier struct {
	client *WSClient
}

// NewWSNotifier creates a new WebSocket-based notifier
func NewWSNotifier(client *WSClient) *WSNotifier {
	return &WSNotifier{
		client: client,
	}
}

// NotifyGenerationStarted sends a StartAnswer message
func (n *WSNotifier) NotifyGenerationStarted(messageID, previousID, conversationID string) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	start := &protocol.StartAnswer{
		ID:             messageID,
		PreviousID:     previousID,
		ConversationID: conversationID,
		AnswerType:     protocol.AnswerTypeText,
	}

	if err := n.client.SendStartAnswer(conversationID, start); err != nil {
		log.Printf("WSNotifier: Failed to send StartAnswer: %v", err)
	}
}

// NotifyMemoryRetrieved sends a MemoryTrace message
func (n *WSNotifier) NotifyMemoryRetrieved(messageID, conversationID string, memoryID string, content string, relevance float32) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	trace := &protocol.MemoryTrace{
		ID:             memoryID,
		MessageID:      messageID,
		ConversationID: conversationID,
		MemoryID:       memoryID,
		Content:        content,
		Relevance:      relevance,
	}

	if err := n.client.SendMemoryTrace(conversationID, trace); err != nil {
		log.Printf("WSNotifier: Failed to send MemoryTrace: %v", err)
	}
}

// NotifyReasoningStep sends a ReasoningStep message
func (n *WSNotifier) NotifyReasoningStep(id, messageID, conversationID string, sequence int, content string) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	step := &protocol.ReasoningStep{
		ID:             id,
		MessageID:      messageID,
		ConversationID: conversationID,
		Sequence:       int32(sequence),
		Content:        content,
	}

	if err := n.client.SendReasoningStep(conversationID, step); err != nil {
		log.Printf("WSNotifier: Failed to send ReasoningStep: %v", err)
	}
}

// NotifyToolUseStart sends a ToolUseRequest message
func (n *WSNotifier) NotifyToolUseStart(id, messageID, conversationID string, toolName string, arguments map[string]any) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	// Convert map[string]any to map[string]interface{}
	args := make(map[string]interface{})
	for k, v := range arguments {
		args[k] = v
	}

	req := &protocol.ToolUseRequest{
		ID:             id,
		MessageID:      messageID,
		ConversationID: conversationID,
		ToolName:       toolName,
		Parameters:     args,
		Execution:      protocol.ToolExecutionServer,
	}

	if err := n.client.SendToolUseRequest(conversationID, req); err != nil {
		log.Printf("WSNotifier: Failed to send ToolUseRequest: %v", err)
	}
}

// NotifyToolUseComplete sends a ToolUseResult message
func (n *WSNotifier) NotifyToolUseComplete(id, requestID, conversationID string, success bool, result any, errorMsg string) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	res := &protocol.ToolUseResult{
		ID:             id,
		RequestID:      requestID,
		ConversationID: conversationID,
		Success:        success,
		Result:         result,
	}

	if !success && errorMsg != "" {
		res.ErrorMessage = errorMsg
	}

	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeToolUseResult, res)
	if err := n.client.SendEnvelope(envelope); err != nil {
		log.Printf("WSNotifier: Failed to send ToolUseResult: %v", err)
	}
}

// NotifySentence sends an AssistantSentence message
func (n *WSNotifier) NotifySentence(id, previousID, conversationID string, sequence int, text string, isFinal bool) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	sentence := &protocol.AssistantSentence{
		ID:             id,
		PreviousID:     previousID,
		ConversationID: conversationID,
		Sequence:       int32(sequence),
		Text:           text,
		IsFinal:        isFinal,
	}

	if err := n.client.SendAssistantSentence(conversationID, sentence); err != nil {
		log.Printf("WSNotifier: Failed to send AssistantSentence: %v", err)
	}
}

// NotifyGenerationComplete sends the final AssistantMessage
func (n *WSNotifier) NotifyGenerationComplete(messageID, conversationID string, content string) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	msg := &protocol.AssistantMessage{
		ID:             messageID,
		ConversationID: conversationID,
		Content:        content,
		Timestamp:      time.Now().UnixMilli(),
	}

	if err := n.client.SendAssistantMessage(conversationID, msg); err != nil {
		log.Printf("WSNotifier: Failed to send AssistantMessage: %v", err)
	}
}

// NotifyGenerationFailed sends an error message
func (n *WSNotifier) NotifyGenerationFailed(messageID, conversationID string, err error) {
	if n.client == nil || !n.client.IsConnected() {
		return
	}

	errorMsg := &protocol.ErrorMessage{
		ID:             messageID,
		ConversationID: conversationID,
		Code:           protocol.ErrCodeInternalError,
		Message:        err.Error(),
		Severity:       protocol.SeverityError,
		Recoverable:    false,
		OriginatingID:  messageID,
	}

	envelope := protocol.NewEnvelope(0, conversationID, protocol.TypeErrorMessage, errorMsg)
	if err := n.client.SendEnvelope(envelope); err != nil {
		log.Printf("WSNotifier: Failed to send ErrorMessage: %v", err)
	}
}
