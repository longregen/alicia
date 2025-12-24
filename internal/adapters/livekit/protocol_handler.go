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
	// MaxBufferSize is the maximum number of messages to keep in memory for reconnection
	MaxBufferSize = 200

	// MaxReconnectionGap is the maximum stanza gap we'll replay from memory/database
	// If the gap is larger, we return an error asking the client to start a new conversation
	MaxReconnectionGap = 1000
)

// AgentSender is an interface for sending data and audio to clients
type AgentSender interface {
	SendData(ctx context.Context, data []byte) error
	SendAudio(ctx context.Context, audio []byte, format string) error
}

// ProtocolHandler handles protocol messages with reconnection semantics
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
	stanzaMu           sync.Mutex // Protects lastServerStanzaID
	lastServerStanzaID int32
}

// NewProtocolHandler creates a new protocol handler
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
		lastServerStanzaID: -1, // Start at -1, will increment to -2, -3, etc.
	}
}

// HandleConfiguration processes a Configuration message and handles reconnection
func (h *ProtocolHandler) HandleConfiguration(ctx context.Context, config *protocol.Configuration) error {
	lastSequenceSeen := config.LastSequenceSeen

	// Load conversation from database
	conversation, err := h.conversationRepo.GetByID(ctx, h.conversationID)
	if err != nil {
		return h.sendError(ctx, protocol.ErrCodeConversationNotFound, "Conversation not found", true)
	}

	// If this is a reconnection (lastSequenceSeen > 0), replay missed messages
	if lastSequenceSeen != 0 {
		return h.handleReconnection(ctx, conversation, lastSequenceSeen)
	}

	// First connection - send acknowledgement
	return h.sendAcknowledgement(ctx, 0, true)
}

