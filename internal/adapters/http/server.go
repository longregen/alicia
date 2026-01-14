package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/adapters/embedding"
	"github.com/longregen/alicia/internal/adapters/http/handlers"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	config                  *config.Config
	router                  *chi.Mux
	httpServer              *http.Server
	conversationRepo        ports.ConversationRepository
	messageRepo             ports.MessageRepository
	toolUseRepo             ports.ToolUseRepository
	noteRepo                ports.NoteRepository
	voteRepo                ports.VoteRepository
	sessionStatsRepo        ports.SessionStatsRepository
	memoryService           ports.MemoryService
	optimizationService     ports.OptimizationService
	optimizationRepo        ports.PromptOptimizationRepository
	trainingBuilderService  *services.TrainingSetBuilderService
	promptVersionService    *services.PromptVersionService
	liveKitService          ports.LiveKitService
	idGen                   ports.IDGenerator
	db                      *pgxpool.Pool
	llmClient               *llm.Client
	asrAdapter              *speech.ASRAdapter
	ttsAdapter              *speech.TTSAdapter
	embeddingClient         *embedding.Client
	mcpAdapter              *mcp.Adapter
	generateResponseUseCase ports.GenerateResponseUseCase
	wsBroadcaster           *handlers.WebSocketBroadcaster
}

