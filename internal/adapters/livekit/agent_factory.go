package livekit

import (
	"fmt"

	"github.com/longregen/alicia/internal/ports"
)

type AgentFactory struct {
	conversationRepo          ports.ConversationRepository
	messageRepo               ports.MessageRepository
	sentenceRepo              ports.SentenceRepository
	reasoningStepRepo         ports.ReasoningStepRepository
	toolUseRepo               ports.ToolUseRepository
	memoryUsageRepo           ports.MemoryUsageRepository
	commentaryRepo            ports.CommentaryRepository
	voteRepo                  ports.VoteRepository
	noteRepo                  ports.NoteRepository
	memoryService             ports.MemoryService
	handleToolUseCase         ports.HandleToolUseCase
	generateResponseUseCase   ports.GenerateResponseUseCase
	processUserMessageUseCase ports.ProcessUserMessageUseCase
	asrService                ports.ASRService
	ttsService                ports.TTSService
	idGenerator               ports.IDGenerator
	asrMinConfidence          float64

	// New use cases for message operations
	sendMessageUseCase          ports.SendMessageUseCase
	regenerateResponseUseCase   ports.RegenerateResponseUseCase
	continueResponseUseCase     ports.ContinueResponseUseCase
	editUserMessageUseCase      ports.EditUserMessageUseCase
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase
	synthesizeSpeechUseCase     ports.SynthesizeSpeechUseCase
}

func NewAgentFactory(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	sentenceRepo ports.SentenceRepository,
	reasoningStepRepo ports.ReasoningStepRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	commentaryRepo ports.CommentaryRepository,
	voteRepo ports.VoteRepository,
	noteRepo ports.NoteRepository,
	memoryService ports.MemoryService,
	handleToolUseCase ports.HandleToolUseCase,
	generateResponseUseCase ports.GenerateResponseUseCase,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
	asrMinConfidence float64,
	sendMessageUseCase ports.SendMessageUseCase,
	regenerateResponseUseCase ports.RegenerateResponseUseCase,
	continueResponseUseCase ports.ContinueResponseUseCase,
	editUserMessageUseCase ports.EditUserMessageUseCase,
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase,
	synthesizeSpeechUseCase ports.SynthesizeSpeechUseCase,
) *AgentFactory {
	return &AgentFactory{
		conversationRepo:            conversationRepo,
		messageRepo:                 messageRepo,
		sentenceRepo:                sentenceRepo,
		reasoningStepRepo:           reasoningStepRepo,
		toolUseRepo:                 toolUseRepo,
		memoryUsageRepo:             memoryUsageRepo,
		commentaryRepo:              commentaryRepo,
		voteRepo:                    voteRepo,
		noteRepo:                    noteRepo,
		memoryService:               memoryService,
		handleToolUseCase:           handleToolUseCase,
		generateResponseUseCase:     generateResponseUseCase,
		processUserMessageUseCase:   processUserMessageUseCase,
		asrService:                  asrService,
		ttsService:                  ttsService,
		idGenerator:                 idGenerator,
		asrMinConfidence:            asrMinConfidence,
		sendMessageUseCase:          sendMessageUseCase,
		regenerateResponseUseCase:   regenerateResponseUseCase,
		continueResponseUseCase:     continueResponseUseCase,
		editUserMessageUseCase:      editUserMessageUseCase,
		editAssistantMessageUseCase: editAssistantMessageUseCase,
		synthesizeSpeechUseCase:     synthesizeSpeechUseCase,
	}
}

func (f *AgentFactory) CreateAgent(
	config *AgentConfig,
	conversationID string,
) (*Agent, *MessageRouter, error) {
	// Create the agent first (without callbacks)
	agent, err := NewAgent(config, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Create codec
	codec := NewCodec()

	// Create protocol handler
	protocolHandler := NewProtocolHandler(
		agent,
		f.conversationRepo,
		f.messageRepo,
		f.sentenceRepo,
		f.reasoningStepRepo,
		f.toolUseRepo,
		f.memoryUsageRepo,
		f.commentaryRepo,
		conversationID,
	)

	// Create message router with all dependencies
	// Pass the agent so the router can set up the voice pipeline
	messageRouter := NewMessageRouter(
		codec,
		protocolHandler,
		f.handleToolUseCase,
		f.generateResponseUseCase,
		f.processUserMessageUseCase,
		f.conversationRepo,
		f.messageRepo,
		f.toolUseRepo,
		f.memoryUsageRepo,
		f.voteRepo,
		f.noteRepo,
		f.memoryService,
		conversationID,
		f.asrService,
		f.ttsService,
		f.idGenerator,
		agent,
		f.asrMinConfidence,
		f.sendMessageUseCase,
		f.regenerateResponseUseCase,
		f.continueResponseUseCase,
		f.editUserMessageUseCase,
		f.editAssistantMessageUseCase,
		f.synthesizeSpeechUseCase,
	)

	// Wire up the agent with the message router as callbacks
	agent.SetCallbacks(messageRouter)

	return agent, messageRouter, nil
}

func (f *AgentFactory) CreateAgentWithCallbacks(
	config *AgentConfig,
	conversationID string,
	callbacksWrapper func(*MessageRouter) ports.LiveKitAgentCallbacks,
) (*Agent, *MessageRouter, error) {
	agent, messageRouter, err := f.CreateAgent(config, conversationID)
	if err != nil {
		return nil, nil, err
	}

	// Apply custom wrapper if provided
	if callbacksWrapper != nil {
		agent.SetCallbacks(callbacksWrapper(messageRouter))
	}

	return agent, messageRouter, nil
}
