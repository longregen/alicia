package livekit

import (
	"fmt"

	"github.com/longregen/alicia/internal/ports"
)

// AgentFactory creates and wires up LiveKit agents with all dependencies
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
	optimizationService       ports.OptimizationService
	handleToolUseCase         ports.HandleToolUseCase
	generateResponseUseCase   ports.GenerateResponseUseCase
	processUserMessageUseCase ports.ProcessUserMessageUseCase
	asrService                ports.ASRService
	ttsService                ports.TTSService
	idGenerator               ports.IDGenerator
}

// NewAgentFactory creates a new agent factory
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
	optimizationService ports.OptimizationService,
	handleToolUseCase ports.HandleToolUseCase,
	generateResponseUseCase ports.GenerateResponseUseCase,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
) *AgentFactory {
	return &AgentFactory{
		conversationRepo:          conversationRepo,
		messageRepo:               messageRepo,
		sentenceRepo:              sentenceRepo,
		reasoningStepRepo:         reasoningStepRepo,
		toolUseRepo:               toolUseRepo,
		memoryUsageRepo:           memoryUsageRepo,
		commentaryRepo:            commentaryRepo,
		voteRepo:                  voteRepo,
		noteRepo:                  noteRepo,
		memoryService:             memoryService,
		optimizationService:       optimizationService,
		handleToolUseCase:         handleToolUseCase,
		generateResponseUseCase:   generateResponseUseCase,
		processUserMessageUseCase: processUserMessageUseCase,
		asrService:                asrService,
		ttsService:                ttsService,
		idGenerator:               idGenerator,
	}
}

// CreateAgent creates a fully-wired agent for a conversation
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
		f.optimizationService,
		conversationID,
		f.asrService,
		f.ttsService,
		f.idGenerator,
		agent,
	)

	// Wire up the agent with the message router as callbacks
	agent.SetCallbacks(messageRouter)

	return agent, messageRouter, nil
}

// CreateAgentWithCallbacks creates an agent with custom callbacks wrapper
// This allows you to wrap the message router with additional logic if needed
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