func NewServer(
	cfg *config.Config,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	toolUseRepo ports.ToolUseRepository,
	noteRepo ports.NoteRepository,
	voteRepo ports.VoteRepository,
	sessionStatsRepo ports.SessionStatsRepository,
	memoryService ports.MemoryService,
	optimizationService ports.OptimizationService,
	optimizationRepo ports.PromptOptimizationRepository,
	trainingBuilderService *services.TrainingSetBuilderService,
	promptVersionService *services.PromptVersionService,
	liveKitService ports.LiveKitService,
	idGen ports.IDGenerator,
	db *pgxpool.Pool,
	llmClient *llm.Client,
	asrAdapter *speech.ASRAdapter,
	ttsAdapter *speech.TTSAdapter,
	embeddingClient *embedding.Client,
	mcpAdapter *mcp.Adapter,
	generateResponseUseCase ports.GenerateResponseUseCase,
	wsBroadcaster *handlers.WebSocketBroadcaster,
) *Server {
	s := &Server{
		config:                  cfg,
		conversationRepo:        conversationRepo,
		messageRepo:             messageRepo,
		toolUseRepo:             toolUseRepo,
		noteRepo:                noteRepo,
		voteRepo:                voteRepo,
		sessionStatsRepo:        sessionStatsRepo,
		memoryService:           memoryService,
		optimizationService:     optimizationService,
		optimizationRepo:        optimizationRepo,
		trainingBuilderService:  trainingBuilderService,
		promptVersionService:    promptVersionService,
		liveKitService:          liveKitService,
		idGen:                   idGen,
		db:                      db,
		llmClient:               llmClient,
		asrAdapter:              asrAdapter,
		ttsAdapter:              ttsAdapter,
		embeddingClient:         embeddingClient,
		mcpAdapter:              mcpAdapter,
		generateResponseUseCase: generateResponseUseCase,
		wsBroadcaster:           wsBroadcaster,
	}

	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(s.config.Server.CORSOrigins))
	r.Use(middleware.Metrics)

	// Health check endpoints (no auth required)
	healthHandler := handlers.NewHealthHandler()
	detailedHealthHandler := handlers.NewHealthHandlerWithDeps(
		s.config,
		s.db,
		s.llmClient,
		s.asrAdapter,
		s.ttsAdapter,
		s.embeddingClient,
		s.liveKitService,
	)
	r.Get("/health", healthHandler.Handle)
	r.Get("/health/detailed", detailedHealthHandler.HandleDetailed)
	r.Handle("/metrics", promhttp.Handler())

	// OpenAI-compatible TTS endpoint (no auth required for voice preview)
	if s.ttsAdapter != nil {
		ttsHandler := handlers.NewTTSHandler(s.ttsAdapter)
		r.Post("/v1/audio/speech", ttsHandler.Speech)
	}

	// Public config endpoint (no auth required - frontend needs this before auth)
	configHandler := handlers.NewConfigHandler(s.config, s.ttsAdapter)
	r.Get("/api/v1/config", configHandler.GetPublicConfig)

	// API routes with authentication
	r.Route("/api/v1", func(r chi.Router) {
		// Apply authentication middleware to all API routes
		r.Use(middleware.Auth)

		conversationsHandler := handlers.NewConversationsHandler(s.conversationRepo, s.idGen)
		r.Post("/conversations", conversationsHandler.Create)
		r.Get("/conversations", conversationsHandler.List)
		r.Get("/conversations/{id}", conversationsHandler.Get)
		r.Patch("/conversations/{id}", conversationsHandler.Patch)
		r.Delete("/conversations/{id}", conversationsHandler.Delete)

		messagesHandler := handlers.NewMessagesHandler(s.conversationRepo, s.messageRepo, s.toolUseRepo, s.idGen, s.generateResponseUseCase, s.wsBroadcaster)
		r.Get("/conversations/{id}/messages", messagesHandler.List)
		r.Post("/conversations/{id}/messages", messagesHandler.Send)
		r.Get("/messages/{id}/siblings", messagesHandler.GetSiblings)
		r.Post("/conversations/{id}/switch-branch", messagesHandler.SwitchBranch)

		syncHandler := handlers.NewSyncHandler(s.conversationRepo, s.messageRepo, s.idGen)
		r.Post("/conversations/{id}/sync", syncHandler.SyncMessages)
		r.Get("/conversations/{id}/sync/status", syncHandler.GetSyncStatus)

		// WebSocket endpoint for MessagePack sync (per-conversation, legacy)
		wsHandler := handlers.NewWebSocketSyncHandler(s.conversationRepo, s.messageRepo, s.idGen, s.wsBroadcaster, s.config.Server.CORSOrigins)
		r.Get("/conversations/{id}/sync/ws", wsHandler.Handle)

		// Multiplexed WebSocket endpoint (single connection, multiple conversations)
		multiplexedWSHandler := handlers.NewMultiplexedWSHandler(s.conversationRepo, s.messageRepo, s.idGen, s.wsBroadcaster, s.config.Server.CORSOrigins)
		r.Get("/ws", multiplexedWSHandler.Handle)

		tokenHandler := handlers.NewTokenHandler(s.conversationRepo, s.liveKitService)
		r.Post("/conversations/{id}/token", tokenHandler.Generate)

		// MCP routes (only if MCP adapter is available)
		if s.mcpAdapter != nil {
			mcpHandler := handlers.NewMCPHandler(s.mcpAdapter)
			r.Get("/mcp/servers", mcpHandler.ListServers)
			r.Post("/mcp/servers", mcpHandler.AddServer)
			r.Delete("/mcp/servers/{name}", mcpHandler.RemoveServer)
			r.Get("/mcp/tools", mcpHandler.ListTools)
		}

		// Voting endpoints
		voteHandler := handlers.NewVoteHandler(s.voteRepo, s.idGen)
		// Message voting
		r.Post("/messages/{id}/vote", voteHandler.VoteOnMessage)
		r.Delete("/messages/{id}/vote", voteHandler.RemoveMessageVote)
		r.Get("/messages/{id}/votes", voteHandler.GetMessageVotes)
		// Tool use voting
		r.Post("/tool-uses/{id}/vote", voteHandler.VoteOnToolUse)
		r.Delete("/tool-uses/{id}/vote", voteHandler.RemoveToolUseVote)
		r.Get("/tool-uses/{id}/votes", voteHandler.GetToolUseVotes)
		r.Post("/tool-uses/{id}/quick-feedback", voteHandler.ToolUseQuickFeedback)
		// Memory voting
		r.Post("/memories/{id}/vote", voteHandler.VoteOnMemory)
		r.Delete("/memories/{id}/vote", voteHandler.RemoveMemoryVote)
		r.Get("/memories/{id}/votes", voteHandler.GetMemoryVotes)
		r.Post("/memories/{id}/irrelevance-reason", voteHandler.MemoryIrrelevanceReason)
		// Memory usage voting (replaces per-memory voting for selection)
		r.Route("/memory-usages/{id}", func(r chi.Router) {
			r.Post("/vote", voteHandler.VoteOnMemoryUsage)
			r.Delete("/vote", voteHandler.RemoveMemoryUsageVote)
			r.Get("/votes", voteHandler.GetMemoryUsageVotes)
			r.Post("/irrelevance-reason", voteHandler.MemoryUsageIrrelevanceReason)
		})
		// Memory extraction voting (nested under messages)
		r.Route("/messages/{messageId}/extracted-memories/{memoryId}", func(r chi.Router) {
			r.Post("/vote", voteHandler.VoteOnMemoryExtraction)
			r.Delete("/vote", voteHandler.RemoveMemoryExtractionVote)
			r.Get("/votes", voteHandler.GetMemoryExtractionVotes)
			r.Post("/quality-feedback", voteHandler.MemoryExtractionQualityFeedback)
		})
		// Reasoning voting
		r.Post("/reasoning/{id}/vote", voteHandler.VoteOnReasoning)
		r.Delete("/reasoning/{id}/vote", voteHandler.RemoveReasoningVote)
		r.Get("/reasoning/{id}/votes", voteHandler.GetReasoningVotes)
		r.Post("/reasoning/{id}/issue", voteHandler.ReasoningIssue)

		// Note endpoints
		noteHandler := handlers.NewNoteHandler(s.noteRepo, s.idGen)
		r.Post("/messages/{id}/notes", noteHandler.CreateMessageNote)
		r.Get("/messages/{id}/notes", noteHandler.GetMessageNotes)
		r.Post("/tool-uses/{id}/notes", noteHandler.CreateToolUseNote)
		r.Post("/reasoning/{id}/notes", noteHandler.CreateReasoningNote)
		r.Put("/notes/{id}", noteHandler.UpdateNote)
		r.Delete("/notes/{id}", noteHandler.DeleteNote)

		// Memory management endpoints (only if memory service is available)
		if s.memoryService != nil {
			memoryHandler := handlers.NewMemoryHandler(s.memoryService)
			r.Post("/memories", memoryHandler.CreateMemory)
			r.Get("/memories", memoryHandler.ListMemories)
			r.Post("/memories/search", memoryHandler.SearchMemories)
			r.Get("/memories/by-tags", memoryHandler.GetByTags)
			r.Get("/memories/{id}", memoryHandler.GetMemory)
			r.Put("/memories/{id}", memoryHandler.UpdateMemory)
			r.Delete("/memories/{id}", memoryHandler.DeleteMemory)
			r.Post("/memories/{id}/tags", memoryHandler.AddTag)
			r.Delete("/memories/{id}/tags/{tag}", memoryHandler.RemoveTag)
			r.Put("/memories/{id}/importance", memoryHandler.SetImportance)
			r.Post("/memories/{id}/pin", memoryHandler.PinMemory)
			r.Post("/memories/{id}/archive", memoryHandler.ArchiveMemory)
		}

		// Server info and stats endpoints
		serverInfoHandler := handlers.NewServerInfoHandler(s.config, s.conversationRepo, s.messageRepo, s.mcpAdapter)
		r.Get("/server/info", serverInfoHandler.GetServerInfo)
		r.Get("/server/stats", serverInfoHandler.GetGlobalStats)
		r.Get("/conversations/{id}/stats", serverInfoHandler.GetSessionStats)

		// Optimization endpoints (only if optimization service is available)
		if s.optimizationService != nil {
			optHandler := handlers.NewOptimizationHandler(s.optimizationService)
			r.Post("/optimizations", optHandler.CreateOptimization)
			r.Get("/optimizations", optHandler.ListOptimizations)
			r.Get("/optimizations/{id}", optHandler.GetOptimization)
			r.Get("/optimizations/{id}/candidates", optHandler.GetCandidates)
			r.Get("/optimizations/{id}/best", optHandler.GetBestCandidate)
			r.Get("/optimizations/{id}/program", optHandler.GetOptimizedProgram)
			r.Get("/optimizations/candidates/{id}/evaluations", optHandler.GetEvaluations)

			// Optimization progress streaming
			if s.optimizationRepo != nil {
				// Cast to concrete type to access progress channel methods
				if concreteOptService, ok := s.optimizationService.(*services.OptimizationService); ok {
					streamHandler := handlers.NewOptimizationStreamHandler(concreteOptService)
					r.Get("/optimizations/{id}/stream", streamHandler.StreamOptimizationProgress)
				} else {
					log.Printf("Warning: optimization service is not *services.OptimizationService, SSE streaming disabled")
				}
			}

			// Feedback integration endpoints
			feedbackHandler := handlers.NewFeedbackHandler(s.voteRepo, s.optimizationService)
			r.Post("/feedback", feedbackHandler.SubmitFeedback)
			r.Get("/feedback/dimensions", feedbackHandler.GetDimensionWeights)
			r.Put("/feedback/dimensions", feedbackHandler.UpdateDimensionWeights)

			// Deployment endpoints (Phase 6)
			deploymentService := services.NewDeploymentService(s.optimizationRepo, s.idGen)
			deploymentHandler := handlers.NewDeploymentHandler(deploymentService)
			r.Post("/deployments", deploymentHandler.DeployPrompt)
			r.Get("/deployments/{prompt_type}/active", deploymentHandler.GetActiveDeployment)
			r.Get("/deployments/{prompt_type}/history", deploymentHandler.ListDeploymentHistory)
			r.Delete("/deployments/{run_id}", deploymentHandler.RollbackDeployment)
		}

		// Training and prompt version endpoints (Phase 8)
		if s.trainingBuilderService != nil && s.optimizationService != nil && s.promptVersionService != nil {
			// Type assert optimizationService to concrete type for TrainingHandler
			optService, ok := s.optimizationService.(*services.OptimizationService)
			if ok {
				trainingHandler := handlers.NewTrainingHandler(
					s.trainingBuilderService,
					optService,
					s.promptVersionService,
				)

				r.Route("/training", func(r chi.Router) {
					r.Get("/stats", trainingHandler.GetTrainingStats)
					r.Post("/optimize", trainingHandler.RunOptimization)
				})

				r.Route("/prompts", func(r chi.Router) {
					r.Get("/versions", trainingHandler.ListPromptVersions)
					r.Post("/versions/{id}/activate", trainingHandler.ActivatePromptVersion)
				})
			}
		}
	})

	// Serve frontend static files if configured (no auth required)
	if s.config.Server.StaticDir != "" {
		fileServer := http.FileServer(http.Dir(s.config.Server.StaticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			fileServer.ServeHTTP(w, r)
		})
	}

	s.router = r
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // No write timeout for WebSocket streaming
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	log.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}
