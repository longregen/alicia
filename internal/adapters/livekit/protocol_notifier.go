package livekit

import (
	"context"
	"log"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

type ProtocolNotifier struct {
	ctx             context.Context
	protocolHandler ProtocolHandlerInterface
	idGenerator     ports.IDGenerator
}

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

func (n *ProtocolNotifier) NotifyThinkingSummary(messageID, conversationID string, summary string) {
	thinkingSummary := &protocol.ThinkingSummary{
		ID:             n.idGenerator.GenerateMessageID(),
		MessageID:      messageID,
		ConversationID: conversationID,
		Content:        summary,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeThinkingSummary,
		Body:           thinkingSummary,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send ThinkingSummary: %v", err)
	}
}

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

func (n *ProtocolNotifier) NotifyGenerationComplete(messageID, conversationID string, content string) {
	log.Printf("Generation completed for message %s", messageID)

	assistantMsg := &protocol.AssistantMessage{
		ID:             messageID,
		ConversationID: conversationID,
		Content:        content,
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeAssistantMessage,
		Body:           assistantMsg,
	}

	if err := n.protocolHandler.SendEnvelope(n.ctx, envelope); err != nil {
		log.Printf("Failed to send AssistantMessage on completion: %v", err)
	}
}

func (n *ProtocolNotifier) NotifyGenerationFailed(messageID, conversationID string, err error) {
	log.Printf("Generation failed for message %s: %v", messageID, err)

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

var _ ports.GenerationNotifier = (*ProtocolNotifier)(nil)
