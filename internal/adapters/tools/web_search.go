package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

const (
	duckDuckGoSearchURL = "https://html.duckduckgo.com/html/"
	searchTimeout       = 15 * time.Second
	maxSearchResults    = 10
)

// WebSearchTool performs web searches
type WebSearchTool struct {
	client *http.Client
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: searchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Searches the web using DuckDuckGo and returns results. Can optionally fetch and convert the content of each result to markdown for deeper analysis."
}

func (t *WebSearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
			"num_results": map[string]any{
				"type":        "integer",
				"description": "Number of results to return (default: 5, max: 10)",
				"default":     5,
			},
			"fetch_content": map[string]any{
				"type":        "boolean",
				"description": "If true, fetches and converts each result page to markdown. This is slower but provides full content. (default: false)",
				"default":     false,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	query = strings.TrimSpace(query)
	if len(query) > 500 {
		return nil, fmt.Errorf("query too long (max 500 characters)")
	}

	numResults := 5
	if v, ok := args["num_results"].(float64); ok {
		numResults = int(v)
	}
	if numResults < 1 {
		numResults = 1
	}
	if numResults > maxSearchResults {
		numResults = maxSearchResults
	}

	fetchContent := false
	if v, ok := args["fetch_content"].(bool); ok {
		fetchContent = v
	}

	// Perform search
	results, err := t.performSearch(ctx, query, numResults)
	if err != nil {
		return nil, err
	}

	// Optionally fetch content
	if fetchContent {
		for i := range results {
			content, err := t.fetchResultContent(ctx, results[i].URL)
			if err != nil {
				results[i].Content = fmt.Sprintf("Error fetching content: %v", err)
			} else {
				results[i].Content = content
			}
		}
	}

	output := WebSearchResult{
		Query:       query,
		ResultCount: len(results),
		Results:     results,
	}

	return output, nil
}

// WebSearchResult is the full search response
type WebSearchResult struct {
	Query       string             `json:"query"`
	ResultCount int                `json:"result_count"`
	Results     []WebSearchHit     `json:"results"`
}

// WebSearchHit represents a single search result
type WebSearchHit struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Content string `json:"content,omitempty"`
}

func (t *WebSearchTool) performSearch(ctx context.Context, query string, limit int) ([]WebSearchHit, error) {
	formData := url.Values{}
	formData.Set("q", query)
	formData.Set("b", "")
	formData.Set("kl", "us-en")

	req, err := http.NewRequestWithContext(ctx, "POST", duckDuckGoSearchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Alicia/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	results := parseSearchResults(string(body), limit)
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for query: %q", query)
	}

	return results, nil
}

func parseSearchResults(html string, limit int) []WebSearchHit {
	var results []WebSearchHit

	// Pattern to find result links
	linkPattern := regexp.MustCompile(`<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	snippetPattern := regexp.MustCompile(`<a[^>]+class="[^"]*result__snippet[^"]*"[^>]*>([\s\S]*?)</a>`)

	// Find all result containers
	resultPattern := regexp.MustCompile(`<div class="result[^"]*"[^>]*>([\s\S]*?)</div>\s*(?:<div class="result|<div class="footer|$)`)
	resultBlocks := resultPattern.FindAllStringSubmatch(html, -1)

	for _, block := range resultBlocks {
		if len(results) >= limit {
			break
		}

		blockHTML := block[1]

		linkMatch := linkPattern.FindStringSubmatch(blockHTML)
		if len(linkMatch) < 3 {
			continue
		}

		resultURL := linkMatch[1]
		// Skip internal DuckDuckGo links
		if strings.Contains(resultURL, "duckduckgo.com") || strings.HasPrefix(resultURL, "/") {
			continue
		}

		result := WebSearchHit{
			URL:   decodeHTMLEntities(resultURL),
			Title: decodeHTMLEntities(stripHTMLTags(linkMatch[2])),
		}

		snippetMatch := snippetPattern.FindStringSubmatch(blockHTML)
		if len(snippetMatch) > 1 {
			result.Snippet = decodeHTMLEntities(stripHTMLTags(snippetMatch[1]))
		}

		results = append(results, result)
	}

	// Fallback: direct link extraction if block parsing fails
	if len(results) == 0 {
		linkMatches := linkPattern.FindAllStringSubmatch(html, limit*2)
		for _, match := range linkMatches {
			if len(results) >= limit {
				break
			}
			if len(match) < 3 {
				continue
			}
			resultURL := match[1]
			if strings.Contains(resultURL, "duckduckgo.com") || strings.HasPrefix(resultURL, "/") {
				continue
			}
			results = append(results, WebSearchHit{
				URL:   decodeHTMLEntities(resultURL),
				Title: decodeHTMLEntities(stripHTMLTags(match[2])),
			})
		}
	}

	return results
}

func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

func decodeHTMLEntities(s string) string {
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&apos;": "'",
		"&nbsp;": " ",
	}
	for entity, char := range replacements {
		s = strings.ReplaceAll(s, entity, char)
	}

	// Numeric entities
	numPattern := regexp.MustCompile(`&#(\d+);`)
	s = numPattern.ReplaceAllStringFunc(s, func(match string) string {
		numStr := strings.TrimPrefix(strings.TrimSuffix(match, ";"), "&#")
		if num, err := strconv.Atoi(numStr); err == nil && num < 1114112 {
			return string(rune(num))
		}
		return match
	})

	return strings.TrimSpace(s)
}

func (t *WebSearchTool) fetchResultContent(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Alicia/1.0)")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		return "", err
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(article.Content)
	if err != nil {
		return "", err
	}

	// Truncate to reasonable size
	const maxContentLength = 5000
	if len(markdown) > maxContentLength {
		markdown = markdown[:maxContentLength] + "\n[truncated...]"
	}

	return markdown, nil
}