// handleReconnection handles the reconnection flow
func (h *ProtocolHandler) handleReconnection(ctx context.Context, conversation *models.Conversation, lastSequenceSeen int32) error {
	// Determine which messages need to be replayed
	// lastSequenceSeen could be from client (positive) or server (negative)
	// We need to check both directions

	var missedServerMessages []*protocol.Envelope

	// For server messages (negative stanzaIDs), check if we need to replay
	// Server messages are stored in absolute value in SequenceNumber
	if lastSequenceSeen < 0 {
		// Client is reporting the last server message they saw
		lastSeenAbs := absInt32(lastSequenceSeen)
		currentServerAbs := absInt32(conversation.LastServerStanzaID)

		gap := currentServerAbs - lastSeenAbs

		if gap > MaxReconnectionGap {
			return h.sendError(ctx, protocol.ErrCodeInvalidState,
				fmt.Sprintf("Reconnection gap too large (%d messages). Please start a new conversation.", gap),
				false)
		}

		// Try to get messages from buffer first
		missedServerMessages = h.messageBuffer.GetMessagesSince(lastSequenceSeen)

		// If not all in buffer, query database for the rest
		if len(missedServerMessages) < int(gap) {
			dbMessages, err := h.messageRepo.GetAfterSequence(ctx, h.conversationID, int(lastSeenAbs))
			if err != nil {
				return fmt.Errorf("failed to query missed messages: %w", err)
			}

			// Convert database messages to envelopes
			// For each message, we need to reconstruct all related protocol messages
			for _, msg := range dbMessages {
				// First, add the message itself (UserMessage or AssistantMessage)
				envelope := h.messageToEnvelope(msg)
				missedServerMessages = append(missedServerMessages, envelope)

				// For assistant messages, fetch and add all related entities
				if msg.IsFromAssistant() {
					// Fetch and add sentences (AssistantSentence messages)
					sentences, err := h.sentenceRepo.GetByMessage(ctx, msg.ID)
					if err == nil && len(sentences) > 0 {
						// Add StartAnswer marker before sentences
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

						// Add sentences
						previousID := msg.ID
						for _, sentence := range sentences {
							sentEnv := h.sentenceToEnvelope(sentence, previousID)
							missedServerMessages = append(missedServerMessages, sentEnv)
							previousID = sentence.ID
						}
					}

					// Fetch and add reasoning steps
					reasoningSteps, err := h.reasoningStepRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, step := range reasoningSteps {
							stepEnv := h.reasoningStepToEnvelope(step)
							missedServerMessages = append(missedServerMessages, stepEnv)
						}
					}

					// Fetch and add tool uses (both requests and results)
					toolUses, err := h.toolUseRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, toolUse := range toolUses {
							// Add ToolUseRequest for pending/running status
							if toolUse.IsPending() || toolUse.IsRunning() {
								reqEnv := h.toolUseToRequestEnvelope(toolUse)
								missedServerMessages = append(missedServerMessages, reqEnv)
							}

							// Add ToolUseResult for completed status
							if toolUse.IsComplete() {
								// First add the request
								reqEnv := h.toolUseToRequestEnvelope(toolUse)
								missedServerMessages = append(missedServerMessages, reqEnv)

								// Then add the result
								resEnv := h.toolUseToResultEnvelope(toolUse)
								missedServerMessages = append(missedServerMessages, resEnv)
							}
						}
					}

					// Fetch and add memory usages
					memoryUsages, err := h.memoryUsageRepo.GetByMessage(ctx, msg.ID)
					if err == nil {
						for _, usage := range memoryUsages {
							memEnv := h.memoryUsageToEnvelope(usage)
							missedServerMessages = append(missedServerMessages, memEnv)
						}
					}

					// Fetch and add commentary
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

	// Send acknowledgement first
	if err := h.sendAcknowledgement(ctx, lastSequenceSeen, true); err != nil {
		return err
	}

	// Replay missed messages
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

// SendEnvelope sends an envelope and buffers it for potential replay
func (h *ProtocolHandler) SendEnvelope(ctx context.Context, envelope *protocol.Envelope) error {
	// Set the stanzaID if not already set and capture current value for persistence
	var stanzaIDToPersist int32
	if envelope.StanzaID == 0 {
		h.stanzaMu.Lock()
		h.lastServerStanzaID -= 1 // Decrement to get next negative ID
		envelope.StanzaID = h.lastServerStanzaID
		stanzaIDToPersist = h.lastServerStanzaID
		h.stanzaMu.Unlock()
	} else {
		// If stanzaID was already set, still need to read current value for persistence
		h.stanzaMu.Lock()
		stanzaIDToPersist = h.lastServerStanzaID
		h.stanzaMu.Unlock()
	}

	// Buffer the message for potential replay
	h.messageBuffer.Add(envelope)

	// Persist the server stanza ID (best-effort, don't fail the send on persistence errors)
	if err := h.conversationRepo.UpdateStanzaIDs(ctx, h.conversationID, 0, stanzaIDToPersist); err != nil {
		// Log but don't fail - stanza tracking is best-effort
		log.Printf("WARNING: Failed to persist server stanza ID for conversation %s: %v", h.conversationID, err)
	}

	// Send the message
	data, err := msgpack.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	return h.agent.SendData(ctx, data)
}

// sendAcknowledgement sends an Acknowledgement message
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

// sendError sends an ErrorMessage
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

// newEnvelope creates a new envelope with standard fields
func newEnvelope(stanzaID int32, conversationID string, msgType protocol.MessageType, body interface{}) *protocol.Envelope {
	return &protocol.Envelope{
		StanzaID:       stanzaID,
		ConversationID: conversationID,
		Type:           msgType,
		Body:           body,
	}
}

// messageToEnvelope converts a database message to a protocol envelope
// This is a simplified version - in practice you'd need to handle different message types
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
		// Default to assistant message
		msgType = protocol.TypeAssistantMessage
		body = &protocol.AssistantMessage{
			ID:             msg.ID,
			PreviousID:     msg.PreviousID,
			ConversationID: msg.ConversationID,
			Content:        msg.Contents,
		}
	}

	// Calculate stanzaID from sequence number
	// For server messages, use negative values
	stanzaID := -int32(msg.SequenceNumber)

	return newEnvelope(stanzaID, msg.ConversationID, msgType, body)
}

// sentenceToEnvelope converts a Sentence model to an AssistantSentence envelope
func (h *ProtocolHandler) sentenceToEnvelope(sentence *models.Sentence, previousID string) *protocol.Envelope {
	body := &protocol.AssistantSentence{
		ID:             sentence.ID,
		PreviousID:     previousID,
		ConversationID: h.conversationID,
		Sequence:       int32(sentence.SequenceNumber),
		Text:           sentence.Text,
		IsFinal:        false, // This would need to be tracked in the model
		Audio:          sentence.AudioData,
	}
	return newEnvelope(-int32(sentence.SequenceNumber), h.conversationID, protocol.TypeAssistantSentence, body)
}

// reasoningStepToEnvelope converts a ReasoningStep model to a ReasoningStep envelope
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

// toolUseToRequestEnvelope converts a ToolUse model to a ToolUseRequest envelope
func (h *ProtocolHandler) toolUseToRequestEnvelope(toolUse *models.ToolUse) *protocol.Envelope {
	body := &protocol.ToolUseRequest{
		ID:             toolUse.ID,
		MessageID:      toolUse.MessageID, // Include the message ID that triggered this tool call
		ConversationID: h.conversationID,
		ToolName:       toolUse.ToolName,
		Parameters:     toolUse.Arguments,
		Execution:      protocol.ToolExecutionServer, // Default to server execution
		TimeoutMs:      protocol.DefaultToolTimeout,
	}
	return newEnvelope(-int32(toolUse.SequenceNumber), h.conversationID, protocol.TypeToolUseRequest, body)
}

// toolUseToResultEnvelope converts a ToolUse model to a ToolUseResult envelope
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

// memoryUsageToEnvelope converts a MemoryUsage model to a MemoryTrace envelope
func (h *ProtocolHandler) memoryUsageToEnvelope(usage *models.MemoryUsage) *protocol.Envelope {
	content := ""
	relevance := usage.SimilarityScore

	// If the memory is loaded, use its content
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

// commentaryToEnvelope converts a Commentary model to a Commentary envelope
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

// UpdateClientStanzaID updates the client stanza ID in the conversation
func (h *ProtocolHandler) UpdateClientStanzaID(ctx context.Context, stanzaID int32) {
	// Only update if the stanza ID is positive (from client)
	if stanzaID > 0 {
		// Best-effort update, don't fail if persistence fails
		if err := h.conversationRepo.UpdateStanzaIDs(ctx, h.conversationID, stanzaID, 0); err != nil {
			// Log but don't fail - stanza tracking is best-effort
			log.Printf("WARNING: Failed to persist client stanza ID for conversation %s: %v", h.conversationID, err)
		}
	}
}

// absInt32 returns the absolute value of an int32
func absInt32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// SendToolUseRequest sends a ToolUseRequest message
func (h *ProtocolHandler) SendToolUseRequest(ctx context.Context, toolUse *models.ToolUse) error {
	envelope := h.toolUseToRequestEnvelope(toolUse)
	// Clear the stanzaID so SendEnvelope will assign it
	envelope.StanzaID = 0
	return h.SendEnvelope(ctx, envelope)
}

// SendToolUseResult sends a ToolUseResult message
func (h *ProtocolHandler) SendToolUseResult(ctx context.Context, toolUse *models.ToolUse) error {
	envelope := h.toolUseToResultEnvelope(toolUse)
	// Clear the stanzaID so SendEnvelope will assign it
	envelope.StanzaID = 0
	return h.SendEnvelope(ctx, envelope)
}

// SendAudio sends audio data to the client via the agent
func (h *ProtocolHandler) SendAudio(ctx context.Context, audio []byte, format string) error {
	if h.agent == nil {
		return fmt.Errorf("agent not available")
	}

	return h.agent.SendAudio(ctx, audio, format)
}
