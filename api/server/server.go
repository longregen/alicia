package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/api/config"
	"github.com/longregen/alicia/api/livekit"
	"github.com/longregen/alicia/api/server/handlers"
	"github.com/longregen/alicia/api/services"
	"github.com/longregen/alicia/api/store"
	"github.com/longregen/alicia/pkg/langfuse"
	"github.com/longregen/alicia/pkg/otel"
)

const ReadTimeout = 30 * time.Second

type Server struct {
	cfg    *config.Config
	router *chi.Mux
	server *http.Server
	hub    *Hub
	store  *store.Store
}

func NewServer(
	cfg *config.Config,
	s *store.Store,
	convSvc *services.ConversationService,
	msgSvc *services.MessageService,
	memorySvc *services.MemoryService,
	toolSvc *services.ToolService,
	mcpSvc *services.MCPService,
	prefsSvc *services.PreferencesService,
	noteSvc *services.NoteService,
	lkSvc *livekit.Service,
) *Server {
	hub := NewHub()
	router := chi.NewRouter()

	router.Use(otel.Middleware("alicia-api"))
	router.Use(Recovery)
	router.Use(Logger)
	router.Use(CORS(cfg.Server.AllowedOrigins))

	var lfClient *langfuse.Client
	if cfg.IsLangfuseConfigured() {
		lfClient = langfuse.New(cfg.Langfuse.Host, cfg.Langfuse.PublicKey, cfg.Langfuse.SecretKey)
	}

	healthH := handlers.NewHealthHandler(handlers.HealthHandlerConfig{
		Langfuse: lfClient,
		DBPing:   func(ctx context.Context) error { return s.Pool().Ping(ctx) },
	})
	router.Get("/health", healthH.Readiness)
	router.Get("/health/ready", healthH.Readiness)
	router.Get("/health/live", healthH.Liveness)
	router.Get("/health/full", healthH.Health)

	wsHandler := NewWSHandler(hub, cfg, s)
	router.Get("/api/v1/ws", wsHandler.ServeHTTP)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(AuthWithConfig(AuthConfig{RequireAuth: cfg.Server.RequireAuth}))

		convH := handlers.NewConversationHandler(convSvc)
		r.Post("/conversations", convH.Create)
		r.Get("/conversations", convH.List)
		r.Get("/conversations/{id}", convH.Get)
		r.Patch("/conversations/{id}", convH.Update)
		r.Delete("/conversations/{id}", convH.Delete)

		msgH := handlers.NewMessageHandler(msgSvc, convSvc, hub)
		r.Get("/conversations/{id}/messages", msgH.List)
		r.Post("/conversations/{id}/messages", msgH.Create)
		r.Get("/messages/{id}", msgH.Get)
		r.Get("/messages/{id}/siblings", msgH.GetSiblings)

		toolH := handlers.NewToolHandler(toolSvc, msgSvc, convSvc)
		r.Get("/tools", toolH.ListTools)
		r.Get("/tool-uses", toolH.ListToolUses)
		r.Get("/tool-uses/{id}", toolH.GetToolUse)
		r.Get("/messages/{id}/tool-uses", toolH.GetToolUsesByMessage)

		feedbackH := handlers.NewFeedbackHandlerWithLangfuse(s, lfClient)
		r.Post("/messages/{id}/feedback", feedbackH.CreateMessageFeedback)
		r.Get("/messages/{id}/feedback", feedbackH.GetMessageFeedback)
		r.Post("/tool-uses/{id}/feedback", feedbackH.CreateToolUseFeedback)
		r.Get("/tool-uses/{id}/feedback", feedbackH.GetToolUseFeedback)
		r.Post("/memory-uses/{id}/feedback", feedbackH.CreateMemoryUseFeedback)
		r.Get("/memory-uses/{id}/feedback", feedbackH.GetMemoryUseFeedback)

		memH := handlers.NewMemoryHandler(memorySvc, msgSvc, convSvc)
		r.Get("/memories", memH.List)
		r.Post("/memories", memH.Create)
		r.Post("/memories/search", memH.Search)
		r.Get("/memories/by-tags", memH.GetByTags)
		r.Get("/memories/{id}", memH.Get)
		r.Put("/memories/{id}", memH.Update)
		r.Delete("/memories/{id}", memH.Delete)
		r.Post("/memories/{id}/pin", memH.Pin)
		r.Post("/memories/{id}/archive", memH.Archive)
		r.Post("/memories/{id}/tags", memH.AddTag)
		r.Delete("/memories/{id}/tags/{tag}", memH.RemoveTag)
		r.Get("/messages/{id}/memory-uses", memH.GetMemoryUsesByMessage)

		noteH := handlers.NewNoteHandler(noteSvc)
		r.Post("/notes", noteH.Create)
		r.Get("/notes", noteH.List)
		r.Get("/notes/{id}", noteH.Get)
		r.Put("/notes/{id}", noteH.Update)
		r.Delete("/notes/{id}", noteH.Delete)

		mcpH := handlers.NewMCPHandler(mcpSvc)
		r.Get("/mcp/servers", mcpH.List)
		r.Post("/mcp/servers", mcpH.Create)
		r.Get("/mcp/servers/{name}", mcpH.Get)
		r.Put("/mcp/servers/{name}", mcpH.Update)
		r.Delete("/mcp/servers/{name}", mcpH.Delete)

		prefsH := handlers.NewPreferencesHandler(prefsSvc, hub)
		r.Get("/preferences", prefsH.Get)
		r.Patch("/preferences", prefsH.Update)

		if lkSvc != nil {
			lkH := handlers.NewLiveKitHandler(convSvc, lkSvc)
			r.Post("/conversations/{id}/token", lkH.GetToken)
			r.Post("/conversations/{id}/room", lkH.CreateRoom)
		}

		if cfg.IsHeadscaleConfigured() {
			vpnH := handlers.NewVpnHandler(cfg)
			r.Post("/vpn/auth-key", vpnH.GetAuthKey)
		}
	})

	return &Server{
		cfg:    cfg,
		router: router,
		hub:    hub,
		store:  s,
	}
}

func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: 0,
	}
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
