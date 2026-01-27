package langfuse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var discardLogger = log.New(io.Discard, "", 0)

type Client struct {
	host       string
	publicKey  string
	secretKey  string
	httpClient *http.Client
	cache      *promptCache
	log        *log.Logger
}

type Prompt struct {
	Name    string      `json:"name"`
	Version int         `json:"version"`
	Prompt  interface{} `json:"prompt"`
	Labels  []string    `json:"labels"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type promptResponse struct {
	Name    string          `json:"name"`
	Version int             `json:"version"`
	Prompt  json.RawMessage `json:"prompt"`
	Labels  []string        `json:"labels"`
	Type    string          `json:"type"`
}

func New(host, publicKey, secretKey string) *Client {
	if host == "" {
		host = os.Getenv("LANGFUSE_HOST")
	}
	if host == "" {
		host = "langfuse.hjkl.lol"
	}

	if publicKey == "" {
		publicKey = os.Getenv("LANGFUSE_PUBLIC_KEY")
	}
	if secretKey == "" {
		secretKey = os.Getenv("LANGFUSE_SECRET_KEY")
	}

	logger := discardLogger
	if v := os.Getenv("LANGFUSE_VERBOSE"); v == "1" || v == "true" {
		logger = log.Default()
	}

	c := &Client{
		host:      strings.TrimSuffix(host, "/"),
		publicKey: publicKey,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: logger,
	}
	c.cache = newPromptCache(c)
	return c
}

func (c *Client) GetPrompt(name string, opts ...Option) (*Prompt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.GetPromptContext(ctx, name, opts...)
}

func (c *Client) GetPromptContext(ctx context.Context, name string, opts ...Option) (*Prompt, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(cfg)
	}

	cacheKey := c.buildCacheKey(name, cfg)
	if prompt, ok := c.cache.get(cacheKey); ok {
		c.cache.triggerRefreshIfStale(cacheKey, name, cfg)
		return prompt, nil
	}

	prompt, err := c.fetchPrompt(ctx, name, cfg)
	if err != nil {
		if fallback, ok := getFallbackPrompt(name); ok {
			return fallback, nil
		}
		return nil, err
	}

	c.cache.set(cacheKey, prompt)
	return prompt, nil
}

func (c *Client) fetchPrompt(ctx context.Context, name string, cfg *options) (*Prompt, error) {
	escapedName := url.PathEscape(name)
	apiURL := fmt.Sprintf("https://%s/api/public/v2/prompts/%s", c.host, escapedName)

	var params []string
	if cfg.label != "" {
		params = append(params, fmt.Sprintf("label=%s", url.QueryEscape(cfg.label)))
	}
	if cfg.version > 0 {
		params = append(params, fmt.Sprintf("version=%d", cfg.version))
	}
	if len(params) > 0 {
		apiURL += "?" + strings.Join(params, "&")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("langfuse: API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to read response: %w", err)
	}

	var apiResp promptResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("langfuse: failed to parse response: %w", err)
	}

	prompt := &Prompt{
		Name:    apiResp.Name,
		Version: apiResp.Version,
		Labels:  apiResp.Labels,
	}

	if apiResp.Type == "chat" {
		var messages []ChatMessage
		if err := json.Unmarshal(apiResp.Prompt, &messages); err != nil {
			return nil, fmt.Errorf("langfuse: failed to parse chat prompt: %w", err)
		}
		prompt.Prompt = messages
	} else {
		var text string
		if err := json.Unmarshal(apiResp.Prompt, &text); err != nil {
			return nil, fmt.Errorf("langfuse: failed to parse text prompt: %w", err)
		}
		prompt.Prompt = text
	}

	return prompt, nil
}

func (c *Client) buildCacheKey(name string, cfg *options) string {
	key := name
	if cfg.label != "" {
		key += ":label=" + cfg.label
	}
	if cfg.version > 0 {
		key += fmt.Sprintf(":version=%d", cfg.version)
	}
	return key
}

func (p *Prompt) Compile(vars map[string]string) string {
	text, ok := p.Prompt.(string)
	if !ok {
		return ""
	}
	return CompileTemplate(text, vars)
}

func (p *Prompt) CompileChat(vars map[string]string) []ChatMessage {
	messages, ok := p.Prompt.([]ChatMessage)
	if !ok {
		return nil
	}

	result := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		result[i] = ChatMessage{
			Role:    msg.Role,
			Content: CompileTemplate(msg.Content, vars),
		}
	}
	return result
}

func (p *Prompt) GetText() string {
	text, _ := p.Prompt.(string)
	return text
}

var templateVarRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// CompileTemplate replaces {{variable}} placeholders in a template with values from vars.
func CompileTemplate(template string, vars map[string]string) string {
	return templateVarRegex.ReplaceAllStringFunc(template, func(match string) string {
		varName := match[2 : len(match)-2]
		if value, ok := vars[varName]; ok {
			return value
		}
		return match
	})
}

// TruncateString truncates s to maxLen characters, appending suffix if truncated.
func TruncateString(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + suffix
}

// TraceParams holds parameters for creating a trace in Langfuse.
type TraceParams struct {
	ID        string         // Required: trace ID (use OTel trace ID for correlation)
	Name      string         // Optional: name identifying the trace type
	SessionID string         // Optional: groups traces into a session
	UserID    string         // Optional: identifies the end user
	Metadata  map[string]any // Optional: arbitrary key-value pairs
	Tags      []string       // Optional: tags for filtering
}

// ingestionEvent is a single event in the Langfuse ingestion batch.
type ingestionEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Body      any       `json:"body"`
}

// traceBody is the body of a trace-create ingestion event.
type traceBody struct {
	ID        string         `json:"id"`
	Name      string         `json:"name,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
	UserID    string         `json:"userId,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// CreateTrace creates a trace in Langfuse via the ingestion API.
// This ensures scores referencing the trace ID will inherit session/user context.
func (c *Client) CreateTrace(ctx context.Context, params TraceParams) error {
	if params.ID == "" {
		return fmt.Errorf("langfuse: trace ID is required")
	}

	now := time.Now().UTC()
	event := ingestionEvent{
		ID:        fmt.Sprintf("evt-%s-%d", params.ID[:8], now.UnixMilli()),
		Type:      "trace-create",
		Timestamp: now,
		Body: traceBody{
			ID:        params.ID,
			Name:      params.Name,
			SessionID: params.SessionID,
			UserID:    params.UserID,
			Metadata:  params.Metadata,
			Tags:      params.Tags,
			Timestamp: now,
		},
	}

	batch := map[string]any{"batch": []ingestionEvent{event}}
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal trace: %w", err)
	}

	apiURL := fmt.Sprintf("https://%s/api/public/ingestion", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: trace creation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Ingestion API returns 207 for partial success, 200/201 for full success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMultiStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: trace ingestion failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Printf("langfuse: created trace %s (name=%s, session=%s, user=%s)", params.ID, params.Name, params.SessionID, params.UserID)
	return nil
}

// SpanParams holds parameters for creating a span observation in Langfuse.
type SpanParams struct {
	TraceID             string         // Required: trace ID to associate with
	ID                  string         // Required: unique span ID
	ParentObservationID string         // Optional: parent span ID for nesting
	Name                string         // Optional: name for the span
	Input               any            // Optional: input data
	Output              any            // Optional: output data
	Metadata            map[string]any // Optional: arbitrary key-value pairs
	StartTime           time.Time      // Optional: when the span started
	EndTime             time.Time      // Optional: when the span ended
}

// spanBody is the body of a span-create ingestion event.
type spanBody struct {
	TraceID             string         `json:"traceId"`
	ID                  string         `json:"id"`
	ParentObservationID string         `json:"parentObservationId,omitempty"`
	Name                string         `json:"name,omitempty"`
	Input               any            `json:"input,omitempty"`
	Output              any            `json:"output,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
	StartTime           time.Time      `json:"startTime"`
	EndTime             time.Time      `json:"endTime,omitempty"`
}

