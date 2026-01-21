package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/adapters/embedding"
	"github.com/longregen/alicia/internal/adapters/http/handlers"
	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/adapters/speech"
	"github.com/longregen/alicia/internal/application/usecases"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	config                      *config.Config
	router                      *chi.Mux
	httpServer                  *http.Server
	conversationRepo            ports.ConversationRepository
	messageRepo                 ports.MessageRepository
	toolUseRepo                 ports.ToolUseRepository
	memoryUsageRepo             ports.MemoryUsageRepository
	noteRepo                    ports.NoteRepository
	voteRepo                    ports.VoteRepository
	sessionStatsRepo            ports.SessionStatsRepository
	memoryService               ports.MemoryService
	liveKitService              ports.LiveKitService
	idGen                       ports.IDGenerator
	db                          *pgxpool.Pool
	llmClient                   *llm.Client
	asrAdapter                  *speech.ASRAdapter
	ttsAdapter                  *speech.TTSAdapter
	embeddingClient             *embedding.Client
	mcpAdapter                  *mcp.Adapter
	generateResponseUseCase     ports.GenerateResponseUseCase
	sendMessageUseCase          ports.SendMessageUseCase
	syncMessagesUseCase         ports.SyncMessagesUseCase
	regenerateResponseUseCase   ports.RegenerateResponseUseCase
	continueResponseUseCase     ports.ContinueResponseUseCase
	editUserMessageUseCase      ports.EditUserMessageUseCase
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase
	memorizeFromUpvoteUseCase   *usecases.MemorizeFromUpvote
	processUserMessageUseCase   ports.ProcessUserMessageUseCase
	wsBroadcaster               *handlers.WebSocketBroadcaster
}

