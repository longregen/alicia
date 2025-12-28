package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:11434/v1", "test-key", "e5-large", 1024)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("expected baseURL to be http://localhost:11434, got %s", client.baseURL)
	}

	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey to be test-key, got %s", client.apiKey)
	}

	if client.model != "e5-large" {
		t.Errorf("expected model to be e5-large, got %s", client.model)
	}

	if client.dimensions != 1024 {
		t.Errorf("expected dimensions to be 1024, got %d", client.dimensions)
	}
}

func TestGetDimensions(t *testing.T) {
	client := NewClient("http://localhost:11434/v1", "", "e5-large", 1024)

	if client.GetDimensions() != 1024 {
		t.Errorf("expected GetDimensions() to return 1024, got %d", client.GetDimensions())
	}
}

func TestNewClient_URLNormalization(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		expectedURL string
	}{
		{
			name:        "URL with /v1 suffix",
			inputURL:    "http://localhost:11434/v1",
			expectedURL: "http://localhost:11434",
		},
		{
			name:        "URL without /v1 suffix",
			inputURL:    "http://localhost:11434",
			expectedURL: "http://localhost:11434",
		},
		{
			name:        "URL with trailing slash",
			inputURL:    "http://localhost:11434/",
			expectedURL: "http://localhost:11434",
		},
		{
			name:        "URL with /v1/ suffix",
			inputURL:    "http://localhost:11434/v1/",
			expectedURL: "http://localhost:11434",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.inputURL, "", "test-model", 1024)
			if client.baseURL != tt.expectedURL {
				t.Errorf("expected baseURL to be %s, got %s", tt.expectedURL, client.baseURL)
			}
		})
	}
}

func TestEmbed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected authorization header")
		}

		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	result, err := client.Embed(context.Background(), "test text")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Embedding) != 3 {
		t.Errorf("expected 3 dimensions, got %d", len(result.Embedding))
	}
	if result.Model != "test-model" {
		t.Errorf("expected model test-model, got %s", result.Model)
	}
}

func TestEmbed_NoEmbeddingReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	_, err := client.Embed(context.Background(), "test text")

	if err == nil {
		t.Fatal("expected error for no embedding returned")
	}
}

func TestEmbedBatch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
				{
					Object:    "embedding",
					Embedding: []float32{0.4, 0.5, 0.6},
					Index:     1,
				},
			},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	results, err := client.EmbedBatch(context.Background(), []string{"text1", "text2"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Embedding[0] != 0.1 {
		t.Errorf("unexpected embedding value")
	}
}

func TestEmbedBatch_EmptyInput(t *testing.T) {
	client := NewClient("http://localhost:11434", "test-key", "test-model", 3)
	results, err := client.EmbedBatch(context.Background(), []string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestEmbedBatch_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	_, err := client.EmbedBatch(context.Background(), []string{"test"})

	if err == nil {
		t.Fatal("expected error for HTTP error")
	}
}

func TestEmbedBatch_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	_, err := client.EmbedBatch(context.Background(), []string{"test"})

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEmbedBatch_DimensionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2},
					Index:     0,
				},
			},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	_, err := client.EmbedBatch(context.Background(), []string{"test"})

	if err == nil {
		t.Fatal("expected error for dimension mismatch")
	}
}

func TestEmbedBatch_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	client.httpClient.Timeout = 100 * time.Millisecond

	_, err := client.EmbedBatch(context.Background(), []string{"test"})

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestEmbedBatch_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no authorization header")
		}
		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "test-model", 3)
	_, err := client.EmbedBatch(context.Background(), []string{"test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmbedBatch_SingleText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Single text should be sent as string, not array
		if _, ok := req.Input.([]interface{}); ok {
			t.Error("expected Input to be string for single text")
		}

		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)
	results, err := client.EmbedBatch(context.Background(), []string{"single text"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestEmbedBatch_NoDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Dimensions != 0 {
			t.Errorf("expected Dimensions to be 0, got %d", req.Dimensions)
		}

		resp := EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 0)
	results, err := client.EmbedBatch(context.Background(), []string{"test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Dimensions != 3 {
		t.Errorf("expected 3 dimensions, got %d", results[0].Dimensions)
	}
}

func TestEmbedBatch_CircuitBreaker(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 3)

	// Trigger circuit breaker by failing multiple times
	for i := 0; i < 6; i++ {
		client.EmbedBatch(context.Background(), []string{"test"})
	}

	// Circuit should be open now
	_, err := client.EmbedBatch(context.Background(), []string{"test"})
	if err == nil {
		t.Fatal("expected circuit breaker to be open")
	}
}
