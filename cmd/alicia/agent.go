package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/adapters/embedding"
	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/livekit"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/application/tools/builtin"
	"github.com/longregen/alicia/internal/application/usecases"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/spf13/cobra"
)

// agentCmd starts the LiveKit agent worker
func agentCmd() *cobra.Command {
	var serveURL string

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Start the agent worker",
		Long: `Start the agent worker to handle response generation.

The agent worker can operate in two modes:

1. LiveKit mode: Listens for LiveKit room events and dispatches agents
   to handle voice conversations when users join rooms with the "conv_" prefix.

2. WebSocket mode: Connects to alicia serve via WebSocket and receives
   response generation requests for ALL conversations.

Required configuration:
  - PostgreSQL database (ALICIA_POSTGRES_URL)
  - LLM endpoint (ALICIA_LLM_URL)

For LiveKit mode (optional):
  - LiveKit (ALICIA_LIVEKIT_URL, ALICIA_LIVEKIT_API_KEY, ALICIA_LIVEKIT_API_SECRET)
  - ASR/TTS via speaches (ALICIA_ASR_URL, ALICIA_TTS_URL)

For WebSocket mode (optional):
  - Serve URL (--serve-url or ALICIA_AGENT_SERVE_URL)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Override config with flag if provided
			if serveURL != "" {
				cfg.Agent.ServeURL = serveURL
			}
			return runAgentWorker(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&serveURL, "serve-url", "", "WebSocket URL for alicia serve (e.g., ws://localhost:8080/api/v1/ws)")

	return cmd
}

// runAgentWorker initializes and starts the agent worker
func runAgentWorker(ctx context.Context) error {
	log.Println("Starting Alicia agent worker...")

	// Validate required configuration
	if cfg.Database.PostgresURL == "" {
		return fmt.Errorf("agent worker requires PostgreSQL. Set ALICIA_POSTGRES_URL")
	}

	// Check if at least one mode is configured
	livekitEnabled := cfg.IsLiveKitConfigured()
	websocketEnabled := cfg.Agent.ServeURL != ""

	if !livekitEnabled && !websocketEnabled {
		return fmt.Errorf("agent worker requires either LiveKit or WebSocket mode. Set ALICIA_LIVEKIT_* or --serve-url/ALICIA_AGENT_SERVE_URL")
	}

	if livekitEnabled {
		if !cfg.IsASRConfigured() {
			log.Println("Warning: ASR not configured - voice transcription will not work")
		}
		if !cfg.IsTTSConfigured() {
			log.Println("Warning: TTS not configured - voice synthesis will not work")
		}
		log.Printf("  LiveKit:  %s", cfg.LiveKit.URL)
	}

	if websocketEnabled {
		log.Printf("  Serve:    %s", cfg.Agent.ServeURL)
	}

	log.Printf("  LLM:      %s", cfg.LLM.URL)
	if cfg.IsASRConfigured() {
		log.Printf("  ASR:      %s", cfg.ASR.URL)
	}
	if cfg.IsTTSConfigured() {
		log.Printf("  TTS:      %s", cfg.TTS.URL)
	}
	log.Println()

	// Initialize database connection pool
	log.Println("Connecting to PostgreSQL...")
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Force UTC timezone to prevent timezone-related issues with TIMESTAMP columns
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}
	defer pool.Close()

	// Test database connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Database connection established")

	// Initialize repositories
	conversationRepo := postgres.NewConversationRepository(pool)
	messageRepo := postgres.NewMessageRepository(pool)
	sentenceRepo := postgres.NewSentenceRepository(pool)
	reasoningStepRepo := postgres.NewReasoningStepRepository(pool)
	toolUseRepo := postgres.NewToolUseRepository(pool)
	toolRepo := postgres.NewToolRepository(pool)
	audioRepo := postgres.NewAudioRepository(pool)
	memoryRepo := postgres.NewMemoryRepository(pool)
	memoryUsageRepo := postgres.NewMemoryUsageRepository(pool)
	commentaryRepo := postgres.NewCommentaryRepository(pool)
	mcpRepo := postgres.NewMCPServerRepository(pool)
	voteRepo := postgres.NewVoteRepository(pool)
	noteRepo := postgres.NewNoteRepository(pool)
	optimizationRepo := postgres.NewOptimizationRepository(pool)

	// Initialize ID generator
	idGen := id.New()

	promptVersionRepo := postgres.NewSystemPromptVersionRepository(pool, idGen)

	// Initialize transaction manager
	txManager := postgres.NewTransactionManager(pool)
	log.Println("Transaction manager initialized")

	// Initialize LiveKit service
	lkConfig := &livekit.ServiceConfig{
		URL:                   cfg.LiveKit.URL,
		APIKey:                cfg.LiveKit.APIKey,
		APISecret:             cfg.LiveKit.APISecret,
		TokenValidityDuration: 6 * time.Hour,
	}
	lkService, err := livekit.NewService(lkConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize LiveKit service: %w", err)
	}
	log.Println("LiveKit service initialized")

	// Initialize ASR service (optional)
	var asrService ports.ASRService
	if cfg.IsASRConfigured() {
		asrService = speech.NewASRAdapterWithModel(cfg.ASR.URL, cfg.ASR.Model)
		log.Println("ASR service initialized")
	}

	// Initialize TTS service (optional)
	var ttsService ports.TTSService
	if cfg.IsTTSConfigured() {
		ttsService = speech.NewTTSAdapterWithModel(cfg.TTS.URL, cfg.TTS.Model, cfg.TTS.Voice)
		log.Println("TTS service initialized")
	}

	// Initialize LLM service
	llmService := llm.NewService(llmClient)
	log.Println("LLM service initialized")

	// Initialize embedding service (optional)
	var embeddingService ports.EmbeddingService
	if cfg.Embedding.URL != "" && cfg.Embedding.Model != "" {
		embeddingService = embedding.NewClient(
			cfg.Embedding.URL,
			cfg.Embedding.APIKey,
			cfg.Embedding.Model,
			cfg.Embedding.Dimensions,
		)
		log.Println("Embedding service initialized")
	}

	// Initialize tool service
	toolService := services.NewToolService(toolRepo, toolUseRepo, messageRepo, idGen)
	log.Println("Tool service initialized")

	// Initialize memory service (optional - requires embedding service)
	var memoryService ports.MemoryService
	if embeddingService != nil {
		memoryService = services.NewMemoryService(
			memoryRepo,
			memoryUsageRepo,
			embeddingService,
			idGen,
			txManager,
		)
		log.Println("Memory service initialized")
	}

	// Initialize optimization service
	optimizationConfig := services.DefaultOptimizationConfig()
	optimizationService := services.NewOptimizationService(
		optimizationRepo,
		llmService,
		idGen,
		optimizationConfig,
	)
	log.Println("Optimization service initialized")

	// Initialize prompt version service
	// TODO: Wire up promptVersionService to use cases that need it
	_ = services.NewPromptVersionService(
		promptVersionRepo,
		idGen,
	)
	log.Println("Prompt version service initialized")

	// Register built-in tools
	if err := builtin.RegisterAllBuiltinTools(ctx, toolService, memoryRepo, embeddingService); err != nil {
		log.Printf("Warning: Failed to register built-in tools: %v", err)
		log.Println("Continuing without built-in tools")
	} else {
		log.Println("Built-in tools registered (calculator, web_search, memory_query)")
	}

	// Initialize MCP adapter (optional)
	mcpAdapter := initMCPAdapter(ctx, toolService, mcpRepo, idGen)

	// Create a tool executor adapter
	toolExecutor := &toolExecutorAdapter{toolService: toolService}

	// Initialize use cases
	handleToolUseCase := usecases.NewHandleToolCall(
		toolRepo,
		toolUseRepo,
		toolExecutor,
		idGen,
	)
	log.Println("HandleToolCall use case initialized")

	// Initialize ParetoResponseGenerator - the unified way to generate responses
	paretoResponseGenerator := usecases.NewParetoResponseGenerator(
		llmService,
		llmService, // Use same LLM for reflection (could use a stronger model here)
		messageRepo,
		conversationRepo,
		toolRepo,
		sentenceRepo,
		toolUseRepo,
		reasoningStepRepo,
		memoryUsageRepo,
		toolService,
		memoryService,
		idGen,
		txManager,
		nil, // No broadcaster needed for CLI agent
		nil, // Use default config
	)
	log.Println("ParetoResponseGenerator initialized")

	// Create adapter for backwards compatibility with GenerateResponseUseCase interface
	generateResponseUseCase := usecases.NewParetoGenerateResponseAdapter(paretoResponseGenerator)
	log.Println("GenerateResponse adapter initialized (using Pareto search)")

	processUserMessageUseCase := usecases.NewProcessUserMessage(
		messageRepo,
		audioRepo,
		conversationRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)
	log.Println("ProcessUserMessage use case initialized")

	// Create SendMessage use case
	sendMessageUseCase := usecases.NewSendMessage(
		conversationRepo,
		messageRepo,
		processUserMessageUseCase,
		generateResponseUseCase,
		txManager,
	)
	log.Println("SendMessage use case initialized")

	// Create RegenerateResponse use case (using Pareto search)
	regenerateResponseUseCase := usecases.NewParetoRegenerateResponse(
		messageRepo,
		conversationRepo,
		paretoResponseGenerator,
		idGen,
	)
	log.Println("RegenerateResponse use case initialized (using Pareto search)")

	// Create ContinueResponse use case (using Pareto search)
	continueResponseUseCase := usecases.NewParetoContinueResponse(
		messageRepo,
		conversationRepo,
		paretoResponseGenerator,
		idGen,
		txManager,
	)
	log.Println("ContinueResponse use case initialized (using Pareto search)")

	// Create EditUserMessage use case
	editUserMessageUseCase := usecases.NewEditUserMessage(
		messageRepo,
		conversationRepo,
		memoryService,
		generateResponseUseCase,
		idGen,
		txManager,
	)
	log.Println("EditUserMessage use case initialized")

	// Create EditAssistantMessage use case
	editAssistantMessageUseCase := usecases.NewEditAssistantMessage(messageRepo)
	log.Println("EditAssistantMessage use case initialized")

	// Create SynthesizeSpeech use case
	synthesizeSpeechUseCase := usecases.NewSynthesizeSpeech(
		audioRepo,
		sentenceRepo,
		ttsService,
		idGen,
	)
	log.Println("SynthesizeSpeech use case initialized")

	// Create agent factory
	agentFactory := livekit.NewAgentFactory(
		conversationRepo,
		messageRepo,
		sentenceRepo,
		reasoningStepRepo,
		toolUseRepo,
		memoryUsageRepo,
		commentaryRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		asrService,
		ttsService,
		idGen,
		cfg.ASR.MinConfidence,
		sendMessageUseCase,
		regenerateResponseUseCase,
		continueResponseUseCase,
		editUserMessageUseCase,
		editAssistantMessageUseCase,
		synthesizeSpeechUseCase,
	)
	log.Println("Agent factory initialized")

	// Set up graceful shutdown
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	var wg sync.WaitGroup
	errorChan := make(chan error, 2)

	// Initialize LiveKit worker if configured
	var worker *livekit.Worker
	if livekitEnabled {
		workerConfig := &livekit.WorkerConfig{
			URL:                   cfg.LiveKit.URL,
			APIKey:                cfg.LiveKit.APIKey,
			APISecret:             cfg.LiveKit.APISecret,
			AgentName:             "alicia-worker",
			TokenValidityDuration: 24 * time.Hour,
			RoomPrefix:            "conv_",
			WorkerCount:           cfg.LiveKit.WorkerCount,
			WorkQueueSize:         cfg.LiveKit.WorkQueueSize,
			TTSSampleRate:         cfg.TTS.SampleRate,
			TTSChannels:           cfg.TTS.Channels,
		}

		var err error
		worker, err = livekit.NewWorker(workerConfig, agentFactory, lkService)
		if err != nil {
			return fmt.Errorf("failed to create LiveKit worker: %w", err)
		}

		// Start LiveKit worker in a goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("LiveKit agent worker started")
			if err := worker.Start(workerCtx); err != nil {
				errorChan <- fmt.Errorf("LiveKit worker error: %w", err)
			}
		}()
	}

	// Initialize WebSocket client if configured
	var wsClient *livekit.WSClient
	if websocketEnabled {
		wsClientConfig := &livekit.WSClientConfig{
			URL:               cfg.Agent.ServeURL,
			ReconnectInterval: 5 * time.Second,
			PingInterval:      30 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      10 * time.Second,
		}

		// Create WebSocket callbacks handler
		wsCallbacks := &wsAgentCallbacks{
			generateResponseUseCase:   generateResponseUseCase,
			regenerateResponseUseCase: regenerateResponseUseCase,
			continueResponseUseCase:   continueResponseUseCase,
			editUserMessageUseCase:    editUserMessageUseCase,
			conversationRepo:          conversationRepo,
			messageRepo:               messageRepo,
			idGen:                     idGen,
		}

		wsClient = livekit.NewWSClient(wsClientConfig, wsCallbacks)
		wsCallbacks.wsClient = wsClient // Set client reference for WSNotifier

		// Connect to serve
		if err := wsClient.Connect(workerCtx); err != nil {
			log.Printf("Warning: Failed to connect to serve WebSocket: %v", err)
			log.Println("WebSocket mode will retry connection...")
		} else {
			log.Println("WebSocket agent connected to serve")
		}
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal or worker error
	select {
	case err := <-errorChan:
		log.Printf("Worker error: %v", err)
		workerCancel()

		if wsClient != nil {
			wsClient.Disconnect()
		}
		if worker != nil {
			worker.Stop()
		}
		closeMCPAdapter(mcpAdapter)

		wg.Wait()
		return err

	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		log.Println("Shutting down gracefully...")

		workerCancel()

		if wsClient != nil {
			wsClient.Disconnect()
			log.Println("WebSocket client disconnected")
		}

		if worker != nil {
			if err := worker.Stop(); err != nil {
				log.Printf("LiveKit worker shutdown error: %v", err)
			}
			log.Println("LiveKit worker stopped")
		}

		closeMCPAdapter(mcpAdapter)
		wg.Wait()
		time.Sleep(1 * time.Second)
		log.Println("Agent worker stopped")
		return nil
	}
}

// wsAgentCallbacks implements livekit.WSClientCallbacks for handling
// ResponseGenerationRequest events from serve via WebSocket
type wsAgentCallbacks struct {
	generateResponseUseCase   ports.GenerateResponseUseCase
	regenerateResponseUseCase ports.RegenerateResponseUseCase
	continueResponseUseCase   ports.ContinueResponseUseCase
	editUserMessageUseCase    ports.EditUserMessageUseCase
	conversationRepo          ports.ConversationRepository
	messageRepo               ports.MessageRepository
	idGen                     ports.IDGenerator
	wsClient                  *livekit.WSClient
}

// OnResponseGenerationRequest handles incoming response generation requests
func (c *wsAgentCallbacks) OnResponseGenerationRequest(ctx context.Context, req *protocol.ResponseGenerationRequest) error {
	log.Printf("========================================")
	log.Printf("[Agent] RECEIVED ResponseGenerationRequest")
	log.Printf("[Agent]   RequestID:      %s", req.ID)
	log.Printf("[Agent]   RequestType:    %s", req.RequestType)
	log.Printf("[Agent]   ConversationID: %s", req.ConversationID)
	log.Printf("[Agent]   MessageID:      %s", req.MessageID)
	log.Printf("[Agent]   EnableTools:    %v", req.EnableTools)
	log.Printf("[Agent]   EnableReasoning:%v", req.EnableReasoning)
	log.Printf("[Agent]   EnableStreaming:%v", req.EnableStreaming)
	log.Printf("========================================")

	// Create a WSNotifier to send generation events back to serve
	notifier := livekit.NewWSNotifier(c.wsClient)

	switch req.RequestType {
	case "send":
		// Generate response for a new user message
		log.Printf("[Agent] Executing GenerateResponse use case...")
		input := &ports.GenerateResponseInput{
			ConversationID:  req.ConversationID,
			UserMessageID:   req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
			Notifier:        notifier,
		}

		output, err := c.generateResponseUseCase.Execute(ctx, input)
		if err != nil {
			log.Printf("[Agent] GenerateResponse FAILED: %v", err)
			notifier.NotifyGenerationFailed(req.MessageID, req.ConversationID, err)
			return err
		}

		// Note: NotifyGenerationComplete is called internally by ParetoResponseGenerator
		if output.Message != nil {
			log.Printf("[Agent] GenerateResponse COMPLETE: messageID=%s, contentLen=%d", output.Message.ID, len(output.Message.Contents))
		}

	case "regenerate":
		// Regenerate response for an existing assistant message
		// Note: The regenerate use case calls GenerateResponse internally
		log.Printf("[Agent] Executing RegenerateResponse use case...")
		input := &ports.RegenerateResponseInput{
			MessageID:       req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
			Notifier:        notifier,
		}

		output, err := c.regenerateResponseUseCase.Execute(ctx, input)
		if err != nil {
			log.Printf("[Agent] RegenerateResponse FAILED: %v", err)
			notifier.NotifyGenerationFailed(req.MessageID, req.ConversationID, err)
			return err
		}

		// Note: NotifyGenerationComplete is called internally by ParetoResponseGenerator
		if output.NewMessage != nil {
			log.Printf("[Agent] RegenerateResponse COMPLETE: newMessageID=%s, contentLen=%d", output.NewMessage.ID, len(output.NewMessage.Contents))
		}

	case "continue":
		// Continue from an existing assistant message
		// Note: The continue use case calls GenerateResponse internally
		log.Printf("[Agent] Executing ContinueResponse use case...")
		input := &ports.ContinueResponseInput{
			TargetMessageID: req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
			Notifier:        notifier,
		}

		output, err := c.continueResponseUseCase.Execute(ctx, input)
		if err != nil {
			log.Printf("[Agent] ContinueResponse FAILED: %v", err)
			notifier.NotifyGenerationFailed(req.MessageID, req.ConversationID, err)
			return err
		}

		// Note: NotifyGenerationComplete is called internally by ParetoResponseGenerator
		if output.TargetMessage != nil {
			log.Printf("[Agent] ContinueResponse COMPLETE: targetMessageID=%s, appendedLen=%d", output.TargetMessage.ID, len(output.AppendedContent))
		}

	case "edit":
		// Generate response after editing a user message
		log.Printf("[Agent] Executing GenerateResponse (for edit) use case...")
		input := &ports.GenerateResponseInput{
			ConversationID:  req.ConversationID,
			UserMessageID:   req.MessageID,
			EnableTools:     req.EnableTools,
			EnableReasoning: req.EnableReasoning,
			EnableStreaming: req.EnableStreaming,
			Notifier:        notifier,
		}

		output, err := c.generateResponseUseCase.Execute(ctx, input)
		if err != nil {
			log.Printf("[Agent] GenerateResponse (for edit) FAILED: %v", err)
			notifier.NotifyGenerationFailed(req.MessageID, req.ConversationID, err)
			return err
		}

		// Note: NotifyGenerationComplete is called internally by ParetoResponseGenerator
		if output.Message != nil {
			log.Printf("[Agent] GenerateResponse (for edit) COMPLETE: messageID=%s, contentLen=%d", output.Message.ID, len(output.Message.Contents))
		}

	default:
		log.Printf("[Agent] Unknown request type: %s", req.RequestType)
	}

	log.Printf("[Agent] Request processing finished for %s", req.RequestType)
	return nil
}

// OnConnected is called when the WebSocket connection is established
func (c *wsAgentCallbacks) OnConnected() {
	log.Println("wsAgentCallbacks: Connected to serve")
}

// OnDisconnected is called when the WebSocket connection is lost
func (c *wsAgentCallbacks) OnDisconnected(err error) {
	if err != nil {
		log.Printf("wsAgentCallbacks: Disconnected from serve: %v", err)
	} else {
		log.Println("wsAgentCallbacks: Disconnected from serve")
	}
}
