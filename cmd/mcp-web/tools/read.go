package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longregen/alicia/cmd/mcp-web/pipeline"
)

// ReadTool fetches a URL and returns clean markdown content
type ReadTool struct{}

func NewReadTool() *ReadTool {
	return &ReadTool{}
}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return "Fetches a web page and returns its main content as clean, LLM-friendly markdown. Automatically extracts the main article content, removes navigation/ads/boilerplate, and converts to markdown format. Supports JavaScript rendering for dynamic pages. Best for reading articles, documentation, and blog posts."
}

func (t *ReadTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch and convert to markdown",
			},
			"include_links": map[string]any{
				"type":        "boolean",
				"description": "Whether to preserve hyperlinks in the markdown output (default: true)",
				"default":     true,
			},
			"include_images": map[string]any{
				"type":        "boolean",
				"description": "Whether to include image references in the output (default: false)",
				"default":     false,
			},
			"max_length": map[string]any{
				"type":        "integer",
				"description": "Maximum character length of the output (default: 50000). Content will be truncated if longer.",
				"default":     50000,
			},
			"wait_for": map[string]any{
				"type":        "string",
				"description": "JavaScript rendering wait strategy. Use 'none' for static pages (fastest), 'load' to wait for page load, 'domcontentloaded' for DOM ready, or 'networkidle' for SPAs/dynamic content (slowest but most complete). Default: 'none'",
				"enum":        []string{"none", "load", "domcontentloaded", "networkidle"},
				"default":     "none",
			},
			"wait_ms": map[string]any{
				"type":        "integer",
				"description": "Additional milliseconds to wait after the wait_for condition is met (useful for animations/lazy loading). Only applies when wait_for is not 'none'. Default: 0",
				"default":     0,
			},
		},
		"required": []string{"url"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("url is required")
	}

	// Parse options
	includeLinks := true
	if v, ok := args["include_links"].(bool); ok {
		includeLinks = v
	}

	includeImages := false
	if v, ok := args["include_images"].(bool); ok {
		includeImages = v
	}

	maxLength := 50000
	if v, ok := args["max_length"].(float64); ok {
		maxLength = int(v)
	}

	waitFor := "none"
	if v, ok := args["wait_for"].(string); ok {
		waitFor = v
	}

	waitMS := 0
	if v, ok := args["wait_ms"].(float64); ok {
		waitMS = int(v)
	}

	var htmlContent, finalURL string
	var err error

	// Fetch HTML - either with JS rendering or plain HTTP
	if waitFor != "none" {
		// Use Playwright for JavaScript rendering
		var strategy pipeline.WaitStrategy
		switch waitFor {
		case "load":
			strategy = pipeline.WaitLoad
		case "domcontentloaded":
			strategy = pipeline.WaitDOMContentLoaded
		case "networkidle":
			strategy = pipeline.WaitNetworkIdle
		default:
			strategy = pipeline.WaitLoad
		}
		htmlContent, finalURL, err = pipeline.FetchWithJS(ctx, url, strategy, waitMS)
	} else {
		// Use plain HTTP fetch (faster for static pages)
		htmlContent, finalURL, err = pipeline.FetchHTML(ctx, url)
	}

	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Extract main content using readability
	content, err := pipeline.ExtractContent(htmlContent, finalURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	// Convert to markdown
	markdown, err := pipeline.HTMLToMarkdown(content.Content, finalURL)
	if err != nil {
		return "", fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Post-process based on options
	if !includeLinks {
		markdown = stripLinks(markdown)
	}
	if !includeImages {
		markdown = stripImages(markdown)
	}

	// Truncate if needed
	if len(markdown) > maxLength {
		markdown = markdown[:maxLength] + "\n\n[Content truncated...]"
	}

	// Build result with metadata
	result := ReadResult{
		URL:             finalURL,
		Title:           content.Title,
		Content:         markdown,
		WordCount:       content.WordCount,
		Excerpt:         content.Excerpt,
		Author:          content.Byline,
		SiteName:        content.SiteName,
		JSRendered:      waitFor != "none",
	}

	// Estimate tokens (rough: ~4 chars per token)
	result.EstimatedTokens = len(markdown) / 4

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(output), nil
}

// ReadResult is the output of the read tool
type ReadResult struct {
	URL             string `json:"url"`
	Title           string `json:"title,omitempty"`
	Content         string `json:"content"`
	WordCount       int    `json:"word_count"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Excerpt         string `json:"excerpt,omitempty"`
	Author          string `json:"author,omitempty"`
	SiteName        string `json:"site_name,omitempty"`
	JSRendered      bool   `json:"js_rendered"`
}

// stripLinks removes markdown links but keeps the text
func stripLinks(md string) string {
	// Replace [text](url) with just text
	result := md
	for {
		start := strings.Index(result, "[")
		if start == -1 {
			break
		}

		// Find the matching ]
		bracketEnd := -1
		depth := 1
		for i := start + 1; i < len(result); i++ {
			if result[i] == '[' {
				depth++
			} else if result[i] == ']' {
				depth--
				if depth == 0 {
					bracketEnd = i
					break
				}
			}
		}

		if bracketEnd == -1 {
			break
		}

		// Check if followed by (url)
		if bracketEnd+1 < len(result) && result[bracketEnd+1] == '(' {
			parenEnd := strings.Index(result[bracketEnd+1:], ")")
			if parenEnd != -1 {
				parenEnd += bracketEnd + 1
				text := result[start+1 : bracketEnd]
				result = result[:start] + text + result[parenEnd+1:]
				continue
			}
		}

		// Not a link, skip this bracket
		result = result[:start] + result[start+1:]
	}
	return result
}

// stripImages removes markdown images
func stripImages(md string) string {
	result := md
	for {
		start := strings.Index(result, "![")
		if start == -1 {
			break
		}

		// Find the matching ]
		bracketEnd := strings.Index(result[start:], "]")
		if bracketEnd == -1 {
			break
		}
		bracketEnd += start

		// Check if followed by (url)
		if bracketEnd+1 < len(result) && result[bracketEnd+1] == '(' {
			parenEnd := strings.Index(result[bracketEnd+1:], ")")
			if parenEnd != -1 {
				parenEnd += bracketEnd + 1
				result = result[:start] + result[parenEnd+1:]
				continue
			}
		}

		// Not properly formatted, skip
		result = result[:start] + result[start+2:]
	}
	return result
}
