package pipeline

import (
	"bytes"
	"net/url"
	"strings"
	"time"

	"codeberg.org/readeck/go-readability/v2"
)

// ContentResult holds the extracted content and metadata
type ContentResult struct {
	Title         string
	Content       string
	TextContent   string
	Byline        string
	Excerpt       string
	SiteName      string
	WordCount     int
	PublishedTime *time.Time
	Image         string
	Favicon       string
	Language      string
}

// ExtractContent extracts main content from HTML using Mozilla's Readability algorithm
func ExtractContent(htmlContent string, pageURL string) (*ContentResult, error) {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		parsedURL = &url.URL{Scheme: "https", Host: "example.com"}
	}

	// Parse with go-readability
	article, err := readability.FromReader(strings.NewReader(htmlContent), parsedURL)
	if err != nil {
		return nil, err
	}

	// Render HTML content
	var htmlBuf bytes.Buffer
	if err := article.RenderHTML(&htmlBuf); err != nil {
		return nil, err
	}

	// Render text content
	var textBuf bytes.Buffer
	if err := article.RenderText(&textBuf); err != nil {
		return nil, err
	}

	result := &ContentResult{
		Title:       article.Title(),
		Content:     htmlBuf.String(),
		TextContent: textBuf.String(),
		Byline:      article.Byline(),
		Excerpt:     article.Excerpt(),
		SiteName:    article.SiteName(),
		Favicon:     article.Favicon(),
		Image:       article.ImageURL(),
		Language:    article.Language(),
	}

	// Set published time if available
	if pubTime, err := article.PublishedTime(); err == nil && !pubTime.IsZero() {
		result.PublishedTime = &pubTime
	}

	// Calculate word count
	result.WordCount = len(strings.Fields(result.TextContent))

	return result, nil
}
