package services

import (
	"sync"
	"time"

	"github.com/longregen/alicia/internal/ports"
)

// OptimizationProgressPublisher manages subscriptions and publishing of optimization progress events
// This separates the pub/sub infrastructure concern from the optimization business logic
type OptimizationProgressPublisher struct {
	channels map[string][]chan ports.OptimizationProgressEvent
	mu       sync.RWMutex

	// Optional broadcaster for WebSocket delivery
	wsBroadcaster ports.OptimizationProgressBroadcaster
}

// Compile-time interface check
var _ ports.OptimizationProgressPublisher = (*OptimizationProgressPublisher)(nil)

// NewOptimizationProgressPublisher creates a new progress publisher
// The wsBroadcaster parameter is optional - pass nil if WebSocket broadcasting is not needed
func NewOptimizationProgressPublisher(wsBroadcaster ports.OptimizationProgressBroadcaster) *OptimizationProgressPublisher {
	return &OptimizationProgressPublisher{
		channels:      make(map[string][]chan ports.OptimizationProgressEvent),
		wsBroadcaster: wsBroadcaster,
	}
}

// Subscribe creates a new channel for receiving progress events for a run
// The returned channel is buffered to prevent blocking the publisher
func (p *OptimizationProgressPublisher) Subscribe(runID string) <-chan ports.OptimizationProgressEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create a buffered channel to prevent blocking the publisher
	ch := make(chan ports.OptimizationProgressEvent, 100)
	p.channels[runID] = append(p.channels[runID], ch)
	return ch
}

// Unsubscribe removes a channel from receiving progress events
// The channel will be closed after removal
func (p *OptimizationProgressPublisher) Unsubscribe(runID string, ch <-chan ports.OptimizationProgressEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	channels := p.channels[runID]
	for i, subscriberCh := range channels {
		if subscriberCh == ch {
			// Remove this channel from the slice
			p.channels[runID] = append(channels[:i], channels[i+1:]...)
			close(subscriberCh)
			break
		}
	}

	// Clean up the map entry if no more subscribers
	if len(p.channels[runID]) == 0 {
		delete(p.channels, runID)
	}
}

// PublishProgress sends a progress event to all subscribers and broadcasts via WebSocket
// Publishing is non-blocking - if a subscriber's buffer is full, the event is dropped for that subscriber
func (p *OptimizationProgressPublisher) PublishProgress(event ports.OptimizationProgressEvent) {
	// Broadcast via WebSocket if broadcaster is available
	if p.wsBroadcaster != nil {
		// Parse timestamp string to Unix milliseconds for the update format
		timestamp, _ := time.Parse(time.RFC3339, event.Timestamp)
		update := ports.OptimizationProgressUpdate{
			RunID:           event.RunID,
			Status:          event.Status,
			Iteration:       event.Iteration,
			MaxIterations:   event.MaxIterations,
			CurrentScore:    event.CurrentScore,
			BestScore:       event.BestScore,
			DimensionScores: event.DimensionScores,
			Message:         event.Message,
			Timestamp:       timestamp.UnixMilli(),
		}
		p.wsBroadcaster.BroadcastOptimizationProgress(event.RunID, update)
	}

	// Publish to SSE channels for backward compatibility
	p.mu.RLock()
	defer p.mu.RUnlock()

	channels := p.channels[event.RunID]
	for _, ch := range channels {
		// Non-blocking send to prevent slow subscribers from blocking the publisher
		select {
		case ch <- event:
		default:
			// Channel buffer is full, skip this update for this subscriber
			// This prevents one slow consumer from affecting others
		}
	}
}

// Close closes all channels for a run (called when optimization completes)
// All subscribers will receive channel closure and should handle it appropriately
func (p *OptimizationProgressPublisher) Close(runID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	channels := p.channels[runID]
	for _, ch := range channels {
		close(ch)
	}
	delete(p.channels, runID)
}

// SubscriberCount returns the number of active subscribers for a run
// This is useful for monitoring and debugging
func (p *OptimizationProgressPublisher) SubscriberCount(runID string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.channels[runID])
}

// ActiveRuns returns a list of run IDs that have active subscribers
// This is useful for monitoring and debugging
func (p *OptimizationProgressPublisher) ActiveRuns() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	runs := make([]string, 0, len(p.channels))
	for runID := range p.channels {
		runs = append(runs, runID)
	}
	return runs
}
