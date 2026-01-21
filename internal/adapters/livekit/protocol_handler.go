package livekit

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	MaxBufferSize = 200

	// If the reconnection gap exceeds this, we return an error asking the client to start a new conversation
	MaxReconnectionGap = 1000
)

type AgentSender interface {
	SendData(ctx context.Context, data []byte) error
	SendAudio(ctx context.Context, audio []byte, format string) error
}

type ProtocolHandlerInterface interface {
	HandleConfiguration(ctx context.Context, config *protocol.Configuration) error
	SendEnvelope(ctx context.Context, envelope *protocol.Envelope) error
	SendAudio(ctx context.Context, audio []byte, format string) error
	SendToolUseRequest(ctx context.Context, toolUse *models.ToolUse) error
	SendToolUseResult(ctx context.Context, toolUse *models.ToolUse) error
	SendAcknowledgement(ctx context.Context, ackedStanzaID int32, success bool) error
	SendError(ctx context.Context, code int32, message string, recoverable bool) error
	GetToolUseRepo() ports.ToolUseRepository
}

type ProtocolHandler struct {
	agent              AgentSender
	messageBuffer      *MessageBuffer
	conversationRepo   ports.ConversationRepository
	messageRepo        ports.MessageRepository
	sentenceRepo       ports.SentenceRepository
	reasoningStepRepo  ports.ReasoningStepRepository
	toolUseRepo        ports.ToolUseRepository
	memoryUsageRepo    ports.MemoryUsageRepository
	commentaryRepo     ports.CommentaryRepository
	conversationID     string
	stanzaMu           sync.Mutex
	lastServerStanzaID int32
}

func NewProtocolHandler(
	agent AgentSender,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	sentenceRepo ports.SentenceRepository,
	reasoningStepRepo ports.ReasoningStepRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	commentaryRepo ports.CommentaryRepository,
	conversationID string,
) *ProtocolHandler {
	return &ProtocolHandler{
		agent:              agent,
		messageBuffer:      NewMessageBuffer(MaxBufferSize),
		conversationRepo:   conversationRepo,
		messageRepo:        messageRepo,
		sentenceRepo:       sentenceRepo,
		reasoningStepRepo:  reasoningStepRepo,
		toolUseRepo:        toolUseRepo,
		memoryUsageRepo:    memoryUsageRepo,
		commentaryRepo:     commentaryRepo,
		conversationID:     conversationID,
		lastServerStanzaID: -1,
	}
}

func (h *ProtocolHandler) HandleConfiguration(ctx context.Context, config *protocol.Configuration) error {
	lastSequenceSeen := config.LastSequenceSeen

	conversation, err := h.conversationRepo.GetByID(ctx, h.conversationID)
	if err != nil {
		return h.sendError(ctx, protocol.ErrCodeConversationNotFound, "Conversation not found", true)
	}

	if lastSequenceSeen != 0 {
		return h.handleReconnection(ctx, conversation, lastSequenceSeen)
	}

	return h.sendAcknowledgement(ctx, 0, true)
}

