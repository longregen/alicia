package livekit

import (
	"context"
	"log"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// ProtocolNotifier implements ports.GenerationNotifier by sending protocol messages
// via a ProtocolHandlerInterface. This allows real-time streaming of generation
// progress to connected clients.
type ProtocolNotifier struct {
	ctx             context.Context
	protocolHandler ProtocolHandlerInterface
	idGenerator     ports.IDGenerator
}

// NewProtocolNotifier creates a new ProtocolNotifier
func NewProtocolNotifier(
	ctx context.Context,
	protocolHandler ProtocolHandlerInterface,
	idGenerator ports.IDGenerator,
) *ProtocolNotifier {
	return &ProtocolNotifier{
		ctx:             ctx,
		protocolHandler: protocolHandler,
		idGenerator:     idGenerator,
	}
}

// NotifyGenerationStarted sends a StartAnswer message when generation begins
func (n *ProtocolNotifier) NotifyGenerationStarted(messageID, previousID, conversationID string) {
	startAnswer := &protocol.StartAnswer{
		ID:             messageID,
		PreviousID:     previousID,
		ConversationID: conversationID,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeStartAnswer,
		Body:           startAnswer,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send StartAnswer: %v", err)
	}
}

// NotifyMemoryRetrieved sends a MemoryTrace message for each retrieved memory
func (n *ProtocolNotifier) NotifyMemoryRetrieved(messageID, conversationID string, memoryID string, content string, relevance float32) {
	memoryTrace := &protocol.MemoryTrace{
		ID:             n.idGenerator.GenerateMessageID(),
		MessageID:      messageID,
		ConversationID: conversationID,
		MemoryID:       memoryID,
		Content:        content,
		Relevance:      relevance,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeMemoryTrace,
		Body:           memoryTrace,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send MemoryTrace for memory %s: %v", memoryID, err)
	}
}

// NotifyReasoningStep sends a ReasoningStep message
func (n *ProtocolNotifier) NotifyReasoningStep(id, messageID, conversationID string, sequence int, content string) {
	reasoningStep := &protocol.ReasoningStep{
		ID:             id,
		MessageID:      messageID,
		ConversationID: conversationID,
		Sequence:       int32(sequence),
		Content:        content,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeReasoningStep,
		Body:           reasoningStep,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send ReasoningStep: %v", err)
	}
}

// NotifyToolUseStart sends a ToolUseRequest message before tool execution
func (n *ProtocolNotifier) NotifyToolUseStart(id, messageID, conversationID string, toolName string, arguments map[string]any) {
	toolUseRequest := &protocol.ToolUseRequest{
		ID:             id,
		MessageID:      messageID,
		ConversationID: conversationID,
		ToolName:       toolName,
		Parameters:     arguments,
		Execution:      protocol.ToolExecutionServer,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeToolUseRequest,
		Body:           toolUseRequest,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send ToolUseRequest for %s: %v", toolName, err)
	}
}

// NotifyToolUseComplete sends a ToolUseResult message after tool execution
func (n *ProtocolNotifier) NotifyToolUseComplete(id, requestID, conversationID string, success bool, result any, errorMsg string) {
	toolUseResult := &protocol.ToolUseResult{
		ID:             id,
		RequestID:      requestID,
		ConversationID: conversationID,
		Success:        success,
		Result:         result,
	}

	if !success && errorMsg != "" {
		toolUseResult.ErrorMessage = errorMsg
		toolUseResult.ErrorCode = "TOOL_EXECUTION_ERROR"
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeToolUseResult,
		Body:           toolUseResult,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send ToolUseResult for %s: %v", requestID, err)
	}
}

// NotifySentence sends an AssistantSentence message for streaming text chunks
func (n *ProtocolNotifier) NotifySentence(id, previousID, conversationID string, sequence int, text string, isFinal bool) {
	sentence := &protocol.AssistantSentence{
		ID:             id,
		PreviousID:     previousID,
		ConversationID: conversationID,
		Sequence:       int32(sequence),
		Text:           text,
		IsFinal:        isFinal,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeAssistantSentence,
		Body:           sentence,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send AssistantSentence: %v", err)
	}
}

// NotifyGenerationComplete sends a final AssistantMessage when generation completes
func (n *ProtocolNotifier) NotifyGenerationComplete(messageID, conversationID string, content string) {
	// The streaming already sends sentences, so completion is primarily for
	// signaling that the message is done. We could send the full AssistantMessage
	// here if needed for non-streaming clients, but the streaming flow handles this.
	log.Printf("Generation completed for message %s", messageID)
}

// NotifyGenerationFailed logs the failure (error handling is done via the stream)
func (n *ProtocolNotifier) NotifyGenerationFailed(messageID, conversationID string, err error) {
	log.Printf("Generation failed for message %s: %v", messageID, err)

	// Send an error message to the client
	errorMsg := &protocol.ErrorMessage{
		ID:             n.idGenerator.GenerateMessageID(),
		ConversationID: conversationID,
		Code:           protocol.ErrCodeInternalError,
		Message:        err.Error(),
		Severity:       protocol.SeverityError,
		Recoverable:    true,
		OriginatingID:  messageID,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeErrorMessage,
		Body:           errorMsg,
	}

	if sendErr := n.protocolHandler.SendEnvelope(n.ctx, envelope); sendErr != nil {
		log.Printf("Failed to send generation error: %v", sendErr)
	}
}

// Ensure ProtocolNotifier implements ports.GenerationNotifier
var _ ports.GenerationNotifier = (*ProtocolNotifier)(nil)
