package livekit

import (
	"context"
	"log"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

type MessageDispatcher interface {
	DispatchMessage(ctx context.Context, envelope *protocol.Envelope) error
}

type DefaultMessageDispatcher struct {
	protocolHandler           ProtocolHandlerInterface
	handleToolUseCase         ports.HandleToolUseCase
	generateResponseUseCase   ports.GenerateResponseUseCase
	processUserMessageUseCase ports.ProcessUserMessageUseCase
	conversationRepo          ports.ConversationRepository
	messageRepo               ports.MessageRepository
	toolUseRepo               ports.ToolUseRepository
	memoryUsageRepo           ports.MemoryUsageRepository
	voteRepo                  ports.VoteRepository
	noteRepo                  ports.NoteRepository
	memoryService             ports.MemoryService
	conversationID            string
	asrService                ports.ASRService
	ttsService                ports.TTSService
	idGenerator               ports.IDGenerator
	generationManager         ResponseGenerationManager

	sendMessageUseCase          ports.SendMessageUseCase
	regenerateResponseUseCase   ports.RegenerateResponseUseCase
	continueResponseUseCase     ports.ContinueResponseUseCase
	editUserMessageUseCase      ports.EditUserMessageUseCase
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase
	synthesizeSpeechUseCase     ports.SynthesizeSpeechUseCase
}

func NewDefaultMessageDispatcher(
	protocolHandler ProtocolHandlerInterface,
	handleToolUseCase ports.HandleToolUseCase,
	generateResponseUseCase ports.GenerateResponseUseCase,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	voteRepo ports.VoteRepository,
	noteRepo ports.NoteRepository,
	memoryService ports.MemoryService,
	conversationID string,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
	generationManager ResponseGenerationManager,
	sendMessageUseCase ports.SendMessageUseCase,
	regenerateResponseUseCase ports.RegenerateResponseUseCase,
	continueResponseUseCase ports.ContinueResponseUseCase,
	editUserMessageUseCase ports.EditUserMessageUseCase,
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase,
	synthesizeSpeechUseCase ports.SynthesizeSpeechUseCase,
) *DefaultMessageDispatcher {
	return &DefaultMessageDispatcher{
		protocolHandler:             protocolHandler,
		handleToolUseCase:           handleToolUseCase,
		generateResponseUseCase:     generateResponseUseCase,
		processUserMessageUseCase:   processUserMessageUseCase,
		conversationRepo:            conversationRepo,
		messageRepo:                 messageRepo,
		toolUseRepo:                 toolUseRepo,
		memoryUsageRepo:             memoryUsageRepo,
		voteRepo:                    voteRepo,
		noteRepo:                    noteRepo,
		memoryService:               memoryService,
		conversationID:              conversationID,
		asrService:                  asrService,
		ttsService:                  ttsService,
		idGenerator:                 idGenerator,
		generationManager:           generationManager,
		sendMessageUseCase:          sendMessageUseCase,
		regenerateResponseUseCase:   regenerateResponseUseCase,
		continueResponseUseCase:     continueResponseUseCase,
		editUserMessageUseCase:      editUserMessageUseCase,
		editAssistantMessageUseCase: editAssistantMessageUseCase,
		synthesizeSpeechUseCase:     synthesizeSpeechUseCase,
	}
}

func (d *DefaultMessageDispatcher) DispatchMessage(ctx context.Context, envelope *protocol.Envelope) error {
	switch envelope.Type {
	case protocol.TypeErrorMessage:
		return d.handleErrorMessage(ctx, envelope)

	case protocol.TypeUserMessage:
		return d.handleUserMessage(ctx, envelope)

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

	case protocol.TypeFeedback:
		return d.handleFeedback(ctx, envelope)

	case protocol.TypeUserNote:
		return d.handleUserNote(ctx, envelope)

	case protocol.TypeMemoryAction:
		return d.handleMemoryAction(ctx, envelope)

	case protocol.TypeResponseGenerationRequest:
		return d.handleResponseGenerationRequest(ctx, envelope)

	default:
		log.Printf("Unhandled message type: %v", envelope.Type)
		return nil
	}
}

func (d *DefaultMessageDispatcher) sendError(ctx context.Context, code int32, message string, recoverable bool) error {
	return d.protocolHandler.SendError(ctx, code, message, recoverable)
}
