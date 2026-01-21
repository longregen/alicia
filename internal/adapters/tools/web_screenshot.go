package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type WebScreenshotTool struct{}

func NewWebScreenshotTool() *WebScreenshotTool {
	return &WebScreenshotTool{}
}

func (t *WebScreenshotTool) Name() string {
	return "web_screenshot"
}

func (t *WebScreenshotTool) Description() string {
	return "Captures a screenshot of a web page. Returns the screenshot as base64-encoded PNG data. Requires chromium or google-chrome to be installed on the system."
}

func (t *WebScreenshotTool) Schema() map[string]any {
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
			"delay_ms": map[string]any{
				"type":        "integer",
				"description": "Milliseconds to wait after page load before capture (default: 1000)",
				"default":     1000,
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebScreenshotTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return nil, fmt.Errorf("url is required")
	}

	width := 1280
	if v, ok := args["width"].(float64); ok && v > 0 {
		width = int(v)
	}

	height := 720
	if v, ok := args["height"].(float64); ok && v > 0 {
		height = int(v)
	}

	fullPage := false
	if v, ok := args["full_page"].(bool); ok {
		fullPage = v
	}

	delayMS := 1000
	if v, ok := args["delay_ms"].(float64); ok && v >= 0 {
		delayMS = int(v)
	}

	chromePath := findChrome()
	if chromePath == "" {
		return nil, fmt.Errorf("chromium or google-chrome not found. Please install chromium-browser or google-chrome")
	}

	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, fmt.Sprintf("screenshot_%d.png", time.Now().UnixNano()))
	defer os.Remove(outputPath)

	chromeArgs := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-software-rasterizer",
		fmt.Sprintf("--window-size=%d,%d", width, height),
		fmt.Sprintf("--screenshot=%s", outputPath),
	}

	if fullPage {
		chromeArgs = append(chromeArgs, "--full-page-screenshot")
	}

	chromeArgs = append(chromeArgs,
		fmt.Sprintf("--virtual-time-budget=%d", 5000+delayMS),
	)

	chromeArgs = append(chromeArgs, urlStr)

	cmd := exec.CommandContext(ctx, chromePath, chromeArgs...)
	cmd.Env = append(os.Environ(), "DISPLAY=:99") // Support headless X

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("chrome failed: %w - output: %s", err, string(output))
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot: %w", err)
	}

	result := WebScreenshotResult{
		URL:    urlStr,
		Width:  width,
		Height: height,
		Format: "png",
		Data:   base64.StdEncoding.EncodeToString(data),
		Size:   len(data),
	}

	return result, nil
}

type WebScreenshotResult struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
	Data   string `json:"data"`
	Size   int    `json:"size_bytes"`
}

func findChrome() string {
	candidates := []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/snap/bin/chromium",
	}

	for _, path := range candidates {
		if p, err := exec.LookPath(path); err == nil {
			return p
		}
	}

	return ""
}