// GenerationParams holds parameters for creating a generation observation in Langfuse.
type GenerationParams struct {
	TraceID             string    // Required: trace ID to associate with
	ID                  string    // Required: unique generation ID (use span ID)
	ParentObservationID string    // Optional: parent span ID for nesting
	Name                string    // Optional: name for the generation (e.g., "llm.chat")
	Model               string    // Optional: model name
	PromptName          string    // Required for linking: Langfuse prompt name
	PromptVersion       int       // Required for linking: Langfuse prompt version
	Input               any       // Optional: input messages
	Output              any       // Optional: output content
	StartTime           time.Time // Optional: when the generation started
	EndTime             time.Time // Optional: when the generation ended
	PromptTokens        int       // Optional: prompt token count
	CompletionTokens    int       // Optional: completion token count
	TotalTokens         int       // Optional: total token count
	ModelParameters     map[string]any // Optional: model config (temperature, max_tokens, etc.)
	Metadata            map[string]any // Optional: arbitrary metadata (tools, streaming, etc.)
}

// generationUsage holds token usage data for a generation.
type generationUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// generationBody is the body of a generation-create ingestion event.
type generationBody struct {
	TraceID             string           `json:"traceId"`
	ID                  string           `json:"id"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Name                string           `json:"name,omitempty"`
	Model               string           `json:"model,omitempty"`
	PromptName          string           `json:"promptName,omitempty"`
	PromptVersion       int              `json:"promptVersion,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Usage               *generationUsage `json:"usage,omitempty"`
	ModelParameters     map[string]any   `json:"modelParameters,omitempty"`
	Metadata            map[string]any   `json:"metadata,omitempty"`
	StartTime           time.Time        `json:"startTime"`
	EndTime             time.Time        `json:"endTime,omitempty"`
}

