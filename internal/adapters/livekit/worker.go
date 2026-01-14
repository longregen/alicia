package livekit

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// WorkerConfig configures the LiveKit agent worker
type WorkerConfig struct {
	URL                   string
	APIKey                string
	APISecret             string
	AgentName             string
	TokenValidityDuration time.Duration
	// Room name prefix to filter which rooms this worker handles
	RoomPrefix    string
	WorkerCount   int // Number of worker goroutines for event processing per agent
	WorkQueueSize int // Size of the buffered work queue per agent

	// TTS audio format configuration (for resampling to 48kHz stereo)
	TTSSampleRate int // TTS output sample rate (default: 24000 for Kokoro)
	TTSChannels   int // TTS output channels: 1=mono, 2=stereo (default: 1)
}

// DefaultWorkerConfig returns a default worker configuration
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		URL:                   "ws://localhost:7880",
		APIKey:                "",
		APISecret:             "",
		AgentName:             "alicia-worker",
		TokenValidityDuration: 24 * time.Hour,
		RoomPrefix:            "conv_", // Only handle conversation rooms
		WorkerCount:           10,
		WorkQueueSize:         100,
		TTSSampleRate:         24000, // Kokoro outputs 24kHz
		TTSChannels:           1,     // Kokoro outputs mono
	}
}

// Worker manages LiveKit agent instances and dispatches them to rooms
type Worker struct {
	config       *WorkerConfig
	agentFactory *AgentFactory
	service      *Service

	// Track active agents
	activeAgents map[string]*AgentInstance
	agentsMutex  sync.RWMutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AgentInstance represents an active agent handling a room
type AgentInstance struct {
	Agent          *Agent
	MessageRouter  *MessageRouter
	RoomName       string
	ConversationID string
	ConnectedAt    time.Time
	CancelFunc     context.CancelFunc
}

// NewWorker creates a new agent worker
func NewWorker(config *WorkerConfig, agentFactory *AgentFactory, service *Service) (*Worker, error) {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	if config.URL == "" {
		return nil, fmt.Errorf("LiveKit URL is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("LiveKit API key is required")
	}

	if config.APISecret == "" {
		return nil, fmt.Errorf("LiveKit API secret is required")
	}

	if agentFactory == nil {
		return nil, fmt.Errorf("agent factory is required")
	}

	if service == nil {
		return nil, fmt.Errorf("LiveKit service is required")
	}

	return &Worker{
		config:       config,
		agentFactory: agentFactory,
		service:      service,
		activeAgents: make(map[string]*AgentInstance),
	}, nil
}

// Start begins the worker loop, listening for room assignments
func (w *Worker) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)

	log.Printf("Starting LiveKit agent worker: %s", w.config.AgentName)
	log.Printf("  LiveKit URL: %s", w.config.URL)
	log.Printf("  Room prefix: %s", w.config.RoomPrefix)

	// Start polling for active rooms
	// In a production system, you'd use LiveKit webhooks for room events
	// For now, we'll poll for rooms and dispatch agents as needed
	w.wg.Add(1)
	go w.pollRooms()

	// Wait for context cancellation
	<-w.ctx.Done()

	log.Println("Worker shutting down...")

	// Stop all active agents
	w.stopAllAgents()

	// Wait for goroutines to finish
	w.wg.Wait()

	log.Println("Worker stopped")
	return nil
}

// Stop gracefully stops the worker
func (w *Worker) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// pollRooms periodically checks for rooms that need agent dispatch
func (w *Worker) pollRooms() {
	defer w.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.checkAndDispatchAgents(w.ctx); err != nil {
				log.Printf("Error checking rooms: %v", err)
			}
		}
	}
}

// checkAndDispatchAgents checks active rooms and dispatches agents as needed
func (w *Worker) checkAndDispatchAgents(ctx context.Context) error {
	// List all active rooms
	rooms, err := w.service.roomClient.ListRooms(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list rooms: %w", err)
	}

	for _, room := range rooms.GetRooms() {
		// Only handle rooms matching our prefix
		if !strings.HasPrefix(room.Name, w.config.RoomPrefix) {
			continue
		}

		// Check if we already have an agent for this room
		w.agentsMutex.RLock()
		_, exists := w.activeAgents[room.Name]
		w.agentsMutex.RUnlock()

		if exists {
			// Agent already handling this room
			continue
		}

		// Check if room has participants (NumParticipants includes all participants)
		// We only dispatch if there are participants in the room
		if room.NumParticipants == 0 {
			// No users in room, don't dispatch agent
			continue
		}

		// Dispatch an agent to this room
		log.Printf("Dispatching agent to room: %s (participants: %d)", room.Name, room.NumParticipants)
		if err := w.dispatchAgent(ctx, room.Name); err != nil {
			log.Printf("Failed to dispatch agent to room %s: %v", room.Name, err)
		}
	}

	return nil
}

