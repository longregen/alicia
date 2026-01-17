package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/longregen/alicia/cmd/mcp-web/pipeline"
)

// ExtractLinksTool extracts all links from a page
type ExtractLinksTool struct{}

func NewExtractLinksTool() *ExtractLinksTool {
	return &ExtractLinksTool{}
}

func (t *ExtractLinksTool) Name() string {
	return "extract_links"
}

func (t *ExtractLinksTool) Description() string {
	return "Extracts all links from a web page. Useful for discovering related pages, navigation structure, or finding specific resources. Can filter by internal/external links or URL patterns."
}

func (t *ExtractLinksTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to extract links from",
			},
			"filter": map[string]any{
				"type":        "string",
				"description": "Filter links: 'internal' (same domain), 'external' (different domain), or 'all' (default: all)",
				"enum":        []string{"all", "internal", "external"},
				"default":     "all",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Optional regex pattern to filter URLs (e.g., '\\.pdf$' for PDFs, '/blog/' for blog posts)",
			},
			"include_text": map[string]any{
				"type":        "boolean",
				"description": "Include the link text/anchor in results (default: true)",
				"default":     true,
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of links to return (default: 100)",
				"default":     100,
			},
		},
		"required": []string{"url"},
	}
}

func (t *ExtractLinksTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	targetURL, ok := args["url"].(string)
	if !ok || targetURL == "" {
		return "", fmt.Errorf("url is required")
	}

	filter := "all"
	if v, ok := args["filter"].(string); ok {
		filter = v
	}

	var pattern *regexp.Regexp
	if p, ok := args["pattern"].(string); ok && p != "" {
		var err error
		pattern, err = regexp.Compile(p)
		if err != nil {
			return "", fmt.Errorf("invalid pattern: %w", err)
		}
	}

	includeText := true
	if v, ok := args["include_text"].(bool); ok {
		includeText = v
	}

	maxResults := 100
	if v, ok := args["max_results"].(float64); ok {
		maxResults = int(v)
	}

	// Fetch the HTML
	htmlContent, finalURL, err := pipeline.FetchHTML(ctx, targetURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Parse base URL for internal/external filtering
	baseURL, err := url.Parse(finalURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Extract all links
	links := extractLinks(htmlContent, finalURL, includeText)

	// Apply filters
	var filtered []LinkInfo
	seen := make(map[string]bool)

	for _, link := range links {
		if len(filtered) >= maxResults {
			break
		}

		// Skip duplicates
		if seen[link.URL] {
			continue
		}

		// Parse link URL for filtering
		linkURL, err := url.Parse(link.URL)
		if err != nil {
			continue
		}

		// Apply internal/external filter
		isInternal := linkURL.Host == "" || linkURL.Host == baseURL.Host
		switch filter {
		case "internal":
			if !isInternal {
				continue
			}
		case "external":
			if isInternal {
				continue
			}
		}

		// Apply pattern filter
		if pattern != nil && !pattern.MatchString(link.URL) {
			continue
		}

		seen[link.URL] = true
		filtered = append(filtered, link)
	}

	output := LinksOutput{
		URL:        finalURL,
		TotalFound: len(links),
		Returned:   len(filtered),
		Filter:     filter,
		Links:      filtered,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	return string(data), nil
}

// LinkInfo represents a single link
type LinkInfo struct {
	URL  string `json:"url"`
	Text string `json:"text,omitempty"`
	Rel  string `json:"rel,omitempty"`
}

// LinksOutput is the full links response
type LinksOutput struct {
	URL        string     `json:"url"`
	TotalFound int        `json:"total_found"`
	Returned   int        `json:"returned"`
	Filter     string     `json:"filter"`
	Links      []LinkInfo `json:"links"`
}

func extractLinks(html string, baseURL string, includeText bool) []LinkInfo {
	var links []LinkInfo

	// Pattern to match anchor tags
	linkPattern := regexp.MustCompile(`<a\s+([^>]*)>([\s\S]*?)</a>`)
	hrefPattern := regexp.MustCompile(`href=["']([^"']+)["']`)
	relPattern := regexp.MustCompile(`rel=["']([^"']+)["']`)

	matches := linkPattern.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		attrs := match[1]
		text := match[2]

		// Extract href
		hrefMatch := hrefPattern.FindStringSubmatch(attrs)
		if len(hrefMatch) < 2 {
			continue
		}
		href := hrefMatch[1]

		// Skip javascript:, mailto:, tel:, etc.
		if strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") ||
			strings.HasPrefix(href, "#") {
			continue
		}

		// Resolve relative URLs
		href = resolveHref(href, baseURL)

		link := LinkInfo{
			URL: href,
		}

		if includeText {
			link.Text = strings.TrimSpace(stripTagsSimple(text))
		}

		// Extract rel attribute
		relMatch := relPattern.FindStringSubmatch(attrs)
		if len(relMatch) > 1 {
			link.Rel = relMatch[1]
		}

		links = append(links, link)
	}

	return links
}

func resolveHref(href, baseURL string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	if strings.HasPrefix(href, "//") {
		return base.Scheme + ":" + href
	}

	if strings.HasPrefix(href, "/") {
		return base.Scheme + "://" + base.Host + href
	}

	// Relative path
	if base.Path == "" || strings.HasSuffix(base.Path, "/") {
		return base.Scheme + "://" + base.Host + base.Path + href
	}

	// Remove last path segment
	lastSlash := strings.LastIndex(base.Path, "/")
	if lastSlash >= 0 {
		return base.Scheme + "://" + base.Host + base.Path[:lastSlash+1] + href
	}

	return base.Scheme + "://" + base.Host + "/" + href
}

func stripTagsSimple(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	result := re.ReplaceAllString(s, " ")
	// Collapse whitespace
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}