// CreateSpan creates a span observation in Langfuse via the ingestion API.
// Spans group related observations (generations, other spans) under a parent.
func (c *Client) CreateSpan(ctx context.Context, params SpanParams) error {
	if params.TraceID == "" {
		return fmt.Errorf("langfuse: traceID is required")
	}
	if params.ID == "" {
		return fmt.Errorf("langfuse: span ID is required")
	}

	now := time.Now().UTC()
	if params.StartTime.IsZero() {
		params.StartTime = now
	}

	event := ingestionEvent{
		ID:        fmt.Sprintf("evt-span-%s-%d", params.ID[:8], now.UnixMilli()),
		Type:      "span-create",
		Timestamp: now,
		Body: spanBody{
			TraceID:             params.TraceID,
			ID:                  params.ID,
			ParentObservationID: params.ParentObservationID,
			Name:                params.Name,
			Input:               params.Input,
			Output:              params.Output,
			Metadata:            params.Metadata,
			StartTime:           params.StartTime,
			EndTime:             params.EndTime,
		},
	}

	batch := map[string]any{"batch": []ingestionEvent{event}}
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal span: %w", err)
	}

	apiURL := fmt.Sprintf("https://%s/api/public/ingestion", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: span creation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMultiStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: span ingestion failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Printf("langfuse: created span %s (name=%s, parent=%s)", params.ID, params.Name, params.ParentObservationID)
	return nil
}