func (h *ProtocolHandler) handleReconnection(ctx context.Context, conversation *models.Conversation, lastSequenceSeen int32) error {
	var missedServerMessages []*protocol.Envelope

	if lastSequenceSeen < 0 {
		lastSeenAbs := absInt32(lastSequenceSeen)
		currentServerAbs := absInt32(conversation.LastServerStanzaID)

		gap := currentServerAbs - lastSeenAbs

		if gap > MaxReconnectionGap {
			return h.sendError(ctx, protocol.ErrCodeInvalidState,
				fmt.Sprintf("Reconnection gap too large (%d messages). Please start a new conversation.", gap),
				false)
		}

		missedServerMessages = h.messageBuffer.GetMessagesSince(lastSequenceSeen)

		if len(missedServerMessages) < int(gap) {
			dbMessages, err := h.messageRepo.GetAfterSequence(ctx, h.conversationID, int(lastSeenAbs))
			if err != nil {
				return fmt.Errorf("failed to query missed messages: %w", err)
			}

			for _, msg := range dbMessages {
				envelope := h.messageToEnvelope(msg)
				missedServerMessages = append(missedServerMessages, envelope)

				if msg.IsFromAssistant() {
					sentences, err := h.sentenceRepo.GetByMessage(ctx, msg.ID)
					if err == nil && len(sentences) > 0 {
						startAnswer := &protocol.Envelope{
							StanzaID:       -int32(msg.SequenceNumber),
							ConversationID: h.conversationID,
							Type:           protocol.TypeStartAnswer,
							Body: &protocol.StartAnswer{
								ID:                   msg.ID,
								PreviousID:           msg.PreviousID,
								ConversationID:       msg.ConversationID,
								AnswerType:           protocol.AnswerTypeText,
								PlannedSentenceCount: int32(len(sentences)),
							},
						}
						missedServerMessages = append(missedServerMessages, startAnswer)

						previousID := msg.ID
						for _, sentence := range sentences {
							sentEnv := h.sentenceToEnvelope(sentence, previousID)
							missedServerMessages = append(missedServerMessages, sentEnv)
							previousID = sentence.ID
						}
					}

					reasoningSteps, err := h.reasoningStepRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, step := range reasoningSteps {
							stepEnv := h.reasoningStepToEnvelope(step)
							missedServerMessages = append(missedServerMessages, stepEnv)
						}
					}

					toolUses, err := h.toolUseRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, toolUse := range toolUses {
							if toolUse.IsPending() || toolUse.IsRunning() {
								reqEnv := h.toolUseToRequestEnvelope(toolUse)
								missedServerMessages = append(missedServerMessages, reqEnv)
							}

							if toolUse.IsComplete() {
								reqEnv := h.toolUseToRequestEnvelope(toolUse)
								missedServerMessages = append(missedServerMessages, reqEnv)

								resEnv := h.toolUseToResultEnvelope(toolUse)
								missedServerMessages = append(missedServerMessages, resEnv)
							}
						}
					}

					memoryUsages, err := h.memoryUsageRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, usage := range memoryUsages {
							memEnv := h.memoryUsageToEnvelope(usage)
							missedServerMessages = append(missedServerMessages, memEnv)
						}
					}

					commentaries, err := h.commentaryRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, commentary := range commentaries {
							commEnv := h.commentaryToEnvelope(commentary)
							missedServerMessages = append(missedServerMessages, commEnv)
						}
					}
				}
			}
		}
	}

	if err := h.sendAcknowledgement(ctx, lastSequenceSeen, true); err != nil {
		return err
	}

	for _, envelope := range missedServerMessages {
		data, err := msgpack.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("failed to marshal envelope: %w", err)
		}

		if err := h.agent.SendData(ctx, data); err != nil {
			return fmt.Errorf("failed to send missed message: %w", err)
		}
	}

	return nil
}

func (h *ProtocolHandler) SendEnvelope(ctx context.Context, envelope *protocol.Envelope) error {
	var stanzaIDToPersist int32
	if envelope.StanzaID == 0 {
		h.stanzaMu.Lock()
		h.lastServerStanzaID -= 1
		envelope.StanzaID = h.lastServerStanzaID
		stanzaIDToPersist = h.lastServerStanzaID
		h.stanzaMu.Unlock()
	} else {
		h.stanzaMu.Lock()
		stanzaIDToPersist = h.lastServerStanzaID
		h.stanzaMu.Unlock()
	}

	h.messageBuffer.Add(envelope)

	// Best-effort persistence - don't fail the send on persistence errors
	if err := h.conversationRepo.UpdateStanzaIDs(ctx, h.conversationID, 0, stanzaIDToPersist); err != nil {
		log.Printf("WARNING: Failed to persist server stanza ID for conversation %s: %v", h.conversationID, err)
	}

	data, err := msgpack.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	return h.agent.SendData(ctx, data)
}

func (h *ProtocolHandler) GetToolUseRepo() ports.ToolUseRepository {
	return h.toolUseRepo
}

