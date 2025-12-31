package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	"github.com/spf13/cobra"
)

// agentCmd starts the LiveKit agent worker
func agentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "Start the LiveKit agent worker",
		Long: `Start the LiveKit agent worker to handle voice conversations.

The agent worker listens for LiveKit room events and dispatches agents
to handle voice conversations when users join rooms with the "conv_" prefix.

Required configuration:
  - PostgreSQL database (ALICIA_POSTGRES_URL)
  - LLM endpoint (ALICIA_LLM_URL)
  - LiveKit (ALICIA_LIVEKIT_URL, ALICIA_LIVEKIT_API_KEY, ALICIA_LIVEKIT_API_SECRET)
  - ASR/TTS via speaches (ALICIA_ASR_URL, ALICIA_TTS_URL)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentWorker(cmd.Context())
		},
	}
}

// runAgentWorker initializes and starts the LiveKit agent worker
func runAgentWorker(ctx context.Context) error {
	log.Println("Starting Alicia agent worker...")

	// Validate required configuration
	if cfg.Database.PostgresURL == "" {
		return fmt.Errorf("agent worker requires PostgreSQL. Set ALICIA_POSTGRES_URL")
	}

	if !cfg.IsLiveKitConfigured() {
		return fmt.Errorf("agent worker requires LiveKit. Set ALICIA_LIVEKIT_URL, ALICIA_LIVEKIT_API_KEY, ALICIA_LIVEKIT_API_SECRET")
	}

	if !cfg.IsASRConfigured() {
		log.Println("Warning: ASR not configured - voice transcription will not work")
	}

	if !cfg.IsTTSConfigured() {
		log.Println("Warning: TTS not configured - voice synthesis will not work")
	}

	log.Printf("  LiveKit:  %s", cfg.LiveKit.URL)
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
	promptVersionService := services.NewPromptVersionService(
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

	generateResponseUseCase := usecases.NewGenerateResponse(
		messageRepo,
		sentenceRepo,
		toolUseRepo,
		toolRepo,
		reasoningStepRepo,
		conversationRepo,
		llmService,
		toolService,
		memoryService,
		promptVersionService,
		idGen,
		txManager,
	)
	log.Println("GenerateResponse use case initialized")

	processUserMessageUseCase := usecases.NewProcessUserMessage(
		messageRepo,
		audioRepo,
		asrService,
		memoryService,
		idGen,
		txManager,
	)
	log.Println("ProcessUserMessage use case initialized")

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
	)
	log.Println("Agent factory initialized")

	// Create worker configuration
	workerConfig := &livekit.WorkerConfig{
		URL:                   cfg.LiveKit.URL,
		APIKey:                cfg.LiveKit.APIKey,
		APISecret:             cfg.LiveKit.APISecret,
		AgentName:             "alicia-worker",
		TokenValidityDuration: 24 * time.Hour,
		RoomPrefix:            "conv_",
		WorkerCount:           cfg.LiveKit.WorkerCount,
		WorkQueueSize:         cfg.LiveKit.WorkQueueSize,
	}

	// Create and start worker
	worker, err := livekit.NewWorker(workerConfig, agentFactory, lkService)
	if err != nil {
		return fmt.Errorf("failed to create worker: %w", err)
	}

	// Set up graceful shutdown
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	// Channel for worker errors
	workerErrors := make(chan error, 1)

	// Start worker in a goroutine
	go func() {
		log.Println("Agent worker started")
		workerErrors <- worker.Start(workerCtx)
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal or worker error
	select {
	case err := <-workerErrors:
		closeMCPAdapter(mcpAdapter)
		if err != nil {
			return fmt.Errorf("worker error: %w", err)
		}
		return nil
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		log.Println("Shutting down gracefully...")

		if err := worker.Stop(); err != nil {
			closeMCPAdapter(mcpAdapter)
			return fmt.Errorf("worker shutdown error: %w", err)
		}

		closeMCPAdapter(mcpAdapter)
		time.Sleep(2 * time.Second)
		log.Println("Worker stopped")
		return nil
	}
}