// UpdateSpan updates an existing span observation in Langfuse (e.g., to set EndTime).
func (c *Client) UpdateSpan(ctx context.Context, params SpanParams) error {
	if params.ID == "" {
		return fmt.Errorf("langfuse: span ID is required")
	}

	now := time.Now().UTC()

	event := ingestionEvent{
		ID:        fmt.Sprintf("evt-span-upd-%s-%d", params.ID[:8], now.UnixMilli()),
		Type:      "span-update",
		Timestamp: now,
		Body: spanBody{
			TraceID:             params.TraceID,
			ID:                  params.ID,
			ParentObservationID: params.ParentObservationID,
			Name:                params.Name,
			Output:              params.Output,
			Metadata:            params.Metadata,
			EndTime:             params.EndTime,
		},
	}

	batch := map[string]any{"batch": []ingestionEvent{event}}
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal span update: %w", err)
	}

	apiURL := fmt.Sprintf("https://%s/api/public/ingestion", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: span update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMultiStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: span update failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CreateGeneration creates a generation observation in Langfuse.
// When promptName and promptVersion are set, Langfuse links this generation to the prompt.
func (c *Client) CreateGeneration(ctx context.Context, params GenerationParams) error {
	if params.TraceID == "" {
		return fmt.Errorf("langfuse: traceID is required")
	}
	if params.ID == "" {
		return fmt.Errorf("langfuse: generation ID is required")
	}

	now := time.Now().UTC()
	if params.StartTime.IsZero() {
		params.StartTime = now
	}

	genBody := generationBody{
		TraceID:             params.TraceID,
		ID:                  params.ID,
		ParentObservationID: params.ParentObservationID,
		Name:                params.Name,
		Model:               params.Model,
		PromptName:          params.PromptName,
		PromptVersion:       params.PromptVersion,
		Input:               params.Input,
		Output:              params.Output,
		StartTime:           params.StartTime,
		EndTime:             params.EndTime,
	}
	if params.PromptTokens > 0 || params.CompletionTokens > 0 || params.TotalTokens > 0 {
		genBody.Usage = &generationUsage{
			PromptTokens:     params.PromptTokens,
			CompletionTokens: params.CompletionTokens,
			TotalTokens:      params.TotalTokens,
		}
	}
	if len(params.ModelParameters) > 0 {
		genBody.ModelParameters = params.ModelParameters
	}
	if len(params.Metadata) > 0 {
		genBody.Metadata = params.Metadata
	}

	event := ingestionEvent{
		ID:        fmt.Sprintf("evt-gen-%s-%d", params.ID[:8], now.UnixMilli()),
		Type:      "generation-create",
		Timestamp: now,
		Body:      genBody,
	}

	batch := map[string]any{"batch": []ingestionEvent{event}}
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal generation: %w", err)
	}

	apiURL := fmt.Sprintf("https://%s/api/public/ingestion", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: generation creation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMultiStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: generation ingestion failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Printf("langfuse: created generation %s (prompt=%s v%d)", params.ID, params.PromptName, params.PromptVersion)
	return nil
}

// ScoreDataType represents the type of score value.
type ScoreDataType string

const (
	ScoreDataTypeNumeric     ScoreDataType = "NUMERIC"
	ScoreDataTypeBoolean     ScoreDataType = "BOOLEAN"
	ScoreDataTypeCategorical ScoreDataType = "CATEGORICAL"
)

// ScoreParams holds parameters for creating a score in Langfuse.
type ScoreParams struct {
	TraceID       string        // Required: ID of the trace to associate the score with
	ObservationID string        // Optional: ID of a specific observation within the trace
	Name          string        // Required: Name of the score (e.g., "pareto/effectiveness")
	Value         float64       // Required: The score value
	DataType      ScoreDataType // Required: Type of the score (NUMERIC, BOOLEAN, CATEGORICAL)
	Comment       string        // Optional: Additional context about the score
}

// scoreRequest is the request body for creating a score via the Langfuse API.
type scoreRequest struct {
	TraceID       string  `json:"traceId"`
	ObservationID string  `json:"observationId,omitempty"`
	Name          string  `json:"name"`
	Value         float64 `json:"value"`
	DataType      string  `json:"dataType"`
	Comment       string  `json:"comment,omitempty"`
}

// CreateScore creates a single score in Langfuse.
func (c *Client) CreateScore(ctx context.Context, params ScoreParams) error {
	return c.createScoreInternal(ctx, params)
}

// CreateScoreBatch creates multiple scores in a single ingestion API call.
func (c *Client) CreateScoreBatch(ctx context.Context, scores []ScoreParams) error {
	if len(scores) == 0 {
		return nil
	}

	now := time.Now().UTC()
	events := make([]ingestionEvent, 0, len(scores))
	for i, s := range scores {
		if s.TraceID == "" || s.Name == "" {
			continue
		}
		dataType := s.DataType
		if dataType == "" {
			dataType = ScoreDataTypeNumeric
		}
		events = append(events, ingestionEvent{
			ID:        fmt.Sprintf("evt-score-%s-%d-%d", s.TraceID[:8], now.UnixMilli(), i),
			Type:      "score-create",
			Timestamp: now,
			Body: scoreRequest{
				TraceID:       s.TraceID,
				ObservationID: s.ObservationID,
				Name:          s.Name,
				Value:         s.Value,
				DataType:      string(dataType),
				Comment:       s.Comment,
			},
		})
	}

	if len(events) == 0 {
		return nil
	}

	batch := map[string]any{"batch": events}
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal score batch: %w", err)
	}

	apiURL := fmt.Sprintf("https://%s/api/public/ingestion", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: score batch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMultiStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: score batch ingestion failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) createScoreInternal(ctx context.Context, params ScoreParams) error {
	if params.TraceID == "" {
		return fmt.Errorf("langfuse: traceID is required")
	}
	if params.Name == "" {
		return fmt.Errorf("langfuse: name is required")
	}
	if params.DataType == "" {
		params.DataType = ScoreDataTypeNumeric
	}

	c.log.Printf("langfuse: creating score %q with value %.4f for trace %s", params.Name, params.Value, params.TraceID)

	reqBody := scoreRequest{
		TraceID:       params.TraceID,
		ObservationID: params.ObservationID,
		Name:          params.Name,
		Value:         params.Value,
		DataType:      string(params.DataType),
		Comment:       params.Comment,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Printf("langfuse: failed to marshal score %q: %v", params.Name, err)
		return fmt.Errorf("langfuse: failed to marshal score request: %w", err)
	}

	apiURL := fmt.Sprintf("https://%s/api/public/scores", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		c.log.Printf("langfuse: failed to create request for score %q: %v", params.Name, err)
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Printf("langfuse: request failed for score %q: %v", params.Name, err)
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		c.log.Printf("langfuse: API error creating score %q: %d %s", params.Name, resp.StatusCode, string(respBody))
		return fmt.Errorf("langfuse: API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ScoreConfig represents a Langfuse score configuration.
type ScoreConfig struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name"`
	DataType    string     `json:"dataType"` // "NUMERIC", "BOOLEAN", "CATEGORICAL"
	MinValue    *float64   `json:"minValue,omitempty"`
	MaxValue    *float64   `json:"maxValue,omitempty"`
	Categories  []Category `json:"categories,omitempty"` // for CATEGORICAL
	Description string     `json:"description,omitempty"`
}

// Category represents a category option for CATEGORICAL score configs.
type Category struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// scoreConfigResponse represents the API response for listing score configs.
type scoreConfigResponse struct {
	Data []ScoreConfig `json:"data"`
	Meta struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		TotalItems int `json:"totalItems"`
		TotalPages int `json:"totalPages"`
	} `json:"meta"`
}

// CreateScoreConfig creates a new score configuration in Langfuse.
// If a config with the same name already exists, this operation is idempotent.
func (c *Client) CreateScoreConfig(ctx context.Context, cfg ScoreConfig) error {
	apiURL := fmt.Sprintf("https://%s/api/public/score-configs", c.host)

	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal score config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		c.log.Printf("langfuse: created score config %q (type: %s)", cfg.Name, cfg.DataType)
		return nil
	}

	// Check if it's a conflict (already exists) - treat as success for idempotency
	if resp.StatusCode == http.StatusConflict {
		c.log.Printf("langfuse: score config %q already exists (idempotent)", cfg.Name)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("langfuse: API error %d: %s", resp.StatusCode, string(respBody))
}

// ListScoreConfigs retrieves all score configurations from Langfuse.
func (c *Client) ListScoreConfigs(ctx context.Context) ([]ScoreConfig, error) {
	apiURL := fmt.Sprintf("https://%s/api/public/score-configs", c.host)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("langfuse: API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to read response: %w", err)
	}

	var apiResp scoreConfigResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("langfuse: failed to parse response: %w", err)
	}

	return apiResp.Data, nil
}

