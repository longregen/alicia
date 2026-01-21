package livekit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type ActiveGeneration struct {
	MessageID  string
	CancelFunc context.CancelFunc
	StartedAt  time.Time
}

type ResponseGenerationManager interface {
	RegisterGeneration(messageID string, cancelFunc context.CancelFunc)
	UnregisterGeneration(messageID string)
	CancelGeneration(targetID string) error
	CleanupStaleGenerations(maxAge time.Duration) int

	RegisterTTS(targetID string, cancelFunc context.CancelFunc)
	UnregisterTTS(targetID string)
	CancelTTS(targetID string) error

	CancelAll()
}

type DefaultResponseGenerationManager struct {
	activeGenerations map[string]*ActiveGeneration
	generationsMutex  sync.RWMutex

	activeTTS map[string]context.CancelFunc
	ttsMutex  sync.RWMutex

	cleanupStopCh chan struct{}
	cleanupDone   chan struct{}
}

func NewDefaultResponseGenerationManager() *DefaultResponseGenerationManager {
	return &DefaultResponseGenerationManager{
		activeGenerations: make(map[string]*ActiveGeneration),
		activeTTS:         make(map[string]context.CancelFunc),
		cleanupStopCh:     make(chan struct{}),
		cleanupDone:       make(chan struct{}),
	}
}

func (m *DefaultResponseGenerationManager) RegisterGeneration(messageID string, cancelFunc context.CancelFunc) {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	m.activeGenerations[messageID] = &ActiveGeneration{
		MessageID:  messageID,
		CancelFunc: cancelFunc,
		StartedAt:  time.Now(),
	}
}

func (m *DefaultResponseGenerationManager) UnregisterGeneration(messageID string) {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	delete(m.activeGenerations, messageID)
}

func (m *DefaultResponseGenerationManager) CancelGeneration(targetID string) error {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

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

func (m *DefaultResponseGenerationManager) CleanupStaleGenerations(maxAge time.Duration) int {
	m.generationsMutex.Lock()
	defer m.generationsMutex.Unlock()

	now := time.Now()
	cleanedCount := 0
	staleIDs := make([]string, 0)

	for id, gen := range m.activeGenerations {
		if now.Sub(gen.StartedAt) > maxAge {
			staleIDs = append(staleIDs, id)
		}
	}

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

func (m *DefaultResponseGenerationManager) RegisterTTS(targetID string, cancelFunc context.CancelFunc) {
	m.ttsMutex.Lock()
	defer m.ttsMutex.Unlock()

	m.activeTTS[targetID] = cancelFunc
}

func (m *DefaultResponseGenerationManager) UnregisterTTS(targetID string) {
	m.ttsMutex.Lock()
	defer m.ttsMutex.Unlock()

	delete(m.activeTTS, targetID)
}

func (m *DefaultResponseGenerationManager) CancelTTS(targetID string) error {
	m.ttsMutex.Lock()
	defer m.ttsMutex.Unlock()

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

func (m *DefaultResponseGenerationManager) CancelAll() {
	_ = m.CancelGeneration("")
	_ = m.CancelTTS("")
}

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

func (m *DefaultResponseGenerationManager) StopPeriodicCleanup() {
	close(m.cleanupStopCh)
	<-m.cleanupDone
	log.Printf("Periodic cleanup stopped")
}
