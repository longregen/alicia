package pipeline

import (
	"encoding/json"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Metadata holds extracted page metadata
type Metadata struct {
	Title         string            `json:"title,omitempty"`
	Description   string            `json:"description,omitempty"`
	Author        string            `json:"author,omitempty"`
	Keywords      []string          `json:"keywords,omitempty"`
	CanonicalURL  string            `json:"canonical_url,omitempty"`
	PublishedTime string            `json:"published_time,omitempty"`
	ModifiedTime  string            `json:"modified_time,omitempty"`
	OpenGraph     map[string]string `json:"open_graph,omitempty"`
	TwitterCard   map[string]string `json:"twitter_card,omitempty"`
	JSONLD        []any             `json:"json_ld,omitempty"`
	Favicon       string            `json:"favicon,omitempty"`
	Language      string            `json:"language,omitempty"`
	Robots        string            `json:"robots,omitempty"`
}

// ExtractMetadata extracts all metadata from HTML
func ExtractMetadata(htmlContent string, baseURL string) (*Metadata, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	meta := &Metadata{
		OpenGraph:   make(map[string]string),
		TwitterCard: make(map[string]string),
	}

	// Extract from <html> tag
	htmlTag := findElement(doc, "html")
	if htmlTag != nil {
		for _, attr := range htmlTag.Attr {
			if attr.Key == "lang" {
				meta.Language = attr.Val
			}
		}
	}

	// Extract <title>
	titleNode := findElement(doc, "title")
	if titleNode != nil && titleNode.FirstChild != nil {
		meta.Title = strings.TrimSpace(titleNode.FirstChild.Data)
	}

	// Extract meta tags and links
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "meta":
				extractMetaTag(n, meta)
			case "link":
				extractLinkTag(n, meta, baseURL)
			case "script":
				extractJSONLD(n, meta)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	// Use og:title as fallback if title is empty
	if meta.Title == "" && meta.OpenGraph["title"] != "" {
		meta.Title = meta.OpenGraph["title"]
	}

	// Use og:description as fallback
	if meta.Description == "" && meta.OpenGraph["description"] != "" {
		meta.Description = meta.OpenGraph["description"]
	}

	return meta, nil
}

func extractMetaTag(node *html.Node, meta *Metadata) {
	var name, property, content, httpEquiv string
	for _, attr := range node.Attr {
		switch attr.Key {
		case "name":
			name = strings.ToLower(attr.Val)
		case "property":
			property = strings.ToLower(attr.Val)
		case "content":
			content = attr.Val
		case "http-equiv":
			httpEquiv = strings.ToLower(attr.Val)
		}
	}

	// Standard meta tags
	switch name {
	case "description":
		meta.Description = content
	case "author":
		meta.Author = content
	case "keywords":
		meta.Keywords = parseKeywords(content)
	case "robots":
		meta.Robots = content
	}

	// Open Graph
	if strings.HasPrefix(property, "og:") {
		key := strings.TrimPrefix(property, "og:")
		meta.OpenGraph[key] = content

		switch key {
		case "title":
			if meta.Title == "" {
				meta.Title = content
			}
		case "description":
			if meta.Description == "" {
				meta.Description = content
			}
		}
	}

	// Article metadata
	switch property {
	case "article:author":
		if meta.Author == "" {
			meta.Author = content
		}
	case "article:published_time":
		meta.PublishedTime = content
	case "article:modified_time":
		meta.ModifiedTime = content
	}

	// Twitter Card
	if strings.HasPrefix(name, "twitter:") {
		key := strings.TrimPrefix(name, "twitter:")
		meta.TwitterCard[key] = content
	}

	// Content-Language
	if httpEquiv == "content-language" && meta.Language == "" {
		meta.Language = content
	}
}

func extractLinkTag(node *html.Node, meta *Metadata, baseURL string) {
	var rel, href string
	for _, attr := range node.Attr {
		switch attr.Key {
		case "rel":
			rel = strings.ToLower(attr.Val)
		case "href":
			href = attr.Val
		}
	}

	switch rel {
	case "canonical":
		meta.CanonicalURL = resolveURL(href, baseURL)
	case "icon", "shortcut icon":
		if meta.Favicon == "" {
			meta.Favicon = resolveURL(href, baseURL)
		}
	}
}

func extractJSONLD(node *html.Node, meta *Metadata) {
	// Check if it's a JSON-LD script
	var isJSONLD bool
	for _, attr := range node.Attr {
		if attr.Key == "type" && attr.Val == "application/ld+json" {
			isJSONLD = true
			break
		}
	}

	if !isJSONLD {
		return
	}

	// Extract script content
	if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
		content := strings.TrimSpace(node.FirstChild.Data)
		if content != "" {
			var data any
			if err := json.Unmarshal([]byte(content), &data); err == nil {
				meta.JSONLD = append(meta.JSONLD, data)

				// Extract common fields from JSON-LD
				extractFromJSONLD(data, meta)
			}
		}
	}
}

func extractFromJSONLD(data any, meta *Metadata) {
	switch v := data.(type) {
	case map[string]any:
		// Check @type
		if typeVal, ok := v["@type"].(string); ok {
			switch typeVal {
			case "Article", "NewsArticle", "BlogPosting":
				if headline, ok := v["headline"].(string); ok && meta.Title == "" {
					meta.Title = headline
				}
				if desc, ok := v["description"].(string); ok && meta.Description == "" {
					meta.Description = desc
				}
				if datePublished, ok := v["datePublished"].(string); ok && meta.PublishedTime == "" {
					meta.PublishedTime = datePublished
				}
				if dateModified, ok := v["dateModified"].(string); ok && meta.ModifiedTime == "" {
					meta.ModifiedTime = dateModified
				}
				if author, ok := v["author"]; ok && meta.Author == "" {
					meta.Author = extractAuthorFromJSONLD(author)
				}
			}
		}

		// Recurse into @graph
		if graph, ok := v["@graph"].([]any); ok {
			for _, item := range graph {
				extractFromJSONLD(item, meta)
			}
		}
	case []any:
		for _, item := range v {
			extractFromJSONLD(item, meta)
		}
	}
}

func extractAuthorFromJSONLD(author any) string {
	switch v := author.(type) {
	case string:
		return v
	case map[string]any:
		if name, ok := v["name"].(string); ok {
			return name
		}
	case []any:
		if len(v) > 0 {
			return extractAuthorFromJSONLD(v[0])
		}
	}
	return ""
}

func parseKeywords(content string) []string {
	if content == "" {
		return nil
	}

	// Split by comma and clean up
	parts := strings.Split(content, ",")
	keywords := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keywords = append(keywords, p)
		}
	}
	return keywords
}

// findElement finds the first element with the given tag name
func findElement(doc *html.Node, tag string) *html.Node {
	var result *html.Node
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)
	return result
}

func resolveURL(href, baseURL string) string {
	if baseURL == "" {
		return href
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		// Get scheme and host from base URL
		re := regexp.MustCompile(`^(https?://[^/]+)`)
		if match := re.FindString(baseURL); match != "" {
			return match + href
		}
	}
	// Relative URL
	if strings.HasSuffix(baseURL, "/") {
		return baseURL + href
	}
	if idx := strings.LastIndex(baseURL, "/"); idx > 7 { // After "http://" or "https://"
		return baseURL[:idx+1] + href
	}
	return href
}