// Ping checks if the Langfuse API is reachable and authenticated.
func (c *Client) Ping(ctx context.Context) error {
	// Use the score-configs endpoint as a lightweight health check
	apiURL := fmt.Sprintf("https://%s/api/public/score-configs?limit=1", c.host)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DatasetParams contains parameters for creating a dataset.
type DatasetParams struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// DatasetItemParams contains parameters for creating a dataset item.
type DatasetItemParams struct {
	DatasetName    string         `json:"datasetName"`
	Input          any            `json:"input"`
	ExpectedOutput any            `json:"expectedOutput,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	ID             string         `json:"id,omitempty"`
}

// DatasetRunItemParams contains parameters for creating a dataset run item.
type DatasetRunItemParams struct {
	DatasetItemID string             `json:"datasetItemId"`
	RunName       string             `json:"runName"`
	Output        any                `json:"output,omitempty"`
	Scores        map[string]float64 `json:"scores,omitempty"`
	Metadata      map[string]any     `json:"metadata,omitempty"`
}

// Dataset represents a Langfuse dataset.
type Dataset struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

// DatasetItem represents a Langfuse dataset item.
type DatasetItem struct {
	ID             string         `json:"id"`
	DatasetID      string         `json:"datasetId"`
	Input          any            `json:"input"`
	ExpectedOutput any            `json:"expectedOutput"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

// CreateDataset creates a new dataset in Langfuse.
// If the dataset already exists, it returns nil (idempotent).
func (c *Client) CreateDataset(ctx context.Context, params DatasetParams) error {
	apiURL := fmt.Sprintf("https://%s/api/public/datasets", c.host)

	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal dataset params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		c.log.Printf("langfuse: created dataset %q", params.Name)
		return nil
	}
	if resp.StatusCode == http.StatusConflict {
		c.log.Printf("langfuse: dataset %q already exists (idempotent)", params.Name)
		return nil // Dataset already exists, this is fine
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("langfuse: create dataset failed with status %d: %s", resp.StatusCode, string(respBody))
}

// CreateDatasetItem adds an item to a dataset.
func (c *Client) CreateDatasetItem(ctx context.Context, params DatasetItemParams) error {
	apiURL := fmt.Sprintf("https://%s/api/public/dataset-items", c.host)

	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal dataset item params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		c.log.Printf("langfuse: created dataset item for dataset %q", params.DatasetName)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("langfuse: create dataset item failed with status %d: %s", resp.StatusCode, string(respBody))
}

// CreateDatasetRunItem records the output of running a dataset item.
func (c *Client) CreateDatasetRunItem(ctx context.Context, params DatasetRunItemParams) error {
	apiURL := fmt.Sprintf("https://%s/api/public/dataset-run-items", c.host)

	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal dataset run item params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		c.log.Printf("langfuse: created dataset run item for run %q", params.RunName)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("langfuse: create dataset run item failed with status %d: %s", resp.StatusCode, string(respBody))
}

// GetDataset retrieves a dataset by name.
func (c *Client) GetDataset(ctx context.Context, name string) (*Dataset, error) {
	escapedName := url.PathEscape(name)
	apiURL := fmt.Sprintf("https://%s/api/public/datasets/%s", c.host, escapedName)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Dataset doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("langfuse: get dataset failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to read response: %w", err)
	}

	var dataset Dataset
	if err := json.Unmarshal(body, &dataset); err != nil {
		return nil, fmt.Errorf("langfuse: failed to parse dataset: %w", err)
	}

	return &dataset, nil
}

// getDatasetItemsResponse represents the paginated response for dataset items.
type getDatasetItemsResponse struct {
	Data []DatasetItem `json:"data"`
	Meta struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		TotalItems int `json:"totalItems"`
		TotalPages int `json:"totalPages"`
	} `json:"meta"`
}

