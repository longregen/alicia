package tools

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type WebExtractLinksTool struct{}

func NewWebExtractLinksTool() *WebExtractLinksTool {
	return &WebExtractLinksTool{}
}

func (t *WebExtractLinksTool) Name() string {
	return "web_extract_links"
}

func (t *WebExtractLinksTool) Description() string {
	return "Extracts all links from a web page. Useful for discovering related pages, navigation structure, or finding specific resources. Can filter by internal/external links or URL patterns."
}

func (t *WebExtractLinksTool) Schema() map[string]any {
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

func (t *WebExtractLinksTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return nil, fmt.Errorf("url is required")
	}

	filter := "all"
	if v, ok := args["filter"].(string); ok && v != "" {
		filter = v
	}

	var pattern *regexp.Regexp
	if v, ok := args["pattern"].(string); ok && v != "" {
		var err error
		pattern, err = regexp.Compile(v)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	includeText := true
	if v, ok := args["include_text"].(bool); ok {
		includeText = v
	}

	maxResults := 100
	if v, ok := args["max_results"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	htmlContent, finalURL, err := fetchHTML(ctx, urlStr)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(finalURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var links []WebLink
	seen := make(map[string]bool)

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if len(links) >= maxResults {
			return
		}

		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}

		linkURL, err := baseURL.Parse(href)
		if err != nil {
			return
		}

		if linkURL.Scheme == "javascript" || linkURL.Scheme == "mailto" {
			return
		}

		absoluteURL := linkURL.String()

		if seen[absoluteURL] {
			return
		}

		isInternal := linkURL.Host == baseURL.Host || linkURL.Host == ""
		switch filter {
		case "internal":
			if !isInternal {
				return
			}
		case "external":
			if isInternal {
				return
			}
		}

		if pattern != nil && !pattern.MatchString(absoluteURL) {
			return
		}

		seen[absoluteURL] = true

		link := WebLink{
			URL:      absoluteURL,
			Internal: isInternal,
		}

		if includeText {
			link.Text = strings.TrimSpace(s.Text())
		}

		links = append(links, link)
	})

	result := WebExtractLinksResult{
		URL:        finalURL,
		TotalFound: len(links),
		Links:      links,
	}

	return result, nil
}

type WebLink struct {
	URL      string `json:"url"`
	Text     string `json:"text,omitempty"`
	Internal bool   `json:"internal"`
}

type WebExtractLinksResult struct {
	URL        string    `json:"url"`
	TotalFound int       `json:"total_found"`
	Links      []WebLink `json:"links"`
}

type WebExtractMetadataTool struct{}

func NewWebExtractMetadataTool() *WebExtractMetadataTool {
	return &WebExtractMetadataTool{}
}

func (t *WebExtractMetadataTool) Name() string {
	return "web_extract_metadata"
}

func (t *WebExtractMetadataTool) Description() string {
	return "Extracts metadata from a web page including title, description, Open Graph tags, Twitter Card tags, JSON-LD structured data, author, publication date, and more. Useful for understanding page context without reading full content."
}

func (t *WebExtractMetadataTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to extract metadata from",
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebExtractMetadataTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return nil, fmt.Errorf("url is required")
	}

	htmlContent, finalURL, err := fetchHTML(ctx, urlStr)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	meta := WebMetadata{
		URL: finalURL,
	}

	meta.Title = doc.Find("title").First().Text()

	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		name = strings.ToLower(name)
		property = strings.ToLower(property)

		switch {
		case name == "description" || property == "description":
			meta.Description = content
		case name == "author" || property == "author":
			meta.Author = content
		case name == "keywords":
			meta.Keywords = strings.Split(content, ",")
			for i := range meta.Keywords {
				meta.Keywords[i] = strings.TrimSpace(meta.Keywords[i])
			}
		case property == "og:title":
			meta.OpenGraph.Title = content
		case property == "og:description":
			meta.OpenGraph.Description = content
		case property == "og:image":
			meta.OpenGraph.Image = content
		case property == "og:url":
			meta.OpenGraph.URL = content
		case property == "og:type":
			meta.OpenGraph.Type = content
		case property == "og:site_name":
			meta.OpenGraph.SiteName = content
		case name == "twitter:card":
			meta.TwitterCard.Card = content
		case name == "twitter:title":
			meta.TwitterCard.Title = content
		case name == "twitter:description":
			meta.TwitterCard.Description = content
		case name == "twitter:image":
			meta.TwitterCard.Image = content
		case name == "twitter:site":
			meta.TwitterCard.Site = content
		case name == "robots":
			meta.Robots = content
		case property == "article:published_time":
			meta.PublishedTime = content
		case property == "article:modified_time":
			meta.ModifiedTime = content
		}
	})

	if canonical, exists := doc.Find("link[rel='canonical']").Attr("href"); exists {
		meta.Canonical = canonical
	}

	if lang, exists := doc.Find("html").Attr("lang"); exists {
		meta.Language = lang
	}

	doc.Find("link[rel='icon'], link[rel='shortcut icon']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && meta.Favicon == "" {
			meta.Favicon = href
		}
	})

	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		meta.JSONLD = append(meta.JSONLD, s.Text())
	})

	return meta, nil
}

type WebMetadata struct {
	URL           string        `json:"url"`
	Title         string        `json:"title,omitempty"`
	Description   string        `json:"description,omitempty"`
	Author        string        `json:"author,omitempty"`
	Keywords      []string      `json:"keywords,omitempty"`
	Canonical     string        `json:"canonical,omitempty"`
	Language      string        `json:"language,omitempty"`
	Favicon       string        `json:"favicon,omitempty"`
	PublishedTime string        `json:"published_time,omitempty"`
	ModifiedTime  string        `json:"modified_time,omitempty"`
	Robots        string        `json:"robots,omitempty"`
	OpenGraph     OpenGraphData `json:"open_graph,omitempty"`
	TwitterCard   TwitterData   `json:"twitter_card,omitempty"`
	JSONLD        []string      `json:"json_ld,omitempty"`
}

type OpenGraphData struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
}

type TwitterData struct {
	Card        string `json:"card,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Site        string `json:"site,omitempty"`
}