func NewServer(
	cfg *config.Config,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	noteRepo ports.NoteRepository,
	voteRepo ports.VoteRepository,
	sessionStatsRepo ports.SessionStatsRepository,
	memoryService ports.MemoryService,
	liveKitService ports.LiveKitService,
	idGen ports.IDGenerator,
	db *pgxpool.Pool,
	llmClient *llm.Client,
	asrAdapter *speech.ASRAdapter,
	ttsAdapter *speech.TTSAdapter,
	embeddingClient *embedding.Client,
	mcpAdapter *mcp.Adapter,
	generateResponseUseCase ports.GenerateResponseUseCase,
	sendMessageUseCase ports.SendMessageUseCase,
	syncMessagesUseCase ports.SyncMessagesUseCase,
	regenerateResponseUseCase ports.RegenerateResponseUseCase,
	continueResponseUseCase ports.ContinueResponseUseCase,
	editUserMessageUseCase ports.EditUserMessageUseCase,
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase,
	memorizeFromUpvoteUseCase *usecases.MemorizeFromUpvote,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	wsBroadcaster *handlers.WebSocketBroadcaster,
) *Server {
	s := &Server{
		config:                      cfg,
		conversationRepo:            conversationRepo,
		messageRepo:                 messageRepo,
		toolUseRepo:                 toolUseRepo,
		memoryUsageRepo:             memoryUsageRepo,
		noteRepo:                    noteRepo,
		voteRepo:                    voteRepo,
		sessionStatsRepo:            sessionStatsRepo,
		memoryService:               memoryService,
		liveKitService:              liveKitService,
		idGen:                       idGen,
		db:                          db,
		llmClient:                   llmClient,
		asrAdapter:                  asrAdapter,
		ttsAdapter:                  ttsAdapter,
		embeddingClient:             embeddingClient,
		mcpAdapter:                  mcpAdapter,
		generateResponseUseCase:     generateResponseUseCase,
		sendMessageUseCase:          sendMessageUseCase,
		syncMessagesUseCase:         syncMessagesUseCase,
		regenerateResponseUseCase:   regenerateResponseUseCase,
		continueResponseUseCase:     continueResponseUseCase,
		editUserMessageUseCase:      editUserMessageUseCase,
		editAssistantMessageUseCase: editAssistantMessageUseCase,
		memorizeFromUpvoteUseCase:   memorizeFromUpvoteUseCase,
		processUserMessageUseCase:   processUserMessageUseCase,
		wsBroadcaster:               wsBroadcaster,
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

	if s.ttsAdapter != nil {
		ttsHandler := handlers.NewTTSHandler(s.ttsAdapter)
		r.Post("/v1/audio/speech", ttsHandler.Speech)
	}

	configHandler := handlers.NewConfigHandler(s.config, s.ttsAdapter)
	r.Get("/api/v1/config", configHandler.GetPublicConfig)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth)

		conversationsHandler := handlers.NewConversationsHandler(s.conversationRepo, s.memoryService, s.idGen)
		r.Post("/conversations", conversationsHandler.Create)
		r.Get("/conversations", conversationsHandler.List)
		r.Get("/conversations/{id}", conversationsHandler.Get)
		r.Patch("/conversations/{id}", conversationsHandler.Patch)
		r.Delete("/conversations/{id}", conversationsHandler.Delete)

		messagesHandler := handlers.NewMessagesHandler(
			s.conversationRepo,
			s.messageRepo,
			s.toolUseRepo,
			s.memoryUsageRepo,
			s.sendMessageUseCase,
			s.processUserMessageUseCase,
			s.editAssistantMessageUseCase,
			s.editUserMessageUseCase,
			s.regenerateResponseUseCase,
			s.continueResponseUseCase,
			s.wsBroadcaster,
			s.idGen,
		)
		r.Get("/conversations/{id}/messages", messagesHandler.List)
		r.Post("/conversations/{id}/messages", messagesHandler.Send)
		r.Get("/messages/{id}/siblings", messagesHandler.GetSiblings)
		r.Post("/conversations/{id}/switch-branch", messagesHandler.SwitchBranch)

		r.Put("/messages/{id}", messagesHandler.EditAssistantMessage)
		r.Put("/messages/{id}/edit-user", messagesHandler.EditUserMessage)
		r.Post("/messages/{id}/regenerate", messagesHandler.Regenerate)
		r.Post("/messages/{id}/continue", messagesHandler.Continue)

		syncHandler := handlers.NewSyncHandler(s.conversationRepo, s.messageRepo, s.syncMessagesUseCase)
		r.Post("/conversations/{id}/sync", syncHandler.SyncMessages)
		r.Get("/conversations/{id}/sync/status", syncHandler.GetSyncStatus)

		wsHandler := handlers.NewWebSocketSyncHandler(s.conversationRepo, s.messageRepo, s.idGen, s.wsBroadcaster, s.config.Server.CORSOrigins)
		r.Get("/conversations/{id}/sync/ws", wsHandler.Handle)

		multiplexedWSHandler := handlers.NewMultiplexedWSHandler(s.conversationRepo, s.messageRepo, s.idGen, s.wsBroadcaster, s.config.Server.CORSOrigins)
		r.Get("/ws", multiplexedWSHandler.Handle)

		tokenHandler := handlers.NewTokenHandler(s.conversationRepo, s.liveKitService)
		r.Post("/conversations/{id}/token", tokenHandler.Generate)

		if s.mcpAdapter != nil {
			mcpHandler := handlers.NewMCPHandler(s.mcpAdapter)
			r.Get("/mcp/servers", mcpHandler.ListServers)
			r.Post("/mcp/servers", mcpHandler.AddServer)
			r.Delete("/mcp/servers/{name}", mcpHandler.RemoveServer)
			r.Get("/mcp/tools", mcpHandler.ListTools)
		}

		voteHandler := handlers.NewVoteHandler(s.voteRepo, s.idGen, s.memorizeFromUpvoteUseCase)
		r.Post("/votes/batch/messages", voteHandler.GetBatchMessageVotes)
		r.Post("/messages/{id}/vote", voteHandler.VoteOnMessage)
		r.Delete("/messages/{id}/vote", voteHandler.RemoveMessageVote)
		r.Get("/messages/{id}/votes", voteHandler.GetMessageVotes)
		r.Post("/tool-uses/{id}/vote", voteHandler.VoteOnToolUse)
		r.Delete("/tool-uses/{id}/vote", voteHandler.RemoveToolUseVote)
		r.Get("/tool-uses/{id}/votes", voteHandler.GetToolUseVotes)
		r.Post("/tool-uses/{id}/quick-feedback", voteHandler.ToolUseQuickFeedback)
		r.Post("/memories/{id}/vote", voteHandler.VoteOnMemory)
		r.Delete("/memories/{id}/vote", voteHandler.RemoveMemoryVote)
		r.Get("/memories/{id}/votes", voteHandler.GetMemoryVotes)
		r.Post("/memories/{id}/irrelevance-reason", voteHandler.MemoryIrrelevanceReason)
		r.Route("/memory-usages/{id}", func(r chi.Router) {
			r.Post("/vote", voteHandler.VoteOnMemoryUsage)
			r.Delete("/vote", voteHandler.RemoveMemoryUsageVote)
			r.Get("/votes", voteHandler.GetMemoryUsageVotes)
			r.Post("/irrelevance-reason", voteHandler.MemoryUsageIrrelevanceReason)
		})
		r.Route("/messages/{messageId}/extracted-memories/{memoryId}", func(r chi.Router) {
			r.Post("/vote", voteHandler.VoteOnMemoryExtraction)
			r.Delete("/vote", voteHandler.RemoveMemoryExtractionVote)
			r.Get("/votes", voteHandler.GetMemoryExtractionVotes)
			r.Post("/quality-feedback", voteHandler.MemoryExtractionQualityFeedback)
		})
		r.Post("/reasoning/{id}/vote", voteHandler.VoteOnReasoning)
		r.Delete("/reasoning/{id}/vote", voteHandler.RemoveReasoningVote)
		r.Get("/reasoning/{id}/votes", voteHandler.GetReasoningVotes)
		r.Post("/reasoning/{id}/issue", voteHandler.ReasoningIssue)

		noteHandler := handlers.NewNoteHandler(s.noteRepo, s.idGen)
		r.Post("/messages/{id}/notes", noteHandler.CreateMessageNote)
		r.Get("/messages/{id}/notes", noteHandler.GetMessageNotes)
		r.Post("/tool-uses/{id}/notes", noteHandler.CreateToolUseNote)
		r.Post("/reasoning/{id}/notes", noteHandler.CreateReasoningNote)
		r.Put("/notes/{id}", noteHandler.UpdateNote)
		r.Delete("/notes/{id}", noteHandler.DeleteNote)

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

		serverInfoHandler := handlers.NewServerInfoHandler(s.config, s.conversationRepo, s.messageRepo, s.mcpAdapter)
		r.Get("/server/info", serverInfoHandler.GetServerInfo)
		r.Get("/server/stats", serverInfoHandler.GetGlobalStats)
		r.Get("/conversations/{id}/stats", serverInfoHandler.GetSessionStats)
	})

	if s.config.Server.StaticDir != "" {
		fileServer := http.FileServer(http.Dir(s.config.Server.StaticDir))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			path := req.URL.Path

			if strings.HasPrefix(path, "/js/lib/") ||
				strings.HasPrefix(path, "/onnx/") ||
				strings.HasPrefix(path, "/models/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}

			fileServer.ServeHTTP(w, req)
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
