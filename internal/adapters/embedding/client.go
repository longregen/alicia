package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/adapters/circuitbreaker"
	"github.com/longregen/alicia/internal/adapters/retry"
	"github.com/longregen/alicia/internal/ports"
)

const (
	// EmbeddingTimeout is the maximum time to wait for embedding generation
	EmbeddingTimeout = 30 * time.Second
)

// Client is an OpenAI-compatible embedding client
type Client struct {
	baseURL     string
	apiKey      string
	model       string
	dimensions  int
	httpClient  *http.Client
	retryConfig retry.BackoffConfig
	breaker     *circuitbreaker.CircuitBreaker
}

// NewClient creates a new embedding client
func NewClient(baseURL, apiKey, model string, dimensions int) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Default request timeout
		},
		retryConfig: retry.HTTPConfig(),
		breaker:     circuitbreaker.New(5, 30*time.Second), // 5 failures, 30s timeout
	}
}

// EmbeddingRequest represents the request to the embeddings API
type EmbeddingRequest struct {
	Input interface{} `json:"input"` // Can be string or []string
	Model string      `json:"model"`
}

// EmbeddingResponse represents the response from the embeddings API
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed generates an embedding for a single text
func (c *Client) Embed(ctx context.Context, text string) (*ports.EmbeddingResult, error) {
	var result *ports.EmbeddingResult
	err := c.breaker.Execute(func() error {
		// Add timeout to prevent hanging on slow/failed embedding requests
		ctx, cancel := context.WithTimeout(ctx, EmbeddingTimeout)
		defer cancel()

		results, err := c.embedBatchInternal(ctx, []string{text})
		if err != nil {
			log.Printf("[EmbeddingClient.Embed] embedBatchInternal failed: baseURL=%s, model=%s, textLen=%d, error=%v",
				c.baseURL, c.model, len(text), err)
			return err
		}
		if len(results) == 0 {
			log.Printf("[EmbeddingClient.Embed] no embedding returned: baseURL=%s, model=%s", c.baseURL, c.model)
			return fmt.Errorf("no embedding returned")
		}
		result = results[0]
		return nil
	})
	if err != nil {
		log.Printf("[EmbeddingClient.Embed] circuit breaker error: %v (state may be open)", err)
	}
	return result, err
}

// EmbedBatch generates embeddings for multiple texts
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([]*ports.EmbeddingResult, error) {
	if len(texts) == 0 {
		return []*ports.EmbeddingResult{}, nil
	}

	var results []*ports.EmbeddingResult
	err := c.breaker.Execute(func() error {
		// Add timeout to prevent hanging on slow/failed embedding requests
		ctx, cancel := context.WithTimeout(ctx, EmbeddingTimeout)
		defer cancel()

		var err error
		results, err = c.embedBatchInternal(ctx, texts)
		return err
	})
	return results, err
}

// GetDimensions returns the dimensionality of the embeddings
func (c *Client) GetDimensions() int {
	return c.dimensions
}

// curlExample returns a curl command for debugging embedding requests
func (c *Client) curlExample() string {
	authHeader := ""
	if c.apiKey != "" {
		authHeader = fmt.Sprintf(` -H "Authorization: Bearer %s"`, c.apiKey)
	}
	return fmt.Sprintf(
		`curl -X POST "%s/v1/embeddings" -H "Content-Type: application/json"%s -d '{"input": "test", "model": "%s"}'`,
		c.baseURL, authHeader, c.model,
	)
}

// embedBatchInternal is the internal implementation for batch embedding
func (c *Client) embedBatchInternal(ctx context.Context, texts []string) ([]*ports.EmbeddingResult, error) {
	// Prepare request
	req := EmbeddingRequest{
		Model: c.model,
	}

	// Handle single vs multiple texts
	if len(texts) == 1 {
		req.Input = texts[0]
	} else {
		req.Input = texts
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var respBody []byte
	var statusCode int

	err = retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/embeddings", bytes.NewReader(body))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			log.Printf("[EmbeddingClient] HTTP request failed: url=%s/v1/embeddings, error=%v", c.baseURL, err)
			return 0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return statusCode, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[EmbeddingClient] API error: url=%s/v1/embeddings, status=%d, body=%s", c.baseURL, resp.StatusCode, string(respBody))
			return statusCode, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
		}

		return statusCode, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w (debug: %s)", err, c.curlExample())
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to EmbeddingResult
	results := make([]*ports.EmbeddingResult, len(embeddingResp.Data))
	for _, data := range embeddingResp.Data {
		dimensions := len(data.Embedding)
		if c.dimensions > 0 && dimensions != c.dimensions {
			log.Printf("[EmbeddingClient] dimension mismatch: expected=%d, got=%d, model=%s", c.dimensions, dimensions, embeddingResp.Model)
			return nil, fmt.Errorf("expected %d dimensions but got %d", c.dimensions, dimensions)
		}

		results[data.Index] = &ports.EmbeddingResult{
			Embedding:  data.Embedding,
			Model:      embeddingResp.Model,
			Dimensions: dimensions,
		}
	}

	return results, nil
}
