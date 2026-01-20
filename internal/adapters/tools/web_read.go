package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// WebReadTool fetches a URL and returns clean markdown content
type WebReadTool struct {
	client *http.Client
}

func NewWebReadTool() *WebReadTool {
	return &WebReadTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (t *WebReadTool) Name() string {
	return "web_read"
}

func (t *WebReadTool) Description() string {
	return "Fetches a web page and returns its main content as clean, LLM-friendly markdown. Automatically extracts the main article content, removes navigation/ads/boilerplate, and converts to markdown format. Best for reading articles, documentation, and blog posts."
}

func (t *WebReadTool) Schema() map[string]any {
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
		},
		"required": []string{"url"},
	}
}

func (t *WebReadTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
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

	// Fetch HTML
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Alicia/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Extract main content using readability
	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	// Convert to markdown
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(article.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Post-process based on options
	if !includeLinks {
		markdown = stripLinks(markdown)
	}
	if !includeImages {
		markdown = stripImages(markdown)
	}

	// Clean up whitespace
	markdown = cleanWhitespace(markdown)

	// Truncate if needed
	if len(markdown) > maxLength {
		markdown = markdown[:maxLength] + "\n\n[Content truncated...]"
	}

	// Build result
	result := WebReadResult{
		URL:             resp.Request.URL.String(),
		Title:           article.Title,
		Content:         markdown,
		Excerpt:         article.Excerpt,
		Author:          article.Byline,
		SiteName:        article.SiteName,
		WordCount:       len(strings.Fields(article.TextContent)),
		EstimatedTokens: len(markdown) / 4,
	}

	return result, nil
}

// WebReadResult is the structured output of the web read tool
type WebReadResult struct {
	URL             string `json:"url"`
	Title           string `json:"title,omitempty"`
	Content         string `json:"content"`
	Excerpt         string `json:"excerpt,omitempty"`
	Author          string `json:"author,omitempty"`
	SiteName        string `json:"site_name,omitempty"`
	WordCount       int    `json:"word_count"`
	EstimatedTokens int    `json:"estimated_tokens"`
}

// stripLinks removes markdown links but keeps the text
func stripLinks(md string) string {
	// Replace [text](url) with just text
	re := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	return re.ReplaceAllString(md, "$1")
}

// stripImages removes markdown images
func stripImages(md string) string {
	// Remove ![alt](url)
	re := regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	return re.ReplaceAllString(md, "")
}

// cleanWhitespace normalizes whitespace in markdown
func cleanWhitespace(md string) string {
	// Replace multiple newlines with double newlines
	re := regexp.MustCompile(`\n{3,}`)
	md = re.ReplaceAllString(md, "\n\n")
	// Trim leading/trailing whitespace
	return strings.TrimSpace(md)
}

// fetchHTML is a helper to fetch raw HTML content
func fetchHTML(ctx context.Context, url string) (string, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Alicia/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), resp.Request.URL.String(), nil
}