// dispatchAgent creates and starts an agent for a specific room
func (w *Worker) dispatchAgent(ctx context.Context, roomName string) error {
	// Extract conversation ID from room name
	conversationID := strings.TrimPrefix(roomName, w.config.RoomPrefix)
	if conversationID == "" {
		return fmt.Errorf("invalid room name format: %s", roomName)
	}

	// Create agent configuration
	agentConfig := &AgentConfig{
		URL:                   w.config.URL,
		APIKey:                w.config.APIKey,
		APISecret:             w.config.APISecret,
		AgentIdentity:         fmt.Sprintf("alicia-agent-%s", conversationID),
		AgentName:             "Alicia",
		TokenValidityDuration: w.config.TokenValidityDuration,
		WorkerCount:           w.config.WorkerCount,
		WorkQueueSize:         w.config.WorkQueueSize,
		TTSSampleRate:         w.config.TTSSampleRate,
		TTSChannels:           w.config.TTSChannels,
	}

	// Create agent and message router using factory
	agent, messageRouter, err := w.agentFactory.CreateAgent(agentConfig, conversationID)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Create context for this agent instance
	agentCtx, agentCancel := context.WithCancel(context.Background())

	// Connect agent to room
	if err := agent.Connect(agentCtx, roomName); err != nil {
		agentCancel()
		return fmt.Errorf("failed to connect agent to room: %w", err)
	}

	// Create agent instance
	instance := &AgentInstance{
		Agent:          agent,
		MessageRouter:  messageRouter,
		RoomName:       roomName,
		ConversationID: conversationID,
		ConnectedAt:    time.Now(),
		CancelFunc:     agentCancel,
	}

	// Register agent instance
	w.agentsMutex.Lock()
	w.activeAgents[roomName] = instance
	w.agentsMutex.Unlock()

	log.Printf("Agent connected to room: %s (conversation: %s)", roomName, conversationID)

	// Monitor agent lifecycle
	w.wg.Add(1)
	go w.monitorAgent(instance)

	return nil
}

// monitorAgent monitors an agent instance and cleans up when it disconnects
func (w *Worker) monitorAgent(instance *AgentInstance) {
	defer w.wg.Done()

	// Create a ticker to check agent status
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			// Worker is shutting down
			w.stopAgent(instance)
			return

		case <-ticker.C:
			// Check if agent is still connected
			if !instance.Agent.IsConnected() {
				log.Printf("Agent disconnected from room: %s", instance.RoomName)
				w.removeAgent(instance.RoomName)
				return
			}

			// Check if room still exists and has participants
			room, err := w.service.GetRoom(w.ctx, instance.RoomName)
			if err != nil {
				log.Printf("Room %s no longer exists, stopping agent", instance.RoomName)
				w.stopAgent(instance)
				w.removeAgent(instance.RoomName)
				return
			}

			// Check if there are any non-agent participants
			hasUsers := false
			for _, p := range room.Participants {
				if !strings.HasPrefix(p.Identity, "alicia-agent") {
					hasUsers = true
					break
				}
			}

			if !hasUsers {
				log.Printf("No users remaining in room %s, stopping agent", instance.RoomName)
				w.stopAgent(instance)
				w.removeAgent(instance.RoomName)
				return
			}
		}
	}
}

// stopAgent gracefully stops an agent instance
func (w *Worker) stopAgent(instance *AgentInstance) {
	log.Printf("Stopping agent for room: %s", instance.RoomName)

	// Cancel agent context
	instance.CancelFunc()

	// Disconnect agent
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := instance.Agent.Disconnect(ctx); err != nil {
		log.Printf("Error disconnecting agent from room %s: %v", instance.RoomName, err)
	}
}

// removeAgent removes an agent from the active agents map
func (w *Worker) removeAgent(roomName string) {
	w.agentsMutex.Lock()
	defer w.agentsMutex.Unlock()

	delete(w.activeAgents, roomName)
	log.Printf("Agent removed from tracking: %s", roomName)
}

// stopAllAgents stops all active agents
func (w *Worker) stopAllAgents() {
	w.agentsMutex.Lock()
	agents := make([]*AgentInstance, 0, len(w.activeAgents))
	for _, instance := range w.activeAgents {
		agents = append(agents, instance)
	}
	w.activeAgents = make(map[string]*AgentInstance)
	w.agentsMutex.Unlock()

	log.Printf("Stopping %d active agents...", len(agents))

	// Stop all agents in parallel
	var stopWg sync.WaitGroup
	for _, instance := range agents {
		stopWg.Add(1)
		go func(inst *AgentInstance) {
			defer stopWg.Done()
			w.stopAgent(inst)
		}(instance)
	}

	stopWg.Wait()
	log.Println("All agents stopped")
}

// GetActiveAgents returns information about currently active agents
func (w *Worker) GetActiveAgents() map[string]*AgentInstance {
	w.agentsMutex.RLock()
	defer w.agentsMutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]*AgentInstance, len(w.activeAgents))
	for k, v := range w.activeAgents {
		result[k] = v
	}

	return result
}

// DispatchToRoom manually dispatches an agent to a specific room
// This is useful for testing or manual intervention
func (w *Worker) DispatchToRoom(ctx context.Context, roomName string) error {
	return w.dispatchAgent(ctx, roomName)
}
