package builtin

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

const (
	// DuckDuckGoSearchURL is the base URL for DuckDuckGo instant answer API
	DuckDuckGoSearchURL = "https://html.duckduckgo.com/html/"
	// SearchTimeout is the timeout for web search requests
	SearchTimeout = 10 * time.Second
	// MaxSearchResults is the maximum number of search results to return
	MaxSearchResults = 10
)

// RegisterWebSearch registers the web search tool with the tool service
// Uses DuckDuckGo HTML search to provide real web search results
func RegisterWebSearch(ctx context.Context, toolService ports.ToolService) error {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to execute",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 5)",
				"default":     5,
			},
		},
		"required": []string{"query"},
	}

	tool, err := toolService.EnsureTool(
		ctx,
		"web_search",
		"Searches the web for information. Returns a list of search results with titles, URLs, and snippets.",
		schema,
	)
	if err != nil {
		return fmt.Errorf("failed to register web search tool: %w", err)
	}

	// Register the executor with DuckDuckGo HTML search
	err = toolService.RegisterExecutor("web_search", func(ctx context.Context, arguments map[string]any) (any, error) {
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("query must be a string")
		}

		// Validate query
		query = strings.TrimSpace(query)
		if query == "" {
			return nil, fmt.Errorf("search query cannot be empty")
		}

		// Validate query length (max 500 characters)
		const maxQueryLength = 500
		if len(query) > maxQueryLength {
			return nil, fmt.Errorf("search query exceeds maximum length: %d characters (limit: %d)", len(query), maxQueryLength)
		}

		limit := 5
		if l, ok := arguments["limit"]; ok {
			switch v := l.(type) {
			case float64:
				limit = int(v)
			case int:
				limit = v
			}
		}

		// Validate and enforce limit bounds
		if limit < 1 {
			return nil, fmt.Errorf("limit must be at least 1 (got %d)", limit)
		}
		if limit > MaxSearchResults {
			return nil, fmt.Errorf("limit exceeds maximum: %d (limit: %d)", limit, MaxSearchResults)
		}

		// Perform DuckDuckGo search
		results, err := performDuckDuckGoSearch(ctx, query, limit)
		if err != nil {
			return nil, fmt.Errorf("web search failed: %w", err)
		}

		return map[string]any{
			"query":        query,
			"limit":        limit,
			"result_count": len(results),
			"results":      results,
		}, nil
	})

	if err != nil {
		return fmt.Errorf("failed to register web search executor: %w", err)
	}

	log.Printf("Registered web search tool with DuckDuckGo: %s", tool.ID)
	return nil
}

// GetWebSearchTool returns the web search tool definition
func GetWebSearchTool() *models.Tool {
	return &models.Tool{
		Name:        "web_search",
		Description: "Searches the web for information",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query",
				},
				"limit": map[string]any{
					"type":    "integer",
					"default": 5,
				},
			},
			"required": []string{"query"},
		},
		Enabled: true,
	}
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// performDuckDuckGoSearch performs a web search using DuckDuckGo HTML
func performDuckDuckGoSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: SearchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 5 redirects
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Prepare form data for POST request
	formData := url.Values{}
	formData.Set("q", query)
	formData.Set("b", "")       // Page offset
	formData.Set("kl", "us-en") // Language

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", DuckDuckGoSearchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	// Set headers to mimic a real browser
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Alicia/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned HTTP status %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}

	// Parse HTML to extract search results
	results := parseSearchResults(string(body), limit)

	// If no results found, provide a helpful error
	if len(results) == 0 {
		return nil, fmt.Errorf("no search results found for query: %q (this may indicate the HTML structure has changed or the search failed)", query)
	}

	return results, nil
}