// GetDatasetItems retrieves all items from a dataset.
func (c *Client) GetDatasetItems(ctx context.Context, datasetName string) ([]DatasetItem, error) {
	var allItems []DatasetItem
	page := 1
	limit := 100

	for {
		escapedName := url.PathEscape(datasetName)
		apiURL := fmt.Sprintf("https://%s/api/public/datasets/%s/items?page=%d&limit=%d", c.host, escapedName, page, limit)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("langfuse: failed to create request: %w", err)
		}

		req.SetBasicAuth(c.publicKey, c.secretKey)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("langfuse: request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("langfuse: get dataset items failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("langfuse: failed to read response: %w", err)
		}

		var result getDatasetItemsResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("langfuse: failed to parse dataset items: %w", err)
		}

		allItems = append(allItems, result.Data...)

		if page >= result.Meta.TotalPages {
			break
		}
		page++
	}

	return allItems, nil
}

// EvaluatorConfig configures a Langfuse managed evaluator.
type EvaluatorConfig struct {
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	EvaluatorType   string            `json:"evaluatorType"` // "llm" for LLM-as-a-judge
	Model           string            `json:"model,omitempty"`
	Template        string            `json:"template"`
	VariableMapping map[string]string `json:"variableMapping,omitempty"` // JSONPath mappings
	TargetFilter    map[string]any    `json:"targetFilter,omitempty"`    // Optional trace filters
	Sampling        float64           `json:"sampling"`                  // 0.0-1.0
	ScoreName       string            `json:"scoreName,omitempty"`       // Name for the resulting score
}

