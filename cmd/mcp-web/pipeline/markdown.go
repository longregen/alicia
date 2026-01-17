package pipeline

import (
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

// HTMLToMarkdown converts HTML content to Markdown
func HTMLToMarkdown(htmlContent string, baseURL string) (string, error) {
	md, err := htmltomarkdown.ConvertString(
		htmlContent,
		converter.WithDomain(baseURL),
	)
	if err != nil {
		return "", err
	}

	// Clean up excessive whitespace
	md = cleanMarkdown(md)

	return md, nil
}

// cleanMarkdown cleans up the markdown output
func cleanMarkdown(md string) string {
	// Remove excessive blank lines (more than 2)
	lines := strings.Split(md, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, "")
			}
		} else {
			blankCount = 0
			result = append(result, strings.TrimRight(line, " \t"))
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// MarkdownOptions configures markdown conversion
type MarkdownOptions struct {
	MaxLength     int
	IncludeLinks  bool
	IncludeImages bool
}

// HTMLToMarkdownWithOptions converts HTML to Markdown with custom options
func HTMLToMarkdownWithOptions(htmlContent string, baseURL string, opts MarkdownOptions) (string, error) {
	md, err := htmltomarkdown.ConvertString(
		htmlContent,
		converter.WithDomain(baseURL),
	)
	if err != nil {
		return "", err
	}

	md = cleanMarkdown(md)

	// Apply max length if specified
	if opts.MaxLength > 0 && len(md) > opts.MaxLength {
		md = truncateMarkdown(md, opts.MaxLength)
	}

	return md, nil
}

// truncateMarkdown truncates markdown at a sensible boundary
func truncateMarkdown(md string, maxLen int) string {
	if len(md) <= maxLen {
		return md
	}

	// Try to truncate at paragraph boundary
	truncated := md[:maxLen]
	if idx := strings.LastIndex(truncated, "\n\n"); idx > maxLen/2 {
		return truncated[:idx] + "\n\n[content truncated...]"
	}

	// Fall back to sentence boundary
	if idx := strings.LastIndex(truncated, ". "); idx > maxLen/2 {
		return truncated[:idx+1] + "\n\n[content truncated...]"
	}

	// Last resort: word boundary
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		return truncated[:idx] + "...\n\n[content truncated...]"
	}

	return truncated + "..."
}
