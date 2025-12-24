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
	liveKitService          ports.LiveKitService
	idGen                   ports.IDGenerator
	db                      *pgxpool.Pool
	llmClient               *llm.Client
	asrAdapter              *speech.ASRAdapter
	ttsAdapter              *speech.TTSAdapter
	embeddingClient         *embedding.Client
	mcpAdapter              *mcp.Adapter
	generateResponseUseCase ports.GenerateResponseUseCase
	broadcaster             *handlers.SSEBroadcaster
}

func NewServer(
	cfg *config.Config,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	liveKitService ports.LiveKitService,
	idGen ports.IDGenerator,
	db *pgxpool.Pool,
	llmClient *llm.Client,
	asrAdapter *speech.ASRAdapter,
	ttsAdapter *speech.TTSAdapter,
	embeddingClient *embedding.Client,
	mcpAdapter *mcp.Adapter,
	generateResponseUseCase ports.GenerateResponseUseCase,
) *Server {
	s := &Server{
		config:                  cfg,
		conversationRepo:        conversationRepo,
		messageRepo:             messageRepo,
		liveKitService:          liveKitService,
		idGen:                   idGen,
		db:                      db,
		llmClient:               llmClient,
		asrAdapter:              asrAdapter,
		ttsAdapter:              ttsAdapter,
		embeddingClient:         embeddingClient,
		mcpAdapter:              mcpAdapter,
		generateResponseUseCase: generateResponseUseCase,
		broadcaster:             handlers.NewSSEBroadcaster(),
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

	// API routes with authentication
	r.Route("/api/v1", func(r chi.Router) {
		// Apply authentication middleware to all API routes
		r.Use(middleware.Auth)

		conversationsHandler := handlers.NewConversationsHandler(s.conversationRepo, s.idGen)
		r.Post("/conversations", conversationsHandler.Create)
		r.Get("/conversations", conversationsHandler.List)
		r.Get("/conversations/{id}", conversationsHandler.Get)
		r.Delete("/conversations/{id}", conversationsHandler.Delete)

		messagesHandler := handlers.NewMessagesHandler(s.conversationRepo, s.messageRepo, s.idGen, s.generateResponseUseCase, s.broadcaster)
		r.Get("/conversations/{id}/messages", messagesHandler.List)
		r.Post("/conversations/{id}/messages", messagesHandler.Send)

		syncHandler := handlers.NewSyncHandler(s.conversationRepo, s.messageRepo, s.idGen, s.broadcaster)
		r.Post("/conversations/{id}/sync", syncHandler.SyncMessages)
		r.Get("/conversations/{id}/sync/status", syncHandler.GetSyncStatus)

		// SSE endpoint for real-time events
		sseHandler := handlers.NewSSEHandler(s.conversationRepo, s.broadcaster)
		r.Get("/conversations/{id}/events", sseHandler.StreamEvents)

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
		Addr:    addr,
		Handler: s.router,
		// Increased timeouts for SSE long-lived connections
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // No write timeout for SSE streaming
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
