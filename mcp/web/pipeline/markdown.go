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

// TruncateContent truncates content at a sensible boundary (paragraph, sentence,
// or word) and appends the given suffix. If content is already within maxLen,
// it is returned unchanged.
func TruncateContent(content string, maxLen int, suffix string) string {
	if len(content) <= maxLen {
		return content
	}

	truncated := content[:maxLen]

	// Try to truncate at paragraph boundary
	if idx := strings.LastIndex(truncated, "\n\n"); idx > maxLen/2 {
		return truncated[:idx] + suffix
	}

	// Fall back to sentence boundary
	if idx := strings.LastIndex(truncated, ". "); idx > maxLen/2 {
		return truncated[:idx+1] + suffix
	}

	// Last resort: word boundary
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		return truncated[:idx] + suffix
	}

	return truncated + suffix
}
