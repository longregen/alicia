package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/longregen/alicia/cmd/mcp-web/pipeline"
	"github.com/longregen/alicia/cmd/mcp-web/security"
)

// ScreenshotTool captures screenshots of web pages using Playwright
type ScreenshotTool struct{}

func NewScreenshotTool() *ScreenshotTool {
	return &ScreenshotTool{}
}

func (t *ScreenshotTool) Name() string {
	return "screenshot"
}

func (t *ScreenshotTool) Description() string {
	return "Captures a screenshot of a web page using a headless Chromium browser. Supports full-page screenshots, custom viewport sizes, and waiting for JavaScript to render. Returns base64-encoded PNG image data."
}

func (t *ScreenshotTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to capture",
			},
			"width": map[string]any{
				"type":        "integer",
				"description": "Viewport width in pixels (default: 1280)",
				"default":     1280,
			},
			"height": map[string]any{
				"type":        "integer",
				"description": "Viewport height in pixels (default: 720)",
				"default":     720,
			},
			"full_page": map[string]any{
				"type":        "boolean",
				"description": "Capture full scrollable page instead of just the viewport (default: false)",
				"default":     false,
			},
			"wait_for": map[string]any{
				"type":        "string",
				"description": "Wait strategy: 'load' (default), 'domcontentloaded', or 'networkidle'",
				"enum":        []string{"load", "domcontentloaded", "networkidle"},
				"default":     "load",
			},
			"wait_ms": map[string]any{
				"type":        "integer",
				"description": "Additional milliseconds to wait after page load before capture (default: 1000)",
				"default":     1000,
			},
		},
		"required": []string{"url"},
	}
}

func (t *ScreenshotTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	targetURL, ok := args["url"].(string)
	if !ok || targetURL == "" {
		return "", fmt.Errorf("url is required")
	}

	// Validate URL for SSRF protection
	if err := security.ValidateURL(targetURL); err != nil {
		return "", fmt.Errorf("URL validation failed: %w", err)
	}

	// Parse options
	width := 1280
	if v, ok := args["width"].(float64); ok {
		width = int(v)
	}

	height := 720
	if v, ok := args["height"].(float64); ok {
		height = int(v)
	}

	fullPage := false
	if v, ok := args["full_page"].(bool); ok {
		fullPage = v
	}

	waitFor := pipeline.WaitLoad
	if v, ok := args["wait_for"].(string); ok {
		switch v {
		case "domcontentloaded":
			waitFor = pipeline.WaitDOMContentLoaded
		case "networkidle":
			waitFor = pipeline.WaitNetworkIdle
		}
	}

	waitMS := 1000
	if v, ok := args["wait_ms"].(float64); ok {
		waitMS = int(v)
	}

	// Capture screenshot
	pool := pipeline.GetBrowserPool()
	screenshot, err := pool.CaptureScreenshot(ctx, targetURL, &pipeline.ScreenshotOptions{
		Width:    width,
		Height:   height,
		FullPage: fullPage,
		WaitFor:  waitFor,
		WaitMS:   waitMS,
	})
	if err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Build result
	result := ScreenshotResult{
		URL:      targetURL,
		Width:    width,
		Height:   height,
		FullPage: fullPage,
		Format:   "png",
		Size:     len(screenshot),
		Data:     base64.StdEncoding.EncodeToString(screenshot),
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(output), nil
}

// ScreenshotResult is the output of the screenshot tool
type ScreenshotResult struct {
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FullPage bool   `json:"full_page"`
	Format   string `json:"format"`
	Size     int    `json:"size_bytes"`
	Data     string `json:"data"`
}
