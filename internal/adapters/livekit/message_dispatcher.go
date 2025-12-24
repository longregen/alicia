package livekit

import (
	"context"
	"log"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// MessageDispatcher routes protocol messages to appropriate handlers
type MessageDispatcher interface {
	DispatchMessage(ctx context.Context, envelope *protocol.Envelope) error
}

// DefaultMessageDispatcher implements MessageDispatcher
type DefaultMessageDispatcher struct {
	protocolHandler           *ProtocolHandler
	handleToolUseCase         ports.HandleToolUseCase
	generateResponseUseCase   ports.GenerateResponseUseCase
	processUserMessageUseCase ports.ProcessUserMessageUseCase
	messageRepo               ports.MessageRepository
	conversationID            string
	asrService                ports.ASRService
	ttsService                ports.TTSService
	idGenerator               ports.IDGenerator
	generationManager         ResponseGenerationManager
}

// NewDefaultMessageDispatcher creates a new message dispatcher
func NewDefaultMessageDispatcher(
	protocolHandler *ProtocolHandler,
	handleToolUseCase ports.HandleToolUseCase,
	generateResponseUseCase ports.GenerateResponseUseCase,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	messageRepo ports.MessageRepository,
	conversationID string,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
	generationManager ResponseGenerationManager,
) *DefaultMessageDispatcher {
	return &DefaultMessageDispatcher{
		protocolHandler:           protocolHandler,
		handleToolUseCase:         handleToolUseCase,
		generateResponseUseCase:   generateResponseUseCase,
		processUserMessageUseCase: processUserMessageUseCase,
		messageRepo:               messageRepo,
		conversationID:            conversationID,
		asrService:                asrService,
		ttsService:                ttsService,
		idGenerator:               idGenerator,
		generationManager:         generationManager,
	}
}

// DispatchMessage routes a protocol message to the appropriate handler
func (d *DefaultMessageDispatcher) DispatchMessage(ctx context.Context, envelope *protocol.Envelope) error {
	// Route based on message type
	switch envelope.Type {
	case protocol.TypeErrorMessage:
		return d.handleErrorMessage(ctx, envelope)

	case protocol.TypeUserMessage:
		return d.handleUserMessage(ctx, envelope)

	// TypeAssistantMessage (3) - Server→Client only, sent by server in non-streaming mode
	// TypeReasoningStep (5) - Server→Client only, sent by server during response generation
	// TypeStartAnswer (13) - Server→Client only, sent by server to initiate streaming response
	// TypeMemoryTrace (14) - Server→Client only, sent by server when memories are retrieved
	// TypeAssistantSentence (16) - Server→Client only, sent by server for streaming chunks

	case protocol.TypeAudioChunk:
		return d.handleAudioChunk(ctx, envelope)

	case protocol.TypeToolUseRequest:
		return d.handleToolUseRequest(ctx, envelope)

	case protocol.TypeToolUseResult:
		return d.handleToolUseResult(ctx, envelope)

	case protocol.TypeAcknowledgement:
		return d.handleAcknowledgement(ctx, envelope)

	case protocol.TypeTranscription:
		return d.handleTranscription(ctx, envelope)

	case protocol.TypeControlStop:
		return d.handleControlStop(ctx, envelope)

	case protocol.TypeControlVariation:
		return d.handleControlVariation(ctx, envelope)

	case protocol.TypeConfiguration:
		return d.handleConfiguration(ctx, envelope)

	case protocol.TypeCommentary:
		return d.handleCommentary(ctx, envelope)

	default:
		log.Printf("Unhandled message type: %v", envelope.Type)
		return nil // Silently ignore unknown message types
	}
}

// sendError sends an error message via the protocol handler
func (d *DefaultMessageDispatcher) sendError(ctx context.Context, code int32, message string, recoverable bool) error {
	return d.protocolHandler.sendError(ctx, code, message, recoverable)
}
