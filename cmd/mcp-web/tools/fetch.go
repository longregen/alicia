package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/longregen/alicia/cmd/mcp-web/pipeline"
)

// FetchRawTool returns the raw HTTP response
type FetchRawTool struct{}

func NewFetchRawTool() *FetchRawTool {
	return &FetchRawTool{}
}

func (t *FetchRawTool) Name() string {
	return "fetch_raw"
}

func (t *FetchRawTool) Description() string {
	return "Fetches a URL and returns the raw HTTP response including headers. Use this for APIs, JSON endpoints, or when you need the unprocessed response. For web pages, prefer the 'read' tool instead."
}

func (t *FetchRawTool) InputSchema() map[string]any {
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

func (t *FetchRawTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("url is required")
	}

	opts := &pipeline.FetchOptions{
		FollowRedirects: true,
	}

	if method, ok := args["method"].(string); ok {
		opts.Method = method
	}

	if headers, ok := args["headers"].(map[string]any); ok {
		opts.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				opts.Headers[k] = s
			}
		}
	}

	if body, ok := args["body"].(string); ok && body != "" {
		opts.Body = strings.NewReader(body)
	}

	if follow, ok := args["follow_redirects"].(bool); ok {
		opts.FollowRedirects = follow
	}

	result, err := pipeline.Fetch(ctx, url, opts)
	if err != nil {
		return "", err
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(output), nil
}

// FetchStructuredTool extracts structured data from a page using CSS selectors
type FetchStructuredTool struct{}

func NewFetchStructuredTool() *FetchStructuredTool {
	return &FetchStructuredTool{}
}

func (t *FetchStructuredTool) Name() string {
	return "fetch_structured"
}

func (t *FetchStructuredTool) Description() string {
	return "Fetches a web page and extracts structured data using CSS selectors (powered by goquery). Supports full CSS selector syntax including classes, IDs, attributes, pseudo-selectors, and combinators."
}

func (t *FetchStructuredTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch",
			},
			"selectors": map[string]any{
				"type":        "object",
				"description": "A map of field names to CSS selectors. Examples: {\"title\": \"h1\", \"prices\": \".product .price\", \"links\": \"nav a[href]\", \"items\": \"ul.list > li\"}",
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

func (t *FetchStructuredTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("url is required")
	}

	selectors, ok := args["selectors"].(map[string]any)
	if !ok || len(selectors) == 0 {
		return "", fmt.Errorf("selectors is required and must be a non-empty object")
	}

	extractType := "text"
	if v, ok := args["extract"].(string); ok && v != "" {
		extractType = v
	}

	// Fetch the HTML
	htmlContent, finalURL, err := pipeline.FetchHTML(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Parse with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract data using selectors
	result := map[string]any{
		"url": finalURL,
	}
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
		} else {
			data[name] = extracted
		}
	}
	result["data"] = data

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(output), nil
}
