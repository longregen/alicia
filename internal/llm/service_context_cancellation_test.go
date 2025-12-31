package llm

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/ports"
)

// TestConvertStreamChunksContextCancellation verifies that convertStreamChunks
// responds to context cancellation and doesn't leak goroutines
func TestConvertStreamChunksContextCancellation(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	// Create a blocking input channel that will never close
	inputChan := make(chan StreamChunk)
	outputChan := make(chan ports.LLMStreamChunk, 10)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create a service instance
	service := &Service{}

	// Start the convertStreamChunks goroutine
	go service.convertStreamChunks(ctx, inputChan, outputChan)

	// Give the goroutine time to start
	time.Sleep(50 * time.Millisecond)

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	afterStart := runtime.NumGoroutine()
	t.Logf("After starting goroutine: %d goroutines", afterStart)

	// Verify the goroutine is running
	if afterStart <= baselineGoroutines {
		t.Logf("WARNING: Expected more goroutines after starting")
	}

	// Now cancel the context
	cancel()

	// Give the goroutine time to respond to cancellation
	time.Sleep(100 * time.Millisecond)

	// Verify the error was sent to the output channel
	select {
	case chunk := <-outputChan:
		if chunk.Error == nil {
			t.Errorf("Expected error chunk from context cancellation, got nil")
		} else if chunk.Error != context.Canceled {
			t.Errorf("Expected context.Canceled error, got: %v", chunk.Error)
		} else {
			t.Logf("Successfully received cancellation error: %v", chunk.Error)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timeout waiting for cancellation error chunk")
	}

	// Verify the output channel was closed
	select {
	case _, ok := <-outputChan:
		if ok {
			t.Errorf("Expected output channel to be closed")
		} else {
			t.Logf("Output channel correctly closed after cancellation")
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for channel closure")
	}

	// Give GC time to clean up
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	afterCancel := runtime.NumGoroutine()
	t.Logf("After context cancellation: %d goroutines", afterCancel)

	// Verify no goroutines leaked
	leaked := afterCancel - baselineGoroutines
	if leaked > 0 {
		t.Errorf("GOROUTINE LEAK: %d goroutine(s) leaked after context cancellation", leaked)
	} else {
		t.Logf("SUCCESS: No goroutines leaked, context cancellation works correctly")
	}

	// Clean up
	close(inputChan)
}

// TestConvertStreamChunksNormalClosure verifies that convertStreamChunks
// exits normally when the input channel closes
func TestConvertStreamChunksNormalClosure(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	// Create channels
	inputChan := make(chan StreamChunk, 5)
	outputChan := make(chan ports.LLMStreamChunk, 10)

	// Create a context
	ctx := context.Background()

	// Create a service instance
	service := &Service{}

	// Start the convertStreamChunks goroutine
	go service.convertStreamChunks(ctx, inputChan, outputChan)

	// Send some test data
	inputChan <- StreamChunk{Content: "test1"}
	inputChan <- StreamChunk{Content: "test2", Done: true}

	// Close the input channel normally
	close(inputChan)

	// Read the output
	chunks := []ports.LLMStreamChunk{}
	for chunk := range outputChan {
		chunks = append(chunks, chunk)
		if len(chunks) > 10 {
			t.Fatalf("Too many chunks received, possible infinite loop")
		}
	}

	// Verify we received the expected chunks
	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	// Give GC time to clean up
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	afterClose := runtime.NumGoroutine()
	t.Logf("After normal channel close: %d goroutines", afterClose)

	// Verify no goroutines leaked
	leaked := afterClose - baselineGoroutines
	if leaked > 0 {
		t.Errorf("GOROUTINE LEAK: %d goroutine(s) leaked after normal closure", leaked)
	} else {
		t.Logf("SUCCESS: No goroutines leaked on normal closure")
	}
}
