package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/longregen/alicia/mcp/web/pipeline"
)

// ExtractMetadataTool extracts page metadata
type ExtractMetadataTool struct{}

func NewExtractMetadataTool() *ExtractMetadataTool {
	return &ExtractMetadataTool{}
}

func (t *ExtractMetadataTool) Name() string {
	return "extract_metadata"
}

func (t *ExtractMetadataTool) Description() string {
	return "Extracts metadata from a web page including title, description, Open Graph tags, Twitter Card tags, JSON-LD structured data, author, publication date, and more. Useful for understanding page context without reading full content."
}

func (t *ExtractMetadataTool) InputSchema() map[string]any {
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

func (t *ExtractMetadataTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	targetURL, ok := args["url"].(string)
	if !ok || targetURL == "" {
		return "", fmt.Errorf("url is required")
	}

	// Fetch the HTML
	htmlContent, finalURL, err := pipeline.FetchHTML(ctx, targetURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Extract metadata
	meta, err := pipeline.ExtractMetadata(htmlContent, finalURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract metadata: %w", err)
	}

	output := MetadataOutput{
		URL:      finalURL,
		Metadata: meta,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	return string(data), nil
}

// MetadataOutput wraps the metadata with the URL
type MetadataOutput struct {
	URL      string             `json:"url"`
	Metadata *pipeline.Metadata `json:"metadata"`
}
