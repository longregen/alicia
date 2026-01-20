package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/adapters/embedding"
	"github.com/longregen/alicia/internal/adapters/http"
	"github.com/longregen/alicia/internal/adapters/http/handlers"
	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/livekit"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/adapters/tracing"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/application/tools/builtin"
	"github.com/longregen/alicia/internal/application/usecases"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
	"github.com/spf13/cobra"
)

// serveCmd starts the HTTP API server
func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		Long: `Start the Alicia HTTP API server for real-time communication.

The server provides REST endpoints for conversation management and
integrates with LiveKit for real-time audio streaming.

Required configuration:
  - PostgreSQL database (ALICIA_POSTGRES_URL)
  - LLM endpoint (ALICIA_LLM_URL)

Optional:
  - LiveKit (ALICIA_LIVEKIT_URL, ALICIA_LIVEKIT_API_KEY, ALICIA_LIVEKIT_API_SECRET)
  - ASR/TTS via speaches (ALICIA_ASR_URL, ALICIA_TTS_URL)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context())
		},
	}
}

// maskDatabaseURL masks the password in a database URL for safe logging
func maskDatabaseURL(dbURL string) string {
	parsed, err := url.Parse(dbURL)
	if err != nil {
		return "[invalid URL]"
	}
	if parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			parsed.User = url.UserPassword(parsed.User.Username(), "****")
		}
	}
	return parsed.String()
}

// runServer initializes and starts the HTTP API server
func runServer(ctx context.Context) error {
	log.Println("Starting Alicia API server...")
	log.Printf("  HTTP:     http://%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("  Postgres: %s", maskDatabaseURL(cfg.Database.PostgresURL))
	log.Printf("  LLM:      %s", cfg.LLM.URL)

	if cfg.IsLiveKitConfigured() {
		log.Printf("  LiveKit:  %s", cfg.LiveKit.URL)
	}
	if cfg.IsASRConfigured() {
		log.Printf("  ASR:      %s", cfg.ASR.URL)
	}
	if cfg.IsTTSConfigured() {
		log.Printf("  TTS:      %s", cfg.TTS.URL)
	}
	log.Println()

	// Initialize OpenTelemetry tracing
	log.Println("Initializing OpenTelemetry tracing...")
	shutdown, err := tracing.InitTracer("alicia-api")
	if err != nil {
		log.Printf("Warning: Failed to initialize tracing: %v", err)
	} else {
		defer func() {
			if err := shutdown(ctx); err != nil {
				log.Printf("Error shutting down tracer: %v", err)
			}
		}()
		log.Println("OpenTelemetry tracing initialized")
	}

	// Validate required configuration
	if cfg.Database.PostgresURL == "" {
		return fmt.Errorf("server mode requires PostgreSQL. Set ALICIA_POSTGRES_URL")
	}

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

	// Initialize ID generator
	idGen := id.New()

	// Initialize repositories
	conversationRepo := postgres.NewConversationRepository(pool)
	messageRepo := postgres.NewMessageRepository(pool)
	sentenceRepo := postgres.NewSentenceRepository(pool)
	reasoningStepRepo := postgres.NewReasoningStepRepository(pool)
	toolRepo := postgres.NewToolRepository(pool)
	toolUseRepo := postgres.NewToolUseRepository(pool)
	audioRepo := postgres.NewAudioRepository(pool)
	memoryRepo := postgres.NewMemoryRepository(pool)
	memoryUsageRepo := postgres.NewMemoryUsageRepository(pool)
	mcpRepo := postgres.NewMCPServerRepository(pool)
	noteRepo := postgres.NewNoteRepository(pool)
	voteRepo := postgres.NewVoteRepository(pool)
	sessionStatsRepo := postgres.NewSessionStatsRepository(pool)
	optimizationRepo := postgres.NewOptimizationRepository(pool)
	trainingExampleRepo := postgres.NewTrainingExampleRepository(pool, idGen)
	promptVersionRepo := postgres.NewSystemPromptVersionRepository(pool, idGen)

	// Initialize transaction manager
	txManager := postgres.NewTransactionManager(pool)
	log.Println("Transaction manager initialized")

	// Initialize Embedding client (optional) - needed before tool service
	var embeddingClient *embedding.Client
	var embeddingService ports.EmbeddingService
	if cfg.IsEmbeddingConfigured() {
		embeddingClient = embedding.NewClient(
			cfg.Embedding.URL,
			cfg.Embedding.APIKey,
			cfg.Embedding.Model,
			cfg.Embedding.Dimensions,
		)
		embeddingService = embeddingClient
		log.Println("Embedding client initialized")
	}

	// Initialize LLM service
	llmService := llm.NewService(llmClient)
	log.Println("LLM service initialized")

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

	// Create WebSocket broadcaster (shared between server and use cases)
	wsBroadcaster := handlers.NewWebSocketBroadcaster()
	log.Println("WebSocket broadcaster initialized")

	// Initialize optimization progress publisher with WebSocket broadcaster
	progressPublisher := services.NewOptimizationProgressPublisher(wsBroadcaster)
	log.Println("Optimization progress publisher initialized")

	// Initialize optimization service with WebSocket broadcaster
	optimizationService := services.NewOptimizationService(
		optimizationRepo,
		llmService,
		idGen,
		services.DefaultOptimizationConfig(),
	).WithBroadcaster(wsBroadcaster)
	log.Println("Optimization service initialized")

	// Initialize prompt version service
	promptVersionService := services.NewPromptVersionService(
		promptVersionRepo,
		idGen,
	)
	log.Println("Prompt version service initialized")

	// Initialize training set builder service
	trainingBuilderConfig := services.DefaultTrainingSetBuilderConfig()
	trainingBuilderService := services.NewTrainingSetBuilderService(
		voteRepo,
		trainingExampleRepo,
		toolRepo,
		memoryRepo,
		idGen,
		trainingBuilderConfig,
	)
	log.Println("Training set builder service initialized")

	// Register built-in tools
	if err := builtin.RegisterAllBuiltinTools(ctx, toolService, memoryRepo, embeddingClient); err != nil {
		log.Printf("Warning: Failed to register built-in tools: %v", err)
	} else {
		log.Println("Built-in tools registered")
	}

	// Initialize native tools (web_*, garden_*)
	if err := initNativeTools(ctx, toolService, pool); err != nil {
		log.Printf("Warning: Failed to initialize native tools: %v", err)
	}

	// Initialize MCP adapter (optional - for custom MCP servers)
	mcpAdapter := initMCPAdapter(ctx, toolService, mcpRepo, idGen)

	// Initialize ASR adapter (optional)
	var asrAdapter *speech.ASRAdapter
	if cfg.IsASRConfigured() {
		asrAdapter = speech.NewASRAdapterWithModel(cfg.ASR.URL, cfg.ASR.Model)
		log.Println("ASR adapter initialized")
	}

	// Initialize TTS adapter (optional)
	var ttsAdapter *speech.TTSAdapter
	if cfg.IsTTSConfigured() {
		ttsAdapter = speech.NewTTSAdapterWithModel(cfg.TTS.URL, cfg.TTS.Model, cfg.TTS.Voice)
		log.Println("TTS adapter initialized")
	}

	// Initialize LiveKit service (optional)
	var liveKitService ports.LiveKitService
	if cfg.IsLiveKitConfigured() {
		lkConfig := &livekit.ServiceConfig{
			URL:                   cfg.LiveKit.URL,
			APIKey:                cfg.LiveKit.APIKey,
			APISecret:             cfg.LiveKit.APISecret,
			TokenValidityDuration: 6 * time.Hour,
		}
		liveKitService, err = livekit.NewService(lkConfig)
		if err != nil {
			log.Printf("Warning: Failed to initialize LiveKit service: %v", err)
			log.Println("LiveKit features will be unavailable")
		} else {
			log.Println("LiveKit service initialized")
		}
	} else {
		log.Println("LiveKit not configured - voice features unavailable")
	}

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
		wsBroadcaster,
		nil, // Use default config
	)
	log.Println("ParetoResponseGenerator initialized")

	// Create adapter for backwards compatibility with GenerateResponseUseCase interface
	generateResponseUseCase := usecases.NewParetoGenerateResponseAdapter(paretoResponseGenerator)
	log.Println("GenerateResponse adapter initialized (using Pareto search)")

	// Create ProcessUserMessage use case (needed by SendMessage)
	processUserMessageUC := usecases.NewProcessUserMessage(
		messageRepo,
		audioRepo,
		conversationRepo,
		asrAdapter,
		memoryService,
		idGen,
		txManager,
	)
	log.Println("ProcessUserMessage use case initialized")

	// Create SendMessage orchestrating use case
	sendMessageUC := usecases.NewSendMessage(
		conversationRepo,
		messageRepo,
		processUserMessageUC,
		generateResponseUseCase,
		txManager,
	)
	log.Println("SendMessage use case initialized")

	// Create SyncMessages use case
	syncMessagesUC := usecases.NewSyncMessages(
		messageRepo,
		conversationRepo,
		idGen,
		txManager,
	)
	log.Println("SyncMessages use case initialized")

	// Create RegenerateResponse use case (using Pareto search)
	regenerateResponseUC := usecases.NewParetoRegenerateResponse(
		messageRepo,
		conversationRepo,
		paretoResponseGenerator,
		idGen,
	)
	log.Println("RegenerateResponse use case initialized (using Pareto search)")

	// Create ContinueResponse use case (using Pareto search)
	continueResponseUC := usecases.NewParetoContinueResponse(
		messageRepo,
		conversationRepo,
		paretoResponseGenerator,
		idGen,
		txManager,
	)
	log.Println("ContinueResponse use case initialized (using Pareto search)")

	// Create EditUserMessage use case
	editUserMessageUC := usecases.NewEditUserMessage(
		messageRepo,
		conversationRepo,
		memoryService,
		generateResponseUseCase,
		idGen,
		txManager,
	)
	log.Println("EditUserMessage use case initialized")

	// Create EditAssistantMessage use case
	editAssistantMessageUC := usecases.NewEditAssistantMessage(messageRepo)
	log.Println("EditAssistantMessage use case initialized")

	// Create RunOptimization use case
	runOptimizationUC := usecases.NewRunOptimization(
		optimizationService,
		progressPublisher,
		wsBroadcaster,
		llmService,
		idGen,
		optimizationRepo,
	)
	log.Println("RunOptimization use case initialized")

	// Create ExtractMemories use case (needed by MemorizeFromUpvote)
	var extractMemoriesUC *usecases.ExtractMemories
	if memoryService != nil {
		extractMemoriesUC = usecases.NewExtractMemories(memoryService, llmService, idGen)
		log.Println("ExtractMemories use case initialized")
	}

	// Create MemorizeFromUpvote use case (optional - requires memory service)
	var memorizeFromUpvoteUC *usecases.MemorizeFromUpvote
	if extractMemoriesUC != nil {
		memorizeFromUpvoteUC = usecases.NewMemorizeFromUpvote(
			messageRepo,
			conversationRepo,
			voteRepo,
			extractMemoriesUC,
		)
		log.Println("MemorizeFromUpvote use case initialized")
	}

	// Create HTTP server
	server := http.NewServer(
		cfg,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		noteRepo,
		voteRepo,
		sessionStatsRepo,
		memoryService,
		optimizationService,
		optimizationRepo,
		trainingBuilderService,
		promptVersionService,
		liveKitService,
		idGen,
		pool,
		llmClient,
		asrAdapter,
		ttsAdapter,
		embeddingClient,
		mcpAdapter,
		generateResponseUseCase,
		sendMessageUC,
		syncMessagesUC,
		regenerateResponseUC,
		continueResponseUC,
		editUserMessageUC,
		editAssistantMessageUC,
		runOptimizationUC,
		memorizeFromUpvoteUC,
		processUserMessageUC,
		wsBroadcaster,
	)

	// Set up graceful shutdown
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("HTTP server listening on %s:%d", cfg.Server.Host, cfg.Server.Port)
		serverErrors <- server.Start()
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErrors:
		closeMCPAdapter(mcpAdapter)
		return fmt.Errorf("server error: %w", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		log.Println("Shutting down gracefully...")

		shutdownCtx, shutdownCancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer shutdownCancel()

		if err := server.Stop(shutdownCtx); err != nil {
			closeMCPAdapter(mcpAdapter)
			return fmt.Errorf("server shutdown error: %w", err)
		}

		closeMCPAdapter(mcpAdapter)
		log.Println("Server stopped")
		return nil
	}
}
