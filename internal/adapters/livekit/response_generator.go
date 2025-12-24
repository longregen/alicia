package livekit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ActiveGeneration tracks an active response generation
type ActiveGeneration struct {
	MessageID  string
	CancelFunc context.CancelFunc
	StartedAt  time.Time
}

// ResponseGenerationManager manages the lifecycle of active response generations and TTS operations
type ResponseGenerationManager interface {
	// Generation management
	RegisterGeneration(messageID string, cancelFunc context.CancelFunc)
	UnregisterGeneration(messageID string)
	CancelGeneration(targetID string) error
	CleanupStaleGenerations(maxAge time.Duration) int

	// TTS management
	RegisterTTS(targetID string, cancelFunc context.CancelFunc)
	UnregisterTTS(targetID string)
	CancelTTS(targetID string) error
}

// DefaultResponseGenerationManager implements ResponseGenerationManager
type DefaultResponseGenerationManager struct {
	// Track active generations for cancellation
	activeGenerations map[string]*ActiveGeneration
	generationsMutex  sync.RWMutex

	// Track active TTS operations for cancellation
	activeTTS map[string]context.CancelFunc
	ttsMutex  sync.RWMutex

	// Background cleanup
	cleanupStopCh chan struct{}
	cleanupDone   chan struct{}
}

// NewDefaultResponseGenerationManager creates a new response generation manager
func NewDefaultResponseGenerationManager() *DefaultResponseGenerationManager {
	return &DefaultResponseGenerationManager{
		activeGenerations: make(map[string]*ActiveGeneration),
		activeTTS:         make(map[string]context.CancelFunc),
		cleanupStopCh:     make(chan struct{}),
		cleanupDone:       make(chan struct{}),
	}
}

// RegisterGeneration registers an active generation for tracking
func (m *DefaultResponseGenerationManager) RegisterGeneration(messageID string, cancelFunc context.CancelFunc) {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	m.activeGenerations[messageID] = &ActiveGeneration{
		MessageID:  messageID,
		CancelFunc: cancelFunc,
		StartedAt:  time.Now(),
	}
}

// UnregisterGeneration removes a generation from tracking
func (m *DefaultResponseGenerationManager) UnregisterGeneration(messageID string) {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	delete(m.activeGenerations, messageID)
}

// CancelGeneration cancels an active generation
func (m *DefaultResponseGenerationManager) CancelGeneration(targetID string) error {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	// If targetID is empty, cancel all active generations
	if targetID == "" {
		if len(m.activeGenerations) == 0 {
			log.Printf("No active generations to cancel")
			return nil
		}

		for id, gen := range m.activeGenerations {
			log.Printf("Cancelling generation for message: %s", id)
			gen.CancelFunc()
		}
		m.activeGenerations = make(map[string]*ActiveGeneration)
		return nil
	}

	// Cancel specific generation
	gen, exists := m.activeGenerations[targetID]
	if !exists {
		log.Printf("No active generation found for target: %s", targetID)
		return fmt.Errorf("no active generation found for target: %s", targetID)
	}

	log.Printf("Cancelling generation for message: %s", targetID)
	gen.CancelFunc()
	delete(m.activeGenerations, targetID)

	return nil
}

// CleanupStaleGenerations removes and cancels generations that have been running longer than maxAge
// Returns the number of stale generations that were cleaned up
func (m *DefaultResponseGenerationManager) CleanupStaleGenerations(maxAge time.Duration) int {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	now := time.Now()
	cleanedCount := 0
	staleIDs := make([]string, 0)

	// Find all stale generations
	for id, gen := range m.activeGenerations {
		if now.Sub(gen.StartedAt) > maxAge {
			staleIDs = append(staleIDs, id)
		}
	}

	// Cancel and remove stale generations
	for _, id := range staleIDs {
		if gen, exists := m.activeGenerations[id]; exists {
			log.Printf("Cleaning up stale generation: %s (age: %v)", id, now.Sub(gen.StartedAt))
			gen.CancelFunc()
			delete(m.activeGenerations, id)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		log.Printf("Cleaned up %d stale generation(s)", cleanedCount)
	}

	return cleanedCount
}

// RegisterTTS registers an active TTS operation for tracking
func (m *DefaultResponseGenerationManager) RegisterTTS(targetID string, cancelFunc context.CancelFunc) {
	m.ttsMutex.Lock()
	defer m.ttsMutex.Unlock()

	m.activeTTS[targetID] = cancelFunc
}

// UnregisterTTS removes a TTS operation from tracking
func (m *DefaultResponseGenerationManager) UnregisterTTS(targetID string) {
	m.ttsMutex.Lock()
	defer m.ttsMutex.Unlock()

	delete(m.activeTTS, targetID)
}

// CancelTTS cancels active TTS operations
func (m *DefaultResponseGenerationManager) CancelTTS(targetID string) error {
	m.ttsMutex.Lock()
	defer m.ttsMutex.Unlock()

	// If targetID is empty, cancel all active TTS operations
	if targetID == "" {
		if len(m.activeTTS) == 0 {
			log.Printf("No active TTS operations to cancel")
			return nil
		}

		for id, cancel := range m.activeTTS {
			log.Printf("Cancelling TTS for target: %s", id)
			cancel()
		}
		m.activeTTS = make(map[string]context.CancelFunc)
		return nil
	}

	// Cancel specific TTS operation
	cancel, exists := m.activeTTS[targetID]
	if !exists {
		log.Printf("No active TTS found for target: %s", targetID)
		return fmt.Errorf("no active TTS found for target: %s", targetID)
	}

	log.Printf("Cancelling TTS for target: %s", targetID)
	cancel()
	delete(m.activeTTS, targetID)

	return nil
}

// StartPeriodicCleanup starts a background goroutine that periodically cleans up stale generations
// Call StopPeriodicCleanup to stop the cleanup routine
func (m *DefaultResponseGenerationManager) StartPeriodicCleanup(interval time.Duration, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(m.cleanupDone)

		log.Printf("Started periodic cleanup of stale generations (interval: %v, maxAge: %v)", interval, maxAge)

		for {
			select {
			case <-ticker.C:
				cleaned := m.CleanupStaleGenerations(maxAge)
				if cleaned > 0 {
					log.Printf("Periodic cleanup: removed %d stale generation(s)", cleaned)
				}
			case <-m.cleanupStopCh:
				log.Printf("Stopping periodic cleanup of stale generations")
				return
			}
		}
	}()
}

// StopPeriodicCleanup stops the background cleanup routine
func (m *DefaultResponseGenerationManager) StopPeriodicCleanup() {
	close(m.cleanupStopCh)
	<-m.cleanupDone
	log.Printf("Periodic cleanup stopped")
}
