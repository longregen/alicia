package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// WebFetchRawTool returns the raw HTTP response
type WebFetchRawTool struct {
	client *http.Client
}

func NewWebFetchRawTool() *WebFetchRawTool {
	return &WebFetchRawTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *WebFetchRawTool) Name() string {
	return "web_fetch_raw"
}

func (t *WebFetchRawTool) Description() string {
	return "Fetches a URL and returns the raw HTTP response including headers. Use this for APIs, JSON endpoints, or when you need the unprocessed response. For web pages, prefer the 'web_read' tool instead."
}

func (t *WebFetchRawTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method (GET, POST, PUT, DELETE, etc.). Default: GET",
				"default":     "GET",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Custom HTTP headers to send with the request",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Request body (for POST, PUT, PATCH requests)",
			},
			"follow_redirects": map[string]any{
				"type":        "boolean",
				"description": "Whether to follow HTTP redirects (default: true)",
				"default":     true,
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchRawTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method := "GET"
	if v, ok := args["method"].(string); ok && v != "" {
		method = strings.ToUpper(v)
	}

	var bodyReader io.Reader
	if body, ok := args["body"].(string); ok && body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Alicia/1.0)")

	// Apply custom headers
	if headers, ok := args["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	// Configure redirect behavior
	client := t.client
	followRedirects := true
	if v, ok := args["follow_redirects"].(bool); ok {
		followRedirects = v
	}
	if !followRedirects {
		client = &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Build headers map
	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	result := WebFetchRawResult{
		URL:        resp.Request.URL.String(),
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       string(body),
		BodyLength: len(body),
	}

	return result, nil
}

// WebFetchRawResult is the output of the fetch raw tool
type WebFetchRawResult struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	BodyLength int               `json:"body_length"`
}

// WebFetchStructuredTool extracts structured data using CSS selectors
type WebFetchStructuredTool struct{}

func NewWebFetchStructuredTool() *WebFetchStructuredTool {
	return &WebFetchStructuredTool{}
}

func (t *WebFetchStructuredTool) Name() string {
	return "web_fetch_structured"
}

func (t *WebFetchStructuredTool) Description() string {
	return "Fetches a web page and extracts structured data using CSS selectors. Supports full CSS selector syntax including classes, IDs, attributes, pseudo-selectors, and combinators."
}

func (t *WebFetchStructuredTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch",
			},
			"selectors": map[string]any{
				"type":        "object",
				"description": "A map of field names to CSS selectors. Examples: {\"title\": \"h1\", \"prices\": \".product .price\", \"links\": \"nav a[href]\"}",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"extract": map[string]any{
				"type":        "string",
				"description": "What to extract: 'text' (default), 'html', or an attribute name like 'href', 'src', 'data-id'",
				"default":     "text",
			},
		},
		"required": []string{"url", "selectors"},
	}
}

func (t *WebFetchStructuredTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	selectors, ok := args["selectors"].(map[string]any)
	if !ok || len(selectors) == 0 {
		return nil, fmt.Errorf("selectors is required and must be a non-empty object")
	}

	extractType := "text"
	if v, ok := args["extract"].(string); ok && v != "" {
		extractType = v
	}

	// Fetch HTML
	htmlContent, finalURL, err := fetchHTML(ctx, url)
	if err != nil {
		return nil, err
	}

	// Parse with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract data using selectors
	data := make(map[string]any)
	for name, sel := range selectors {
		selector, ok := sel.(string)
		if !ok {
			continue
		}

		var extracted []string
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			var value string
			switch extractType {
			case "text":
				value = strings.TrimSpace(s.Text())
			case "html":
				value, _ = s.Html()
			default:
				// Treat as attribute name
				value, _ = s.Attr(extractType)
			}
			if value != "" {
				extracted = append(extracted, value)
			}
		})

		// Return single value if only one match, array otherwise
		if len(extracted) == 1 {
			data[name] = extracted[0]
		} else if len(extracted) > 0 {
			data[name] = extracted
		} else {
			data[name] = nil
		}
	}

	result := WebFetchStructuredResult{
		URL:  finalURL,
		Data: data,
	}

	return result, nil
}

// WebFetchStructuredResult is the output of the fetch structured tool
type WebFetchStructuredResult struct {
	URL  string         `json:"url"`
	Data map[string]any `json:"data"`
}
