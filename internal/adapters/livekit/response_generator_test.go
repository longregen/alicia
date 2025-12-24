package livekit

import (
	"context"
	"testing"
	"time"
)

func TestResponseGenerationManager_RegisterAndUnregister(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.RegisterGeneration("msg_1", cancel)

	// Check that generation is registered
	mgr.generationsMutex.RLock()
	if _, exists := mgr.activeGenerations["msg_1"]; !exists {
		t.Error("expected generation to be registered")
	}
	mgr.generationsMutex.RUnlock()

	mgr.UnregisterGeneration("msg_1")

	// Check that generation is unregistered
	mgr.generationsMutex.RLock()
	if _, exists := mgr.activeGenerations["msg_1"]; exists {
		t.Error("expected generation to be unregistered")
	}
	mgr.generationsMutex.RUnlock()
}

func TestResponseGenerationManager_CancelSpecificGeneration(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	ctx, cancel := context.WithCancel(context.Background())

	mgr.RegisterGeneration("msg_1", cancel)

	err := mgr.CancelGeneration("msg_1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should be unregistered after cancel
	mgr.generationsMutex.RLock()
	if _, exists := mgr.activeGenerations["msg_1"]; exists {
		t.Error("expected generation to be removed after cancel")
	}
	mgr.generationsMutex.RUnlock()

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("expected context to be cancelled")
	}
}

func TestResponseGenerationManager_CancelNonExistent(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	err := mgr.CancelGeneration("non_existent")
	if err == nil {
		t.Error("expected error when cancelling non-existent generation")
	}
}

func TestResponseGenerationManager_CancelAllGenerations(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	mgr.RegisterGeneration("msg_1", cancel1)
	mgr.RegisterGeneration("msg_2", cancel2)

	// Cancel all by passing empty targetID
	err := mgr.CancelGeneration("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// All should be unregistered
	mgr.generationsMutex.RLock()
	if len(mgr.activeGenerations) != 0 {
		t.Errorf("expected all generations to be removed, found %d", len(mgr.activeGenerations))
	}
	mgr.generationsMutex.RUnlock()

	// Contexts should be cancelled
	select {
	case <-ctx1.Done():
		// Expected
	default:
		t.Error("expected ctx1 to be cancelled")
	}

	select {
	case <-ctx2.Done():
		// Expected
	default:
		t.Error("expected ctx2 to be cancelled")
	}
}

func TestResponseGenerationManager_CancelAllWhenEmpty(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	// Should not error when cancelling all with no active generations
	err := mgr.CancelGeneration("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResponseGenerationManager_CleanupStaleGenerations(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	ctx1, cancel1 := context.WithCancel(context.Background())
	_, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	// Register one old generation
	mgr.generationsMutex.Lock()
	mgr.activeGenerations["msg_old"] = &ActiveGeneration{
		MessageID:  "msg_old",
		CancelFunc: cancel1,
		StartedAt:  time.Now().Add(-10 * time.Minute),
	}
	mgr.generationsMutex.Unlock()

	// Register one recent generation
	mgr.RegisterGeneration("msg_new", cancel2)

	// Cleanup with 5 minute threshold
	cleaned := mgr.CleanupStaleGenerations(5 * time.Minute)

	if cleaned != 1 {
		t.Errorf("expected 1 stale generation cleaned, got %d", cleaned)
	}

	// Old generation should be removed
	mgr.generationsMutex.RLock()
	if _, exists := mgr.activeGenerations["msg_old"]; exists {
		t.Error("expected old generation to be removed")
	}

	// New generation should still exist
	if _, exists := mgr.activeGenerations["msg_new"]; !exists {
		t.Error("expected new generation to still exist")
	}
	mgr.generationsMutex.RUnlock()

	// Old context should be cancelled
	select {
	case <-ctx1.Done():
		// Expected
	default:
		t.Error("expected old context to be cancelled")
	}
}

func TestResponseGenerationManager_CleanupNothingStale(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.RegisterGeneration("msg_1", cancel)

	// Cleanup with very high threshold
	cleaned := mgr.CleanupStaleGenerations(24 * time.Hour)

	if cleaned != 0 {
		t.Errorf("expected 0 stale generations cleaned, got %d", cleaned)
	}

	// Generation should still exist
	mgr.generationsMutex.RLock()
	if _, exists := mgr.activeGenerations["msg_1"]; !exists {
		t.Error("expected generation to still exist")
	}
	mgr.generationsMutex.RUnlock()
}

func TestResponseGenerationManager_TTS(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	_, cancel := context.WithCancel(context.Background())

	mgr.RegisterTTS("target_1", cancel)

	// Check that TTS is registered
	mgr.ttsMutex.RLock()
	if _, exists := mgr.activeTTS["target_1"]; !exists {
		t.Error("expected TTS to be registered")
	}
	mgr.ttsMutex.RUnlock()

	mgr.UnregisterTTS("target_1")

	// Check that TTS is unregistered
	mgr.ttsMutex.RLock()
	if _, exists := mgr.activeTTS["target_1"]; exists {
		t.Error("expected TTS to be unregistered")
	}
	mgr.ttsMutex.RUnlock()
}

func TestResponseGenerationManager_CancelTTS(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	ctx, cancel := context.WithCancel(context.Background())

	mgr.RegisterTTS("target_1", cancel)

	err := mgr.CancelTTS("target_1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should be unregistered after cancel
	mgr.ttsMutex.RLock()
	if _, exists := mgr.activeTTS["target_1"]; exists {
		t.Error("expected TTS to be removed after cancel")
	}
	mgr.ttsMutex.RUnlock()

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("expected context to be cancelled")
	}
}

func TestResponseGenerationManager_CancelTTSNonExistent(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	err := mgr.CancelTTS("non_existent")
	if err == nil {
		t.Error("expected error when cancelling non-existent TTS")
	}
}

func TestResponseGenerationManager_CancelAllTTS(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	mgr.RegisterTTS("target_1", cancel1)
	mgr.RegisterTTS("target_2", cancel2)

	// Cancel all by passing empty targetID
	err := mgr.CancelTTS("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// All should be unregistered
	mgr.ttsMutex.RLock()
	if len(mgr.activeTTS) != 0 {
		t.Errorf("expected all TTS to be removed, found %d", len(mgr.activeTTS))
	}
	mgr.ttsMutex.RUnlock()

	// Contexts should be cancelled
	select {
	case <-ctx1.Done():
		// Expected
	default:
		t.Error("expected ctx1 to be cancelled")
	}

	select {
	case <-ctx2.Done():
		// Expected
	default:
		t.Error("expected ctx2 to be cancelled")
	}
}

func TestResponseGenerationManager_CancelAllTTSWhenEmpty(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	// Should not error when cancelling all with no active TTS
	err := mgr.CancelTTS("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResponseGenerationManager_Concurrent(t *testing.T) {
	mgr := NewDefaultResponseGenerationManager()

	done := make(chan bool)

	// Register and unregister concurrently
	go func() {
		for i := 0; i < 100; i++ {
			_, cancel := context.WithCancel(context.Background())
			mgr.RegisterGeneration("msg_1", cancel)
			mgr.UnregisterGeneration("msg_1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_, cancel := context.WithCancel(context.Background())
			mgr.RegisterTTS("target_1", cancel)
			mgr.UnregisterTTS("target_1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			mgr.CleanupStaleGenerations(1 * time.Hour)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done
}
