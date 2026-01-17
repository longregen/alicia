package pipeline

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/longregen/alicia/cmd/mcp-web/security"
)

const (
	DefaultTimeout    = 30 * time.Second
	MaxResponseSize   = 10 * 1024 * 1024 // 10MB
	MaxRedirects      = 5
	DefaultUserAgent  = "Mozilla/5.0 (compatible; MCPWeb/1.0; +https://github.com/longregen/alicia)"
)

// FetchOptions configures the HTTP fetch behavior
type FetchOptions struct {
	Timeout     time.Duration
	Headers     map[string]string
	Method      string
	Body        io.Reader
	FollowRedirects bool
}

// FetchResult holds the result of a fetch operation
type FetchResult struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	FinalURL    string            `json:"final_url"`
	ContentType string            `json:"content_type"`
}

// Fetch performs an HTTP request with security validation
func Fetch(ctx context.Context, url string, opts *FetchOptions) (*FetchResult, error) {
	// Validate URL to prevent SSRF
	if err := security.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	if opts == nil {
		opts = &FetchOptions{}
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}

	// Create client with redirect handling
	redirectCount := 0
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > MaxRedirects {
				return fmt.Errorf("too many redirects (max %d)", MaxRedirects)
			}

			// Validate redirect URL
			if err := security.ValidateURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect URL validation failed: %w", err)
			}

			if !opts.FollowRedirects {
				return http.ErrUseLastResponse
			}

			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, method, url, opts.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Apply custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract headers
	headers := make(map[string]string)
	for key := range resp.Header {
		headers[key] = resp.Header.Get(key)
	}

	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	return &FetchResult{
		StatusCode:  resp.StatusCode,
		Headers:     headers,
		Body:        string(body),
		FinalURL:    resp.Request.URL.String(),
		ContentType: contentType,
	}, nil
}

// FetchHTML fetches a URL and returns just the HTML content
func FetchHTML(ctx context.Context, url string) (string, string, error) {
	result, err := Fetch(ctx, url, &FetchOptions{
		FollowRedirects: true,
	})
	if err != nil {
		return "", "", err
	}

	if result.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d: %s", result.StatusCode, http.StatusText(result.StatusCode))
	}

	return result.Body, result.FinalURL, nil
}