func (h *ProtocolHandler) SendAcknowledgement(ctx context.Context, ackedStanzaID int32, success bool) error {
	return h.sendAcknowledgement(ctx, ackedStanzaID, success)
}

func (h *ProtocolHandler) SendError(ctx context.Context, code int32, message string, recoverable bool) error {
	return h.sendError(ctx, code, message, recoverable)
}

func (h *ProtocolHandler) sendAcknowledgement(ctx context.Context, ackedStanzaID int32, success bool) error {
	envelope := &protocol.Envelope{
		ConversationID: h.conversationID,
		Type:           protocol.TypeAcknowledgement,
		Body: &protocol.Acknowledgement{
			ConversationID: h.conversationID,
			AckedStanzaID:  ackedStanzaID,
			Success:        success,
		},
	}

	return h.SendEnvelope(ctx, envelope)
}

func (h *ProtocolHandler) sendError(ctx context.Context, code int32, message string, recoverable bool) error {
	envelope := &protocol.Envelope{
		ConversationID: h.conversationID,
		Type:           protocol.TypeErrorMessage,
		Body: &protocol.ErrorMessage{
			ConversationID: h.conversationID,
			Code:           code,
			Message:        message,
			Severity:       protocol.SeverityError,
			Recoverable:    recoverable,
		},
	}

	return h.SendEnvelope(ctx, envelope)
}

func newEnvelope(stanzaID int32, conversationID string, msgType protocol.MessageType, body interface{}) *protocol.Envelope {
	return &protocol.Envelope{
		StanzaID:       stanzaID,
		ConversationID: conversationID,
		Type:           msgType,
		Body:           body,
	}
}

func (h *ProtocolHandler) messageToEnvelope(msg *models.Message) *protocol.Envelope {
	var msgType protocol.MessageType
	var body interface{}

	switch msg.Role {
	case models.MessageRoleUser:
		msgType = protocol.TypeUserMessage
		body = &protocol.UserMessage{
			ID:             msg.ID,
			PreviousID:     msg.PreviousID,
			ConversationID: msg.ConversationID,
			Content:        msg.Contents,
		}
	case models.MessageRoleAssistant:
		msgType = protocol.TypeAssistantMessage
		body = &protocol.AssistantMessage{
			ID:             msg.ID,
			PreviousID:     msg.PreviousID,
			ConversationID: msg.ConversationID,
			Content:        msg.Contents,
		}
	default:
		msgType = protocol.TypeAssistantMessage
		body = &protocol.AssistantMessage{
			ID:             msg.ID,
			PreviousID:     msg.PreviousID,
			ConversationID: msg.ConversationID,
			Content:        msg.Contents,
		}
	}

	// Server messages use negative stanzaIDs derived from sequence number
	stanzaID := -int32(msg.SequenceNumber)

	return newEnvelope(stanzaID, msg.ConversationID, msgType, body)
}

func (h *ProtocolHandler) sentenceToEnvelope(sentence *models.Sentence, previousID string) *protocol.Envelope {
	body := &protocol.AssistantSentence{
		ID:             sentence.ID,
		PreviousID:     previousID,
		ConversationID: h.conversationID,
		Sequence:       int32(sentence.SequenceNumber),
		Text:           sentence.Text,
		IsFinal:        false,
		Audio:          sentence.AudioData,
	}
	return newEnvelope(-int32(sentence.SequenceNumber), h.conversationID, protocol.TypeAssistantSentence, body)
}

func (h *ProtocolHandler) reasoningStepToEnvelope(step *models.ReasoningStep) *protocol.Envelope {
	body := &protocol.ReasoningStep{
		ID:             step.ID,
		MessageID:      step.MessageID,
		ConversationID: h.conversationID,
		Sequence:       int32(step.SequenceNumber),
		Content:        step.Content,
	}
	return newEnvelope(-int32(step.SequenceNumber), h.conversationID, protocol.TypeReasoningStep, body)
}

