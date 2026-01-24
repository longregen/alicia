package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/longregen/alicia/mcp/web/pipeline"
	"github.com/longregen/alicia/shared/httpclient"
)

const (
	DuckDuckGoSearchURL = "https://html.duckduckgo.com/html/"
	KagiSearchURL       = "https://kagi.com/api/v0/search"
	SearchTimeout       = 15 * time.Second
	MaxSearchResults    = 10
)

// Pre-compiled regex patterns for search result parsing
var (
	ddgLinkPattern = regexp.MustCompile(`<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	snippetPattern = regexp.MustCompile(`<a[^>]+class="[^"]*result__snippet[^"]*"[^>]*>([\s\S]*?)</a>`)
	resultPattern  = regexp.MustCompile(`<div class="result[^"]*"[^>]*>([\s\S]*?)</div>\s*(?:<div class="result|<div class="footer|$)`)
	tagPattern     = regexp.MustCompile(`<[^>]*>`)
	numPattern     = regexp.MustCompile(`&#(\d+);`)
)

// htmlEntityReplacements maps HTML entities to their decoded characters
var htmlEntityReplacements = map[string]string{
	"&amp;":  "&",
	"&lt;":   "<",
	"&gt;":   ">",
	"&quot;": "\"",
	"&#39;":  "'",
	"&apos;": "'",
	"&nbsp;": " ",
}

// SearchTool performs web searches
type SearchTool struct{}

func NewSearchTool() *SearchTool {
	return &SearchTool{}
}

func (t *SearchTool) Name() string {
	return "search"
}

func (t *SearchTool) Description() string {
	return "Searches the web using DuckDuckGo and returns results. Can optionally fetch and convert the content of each result to markdown for deeper analysis."
}

func (t *SearchTool) InputSchema() map[string]any {
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

func (t *SearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	query = strings.TrimSpace(query)
	if len(query) > 500 {
		return "", fmt.Errorf("query too long (max 500 characters)")
	}

	numResults := 5
	if v, ok := args["num_results"].(float64); ok {
		numResults = int(v)
	}
	if numResults < 1 {
		numResults = 1
	}
	if numResults > MaxSearchResults {
		numResults = MaxSearchResults
	}

	fetchContent := false
	if v, ok := args["fetch_content"].(bool); ok {
		fetchContent = v
	}

	// Perform the search
	results, err := performSearch(ctx, query, numResults)
	if err != nil {
		return "", err
	}

	// Optionally fetch content for each result
	if fetchContent {
		for i := range results {
			content, err := fetchResultContent(ctx, results[i].URL)
			if err != nil {
				results[i].Content = fmt.Sprintf("Error fetching content: %v", err)
			} else {
				results[i].Content = content
			}
		}
	}

	output := SearchOutput{
		Query:       query,
		ResultCount: len(results),
		Results:     results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	return string(data), nil
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Content string `json:"content,omitempty"`
}

// SearchOutput is the full search response
type SearchOutput struct {
	Query       string         `json:"query"`
	ResultCount int            `json:"result_count"`
	Results     []SearchResult `json:"results"`
}

func performSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// Use Kagi if API key is set, otherwise fall back to DuckDuckGo
	if kagiKey := os.Getenv("KAGI_API_KEY"); kagiKey != "" {
		return performKagiSearch(ctx, query, limit, kagiKey)
	}
	return performDuckDuckGoSearch(ctx, query, limit)
}

// KagiSearchResponse represents Kagi API response
type KagiSearchResponse struct {
	Meta struct {
		ID   string `json:"id"`
		Node string `json:"node"`
		Ms   int    `json:"ms"`
	} `json:"meta"`
	Data []struct {
		T       int    `json:"t"` // type: 0=search result, 1=related searches
		Rank    int    `json:"rank,omitempty"`
		URL     string `json:"url,omitempty"`
		Title   string `json:"title,omitempty"`
		Snippet string `json:"snippet,omitempty"`
	} `json:"data"`
}

func performKagiSearch(ctx context.Context, query string, limit int, apiKey string) ([]SearchResult, error) {
	client := httpclient.New(httpclient.WithTimeout(SearchTimeout))

	reqURL := fmt.Sprintf("%s?q=%s&limit=%d", KagiSearchURL, url.QueryEscape(query), limit)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kagi request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Kagi search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Kagi returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var kagiResp KagiSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&kagiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Kagi response: %w", err)
	}

	var results []SearchResult
	for _, item := range kagiResp.Data {
		if item.T != 0 { // Skip non-search results
			continue
		}
		if len(results) >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Snippet,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for query: %q", query)
	}

	return results, nil
}

func performDuckDuckGoSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	client := &http.Client{
		Timeout: SearchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	formData := url.Values{}
	formData.Set("q", query)
	formData.Set("b", "")
	formData.Set("kl", "us-en")

	req, err := http.NewRequestWithContext(ctx, "POST", DuckDuckGoSearchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MCPWeb/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
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

func parseSearchResults(html string, limit int) []SearchResult {
	var results []SearchResult

	// Find all result containers using pre-compiled patterns
	resultBlocks := resultPattern.FindAllStringSubmatch(html, -1)

	for _, block := range resultBlocks {
		if len(results) >= limit {
			break
		}

		blockHTML := block[1]

		linkMatch := ddgLinkPattern.FindStringSubmatch(blockHTML)
		if len(linkMatch) < 3 {
			continue
		}

		resultURL := linkMatch[1]
		// Skip internal DuckDuckGo links
		if strings.Contains(resultURL, "duckduckgo.com") || strings.HasPrefix(resultURL, "/") {
			continue
		}

		result := SearchResult{
			URL:   decodeEntities(resultURL),
			Title: decodeEntities(stripTags(linkMatch[2])),
		}

		snippetMatch := snippetPattern.FindStringSubmatch(blockHTML)
		if len(snippetMatch) > 1 {
			result.Snippet = decodeEntities(stripTags(snippetMatch[1]))
		}

		results = append(results, result)
	}

	// Fallback: direct link extraction if block parsing fails
	if len(results) == 0 {
		linkMatches := ddgLinkPattern.FindAllStringSubmatch(html, limit*2)
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
			results = append(results, SearchResult{
				URL:   decodeEntities(resultURL),
				Title: decodeEntities(stripTags(match[2])),
			})
		}
	}

	return results
}

func stripTags(s string) string {
	return strings.TrimSpace(tagPattern.ReplaceAllString(s, ""))
}

func decodeEntities(s string) string {
	for entity, char := range htmlEntityReplacements {
		s = strings.ReplaceAll(s, entity, char)
	}

	// Numeric entities
	s = numPattern.ReplaceAllStringFunc(s, func(match string) string {
		numStr := strings.TrimPrefix(strings.TrimSuffix(match, ";"), "&#")
		if num, err := strconv.Atoi(numStr); err == nil && num < 1114112 {
			return string(rune(num))
		}
		return match
	})

	return strings.TrimSpace(s)
}

func fetchResultContent(ctx context.Context, url string) (string, error) {
	htmlContent, finalURL, err := pipeline.FetchHTML(ctx, url)
	if err != nil {
		return "", err
	}

	content, err := pipeline.ExtractContent(htmlContent, finalURL)
	if err != nil {
		return "", err
	}

	markdown, err := pipeline.HTMLToMarkdown(content.Content, finalURL)
	if err != nil {
		return "", err
	}

	// Truncate to reasonable size
	markdown = pipeline.TruncateContent(markdown, 5000, "\n\n[truncated...]")

	return markdown, nil
}
