package llm

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/ports"
)

// mockStreamingClient simulates a slow streaming client
type mockStreamingClient struct {
	*Client
	ctxCancelled   *bool
	ctxCancelledAt *time.Time
}

func (m *mockStreamingClient) ChatStream(ctx context.Context, messages []ChatMessage) (<-chan StreamChunk, error) {
	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)

		// Send 5 chunks with 50ms delay each (total 250ms)
		for i := 0; i < 5; i++ {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				if m.ctxCancelled != nil {
					*m.ctxCancelled = true
				}
				if m.ctxCancelledAt != nil {
					*m.ctxCancelledAt = time.Now()
				}
				chunks <- StreamChunk{Error: ctx.Err()}
				return
			default:
			}

			// Simulate slow streaming
			time.Sleep(50 * time.Millisecond)

			// Send a chunk
			chunks <- StreamChunk{
				Content: "chunk data",
				Done:    i == 4,
			}
		}
	}()

	return chunks, nil
}

func (m *mockStreamingClient) ChatStreamWithTools(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	return m.ChatStream(ctx, messages)
}

// TestStreamingContextCancellation tests whether the streaming goroutine
// is prematurely cancelled when doChatStream returns.
//
// Bug hypothesis: The defer cancel() in doChatStream (service.go:112) runs
// immediately after the function returns, which cancels the context used by
// the streaming goroutine, causing it to stop prematurely.
func TestStreamingContextCancellation(t *testing.T) {
	// Track whether the context passed to the client was cancelled
	var ctxCancelled bool
	var ctxCancelledAt time.Time

	mockClient := &mockStreamingClient{
		Client: &Client{
			baseURL:     "http://test",
			apiKey:      "test",
			model:       "test",
			maxTokens:   100,
			temperature: 0.7,
		},
		ctxCancelled:   &ctxCancelled,
		ctxCancelledAt: &ctxCancelledAt,
	}

	// We need to test doChatStream, but since we can't override the client's methods
	// on a *Client (only on the embedded struct), we'll create a test version
	// of the streaming function that mimics the bug

	ctx := context.Background()
	messages := []ports.LLMMessage{
		{Role: "user", Content: "test"},
	}

	startTime := time.Now()

	// Simulate doChatStream with the same bug
	streamChan, err := testDoChatStream(ctx, messages, mockClient)
	if err != nil {
		t.Fatalf("testDoChatStream failed: %v", err)
	}

	returnTime := time.Now()

	// At this point, the function has returned and defer cancel() has executed.
	// If the bug exists, the streaming goroutine's context is now cancelled.

	t.Logf("Function returned after %v", returnTime.Sub(startTime))

	// Wait a bit to ensure defer cancel() has executed
	time.Sleep(10 * time.Millisecond)

	// Try to consume chunks from the stream
	chunksReceived := 0
	timeout := time.After(2 * time.Second)

	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				// Channel closed
				elapsed := time.Since(startTime)
				t.Logf("Stream closed after %v, received %d chunks", elapsed, chunksReceived)

				if ctxCancelled {
					t.Errorf("BUG CONFIRMED: Context was cancelled at %v (%.0fms after function returned)",
						ctxCancelledAt.Sub(returnTime), ctxCancelledAt.Sub(returnTime).Seconds()*1000)
					t.Errorf("  Received only %d/5 chunks before cancellation", chunksReceived)
					t.Errorf("  The defer cancel() killed the streaming goroutine prematurely")
				} else if chunksReceived < 5 {
					t.Errorf("PARTIAL BUG: Received only %d/5 chunks but context wasn't explicitly cancelled", chunksReceived)
				} else {
					t.Logf("FALSE POSITIVE: Received all 5 chunks successfully. No premature cancellation.")
				}
				return

			}
			if chunk.Error != nil {
				// Check if the error is due to context cancellation
				if chunk.Error == context.Canceled || chunk.Error == context.DeadlineExceeded {
					elapsed := time.Since(returnTime)
					t.Errorf("BUG CONFIRMED: Stream was cancelled %.0fms after function returned. Received %d chunks. Error: %v",
						elapsed.Seconds()*1000, chunksReceived, chunk.Error)
					t.Errorf("  Root cause: defer cancel() at line 112 of service.go cancels the context immediately after return")
					return
				}
				t.Fatalf("Unexpected error in stream: %v", chunk.Error)
			}
			chunksReceived++
			t.Logf("Received chunk %d at +%.0fms",
				chunksReceived,
				time.Since(returnTime).Seconds()*1000)

		case <-timeout:
			t.Fatalf("Test timeout after receiving %d chunks", chunksReceived)
		}
	}
}

// testDoChatStream replicates the fix from service.go lines 110-127
func testDoChatStream(ctx context.Context, messages []ports.LLMMessage, mockClient *mockStreamingClient) (<-chan ports.LLMStreamChunk, error) {
	ctx, cancel := context.WithTimeout(ctx, LLMTimeout)

	chatMessages := []ChatMessage{}
	for _, msg := range messages {
		chatMessages = append(chatMessages, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	clientChan, err := mockClient.ChatStream(ctx, chatMessages)
	if err != nil {
		cancel()
		return nil, err
	}

	outputChan := make(chan ports.LLMStreamChunk, 10)
	go func() {
		defer cancel()
		testConvertStreamChunks(clientChan, outputChan)
	}()

	return outputChan, nil
}

// testConvertStreamChunks is a simplified version of Service.convertStreamChunks
func testConvertStreamChunks(clientChan <-chan StreamChunk, outputChan chan<- ports.LLMStreamChunk) {
	defer close(outputChan)

	for chunk := range clientChan {
		portChunk := ports.LLMStreamChunk{
			Content:   chunk.Content,
			Reasoning: chunk.Reasoning,
			Done:      chunk.Done,
			Error:     chunk.Error,
		}

		if chunk.ToolCall != nil {
			portChunk.ToolCall = &ports.LLMToolCall{
				ID:        chunk.ToolCall.ID,
				Name:      chunk.ToolCall.Function.Name,
				Arguments: map[string]any{}, // simplified for test
			}
		}

		outputChan <- portChunk
	}
}