func (h *ProtocolHandler) toolUseToRequestEnvelope(toolUse *models.ToolUse) *protocol.Envelope {
	body := &protocol.ToolUseRequest{
		ID:             toolUse.ID,
		MessageID:      toolUse.MessageID,
		ConversationID: h.conversationID,
		ToolName:       toolUse.ToolName,
		Parameters:     toolUse.Arguments,
		Execution:      protocol.ToolExecutionServer,
		TimeoutMs:      protocol.DefaultToolTimeout,
	}
	return newEnvelope(-int32(toolUse.SequenceNumber), h.conversationID, protocol.TypeToolUseRequest, body)
}

func (h *ProtocolHandler) toolUseToResultEnvelope(toolUse *models.ToolUse) *protocol.Envelope {
	success := toolUse.Status == models.ToolStatusSuccess
	var errorCode, errorMessage string
	if toolUse.Status == models.ToolStatusError {
		errorCode = "TOOL_ERROR"
		errorMessage = toolUse.ErrorMessage
	} else if toolUse.Status == models.ToolStatusCancelled {
		errorCode = "TOOL_CANCELLED"
		errorMessage = "Tool execution was cancelled"
	}

	body := &protocol.ToolUseResult{
		ID:             toolUse.ID + "_result",
		RequestID:      toolUse.ID,
		ConversationID: h.conversationID,
		Success:        success,
		Result:         toolUse.Result,
		ErrorCode:      errorCode,
		ErrorMessage:   errorMessage,
	}
	return newEnvelope(-int32(toolUse.SequenceNumber), h.conversationID, protocol.TypeToolUseResult, body)
}

func (h *ProtocolHandler) memoryUsageToEnvelope(usage *models.MemoryUsage) *protocol.Envelope {
	content := ""
	relevance := usage.SimilarityScore

	if usage.Memory != nil {
		content = usage.Memory.Content
	}

	body := &protocol.MemoryTrace{
		ID:             usage.ID,
		MessageID:      usage.MessageID,
		ConversationID: usage.ConversationID,
		MemoryID:       usage.MemoryID,
		Content:        content,
		Relevance:      relevance,
	}
	return newEnvelope(-int32(usage.PositionInResults), h.conversationID, protocol.TypeMemoryTrace, body)
}

func (h *ProtocolHandler) commentaryToEnvelope(commentary *models.Commentary) *protocol.Envelope {
	commentaryType := ""
	if commentary.Meta != nil {
		if ct, ok := commentary.Meta["commentary_type"].(string); ok {
			commentaryType = ct
		}
	}

	body := &protocol.Commentary{
		ID:             commentary.ID,
		MessageID:      commentary.MessageID,
		ConversationID: commentary.ConversationID,
		Content:        commentary.Content,
		CommentaryType: commentaryType,
	}
	return newEnvelope(-1, h.conversationID, protocol.TypeCommentary, body)
}

func (h *ProtocolHandler) UpdateClientStanzaID(ctx context.Context, stanzaID int32) {
	// Only update for positive stanzaIDs (from client)
	if stanzaID > 0 {
		if err := h.conversationRepo.UpdateStanzaIDs(ctx, h.conversationID, stanzaID, 0); err != nil {
			log.Printf("WARNING: Failed to persist client stanza ID for conversation %s: %v", h.conversationID, err)
		}
	}
}

func absInt32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func (h *ProtocolHandler) SendToolUseRequest(ctx context.Context, toolUse *models.ToolUse) error {
	envelope := h.toolUseToRequestEnvelope(toolUse)
	envelope.StanzaID = 0
	return h.SendEnvelope(ctx, envelope)
}

func (h *ProtocolHandler) SendToolUseResult(ctx context.Context, toolUse *models.ToolUse) error {
	envelope := h.toolUseToResultEnvelope(toolUse)
	envelope.StanzaID = 0
	return h.SendEnvelope(ctx, envelope)
}

func (h *ProtocolHandler) SendAudio(ctx context.Context, audio []byte, format string) error {
	if h.agent == nil {
		return fmt.Errorf("agent not available")
	}

	return h.agent.SendAudio(ctx, audio, format)
}