// Evaluator represents a Langfuse managed evaluator.
type Evaluator struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	EvaluatorType   string            `json:"evaluatorType"`
	Model           string            `json:"model"`
	Template        string            `json:"template"`
	VariableMapping map[string]string `json:"variableMapping"`
	TargetFilter    map[string]any    `json:"targetFilter"`
	Sampling        float64           `json:"sampling"`
	ScoreName       string            `json:"scoreName"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

// evaluatorsResponse represents the API response for listing evaluators.
type evaluatorsResponse struct {
	Data []Evaluator `json:"data"`
}

// CreateEvaluator creates a new managed evaluator in Langfuse.
// Note: Langfuse's evaluator API may not be publicly available yet.
// This method will return an error if the API is not available, which callers
// should handle gracefully.
func (c *Client) CreateEvaluator(ctx context.Context, cfg EvaluatorConfig) error {
	apiURL := fmt.Sprintf("https://%s/api/public/evaluators", c.host)

	payload := map[string]any{
		"name":          cfg.Name,
		"evaluatorType": cfg.EvaluatorType,
		"template":      cfg.Template,
		"sampling":      cfg.Sampling,
	}

	if cfg.Description != "" {
		payload["description"] = cfg.Description
	}
	if cfg.Model != "" {
		payload["model"] = cfg.Model
	}
	if cfg.VariableMapping != nil {
		payload["variableMapping"] = cfg.VariableMapping
	}
	if cfg.TargetFilter != nil {
		payload["targetFilter"] = cfg.TargetFilter
	}
	if cfg.ScoreName != "" {
		payload["scoreName"] = cfg.ScoreName
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal evaluator config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: evaluator creation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle various response codes
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		c.log.Printf("langfuse: created evaluator %q", cfg.Name)
		return nil
	case http.StatusConflict:
		// Evaluator already exists - this is OK for idempotent setup
		c.log.Printf("langfuse: evaluator %q already exists (idempotent)", cfg.Name)
		return nil
	case http.StatusNotFound:
		// API endpoint doesn't exist yet - return error for caller to handle
		return fmt.Errorf("langfuse: evaluator API not available (404) - feature may not be enabled")
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("langfuse: evaluator creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}

// ListEvaluators retrieves all managed evaluators from Langfuse.
// Note: Langfuse's evaluator API may not be publicly available yet.
func (c *Client) ListEvaluators(ctx context.Context) ([]Evaluator, error) {
	apiURL := fmt.Sprintf("https://%s/api/public/evaluators", c.host)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.publicKey, c.secretKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("langfuse: evaluator list request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// API endpoint doesn't exist yet - return error for caller to handle
		return nil, fmt.Errorf("langfuse: evaluator API not available (404) - feature may not be enabled")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("langfuse: evaluator list failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to read evaluator list response: %w", err)
	}

	var evalResp evaluatorsResponse
	if err := json.Unmarshal(body, &evalResp); err != nil {
		return nil, fmt.Errorf("langfuse: failed to parse evaluator list response: %w", err)
	}

	return evalResp.Data, nil
}

// GetEvaluator retrieves a specific evaluator by name.
func (c *Client) GetEvaluator(ctx context.Context, name string) (*Evaluator, error) {
	evaluators, err := c.ListEvaluators(ctx)
	if err != nil {
		return nil, err
	}

	for _, e := range evaluators {
		if e.Name == name {
			return &e, nil
		}
	}

	return nil, fmt.Errorf("langfuse: evaluator %q not found", name)
}

// EvaluatorExists checks if an evaluator with the given name exists.
func (c *Client) EvaluatorExists(ctx context.Context, name string) (bool, error) {
	evaluators, err := c.ListEvaluators(ctx)
	if err != nil {
		// If the API is not available, assume evaluators don't exist
		if strings.Contains(err.Error(), "404") {
			c.log.Printf("langfuse: evaluator API not available, assuming evaluator %q does not exist", name)
			return false, nil
		}
		return false, err
	}

	for _, e := range evaluators {
		if e.Name == name {
			return true, nil
		}
	}

	return false, nil
}