// parseSearchResults extracts search results from DuckDuckGo HTML
func parseSearchResults(html string, limit int) []SearchResult {
	var results []SearchResult

	// DuckDuckGo HTML structure patterns
	// Try multiple patterns for robustness as DuckDuckGo may change their HTML structure

	// Pattern 1: result__body containers (current structure)
	resultPattern1 := regexp.MustCompile(`<div class="result__body">([\s\S]*?)</div>[\s\S]*?(?:<div class="result__body">|<div class="footer">|$)`)
	// Pattern 2: Backup pattern for links
	linkPattern := regexp.MustCompile(`<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	// Pattern 3: Snippet pattern
	snippetPattern := regexp.MustCompile(`<a[^>]+class="[^"]*result__snippet[^"]*"[^>]*>([\s\S]*?)</a>`)

	// Find all result blocks using pattern 1
	resultBlocks := resultPattern1.FindAllStringSubmatch(html, -1)

	// If pattern 1 doesn't work well, try extracting results differently
	if len(resultBlocks) < 1 {
		// Fallback: Find all links that look like results
		linkMatches := linkPattern.FindAllStringSubmatch(html, -1)
		for i, match := range linkMatches {
			if i >= limit {
				break
			}
			if len(match) >= 3 {
				// Skip DuckDuckGo internal links
				url := match[1]
				if strings.Contains(url, "duckduckgo.com") || strings.HasPrefix(url, "/") {
					continue
				}

				result := SearchResult{
					URL:   decodeHTMLEntities(url),
					Title: decodeHTMLEntities(match[2]),
				}
				results = append(results, result)
			}
		}
	} else {
		// Process result blocks from pattern 1
		for i, block := range resultBlocks {
			if i >= limit {
				break
			}

			blockHTML := block[1]

			// Extract title and URL from the block
			linkMatch := linkPattern.FindStringSubmatch(blockHTML)
			snippetMatch := snippetPattern.FindStringSubmatch(blockHTML)

			if len(linkMatch) >= 3 {
				url := linkMatch[1]
				// Skip DuckDuckGo internal links
				if strings.Contains(url, "duckduckgo.com") || strings.HasPrefix(url, "/") {
					continue
				}

				result := SearchResult{
					URL:   decodeHTMLEntities(url),
					Title: decodeHTMLEntities(linkMatch[2]),
				}

				if len(snippetMatch) >= 2 {
					result.Snippet = decodeHTMLEntities(snippetMatch[1])
				}

				results = append(results, result)
			}
		}
	}

	return results
}

// decodeHTMLEntities decodes common HTML entities and removes HTML tags
func decodeHTMLEntities(s string) string {
	// First, remove HTML tags
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	result := tagPattern.ReplaceAllString(s, "")

	// Common HTML entities
	replacements := map[string]string{
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&#39;":    "'",
		"&apos;":   "'",
		"&nbsp;":   " ",
		"&#x27;":   "'",
		"&#x2F;":   "/",
		"&ndash;":  "–",
		"&mdash;":  "—",
		"&hellip;": "…",
		"&copy;":   "©",
		"&reg;":    "®",
		"&trade;":  "™",
		"&ldquo;":  "\u201C",
		"&rdquo;":  "\u201D",
		"&lsquo;":  "\u2018",
		"&rsquo;":  "\u2019",
	}

	for entity, char := range replacements {
		result = strings.ReplaceAll(result, entity, char)
	}

	// Handle numeric HTML entities (e.g., &#39; or &#x27;)
	numericEntityPattern := regexp.MustCompile(`&#(\d+);`)
	result = numericEntityPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the numeric code
		numStr := strings.TrimPrefix(strings.TrimSuffix(match, ";"), "&#")
		if num, err := strconv.Atoi(numStr); err == nil && num < 1114112 {
			return string(rune(num))
		}
		return match
	})

	// Handle hexadecimal HTML entities (e.g., &#x27;)
	hexEntityPattern := regexp.MustCompile(`&#[xX]([0-9a-fA-F]+);`)
	result = hexEntityPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the hex code
		hexStr := strings.TrimPrefix(strings.TrimSuffix(match, ";"), "&#x")
		hexStr = strings.TrimPrefix(hexStr, "X")
		if num, err := strconv.ParseInt(hexStr, 16, 32); err == nil && num < 1114112 {
			return string(rune(num))
		}
		return match
	})

	// Clean up extra whitespace
	result = strings.Join(strings.Fields(result), " ")

	return strings.TrimSpace(result)
}
