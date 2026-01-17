package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/longregen/alicia/cmd/mcp-web/security"
)

// BrowserPool manages Rod browser instances
type BrowserPool struct {
	mu          sync.Mutex
	browser     *rod.Browser
	initialized bool
}

var (
	defaultPool *BrowserPool
	poolOnce    sync.Once
)

// GetBrowserPool returns the singleton browser pool
func GetBrowserPool() *BrowserPool {
	poolOnce.Do(func() {
		defaultPool = &BrowserPool{}
	})
	return defaultPool
}

// Initialize starts the headless browser
func (p *BrowserPool) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	// Find or download Chrome/Chromium
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).Headless(true).MustLaunch()

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	p.browser = browser
	p.initialized = true

	return nil
}

// Close shuts down the browser pool
func (p *BrowserPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil
	}

	if p.browser != nil {
		p.browser.Close()
	}

	p.initialized = false
	return nil
}

// WaitStrategy defines how to wait for page readiness
type WaitStrategy string

const (
	WaitLoad             WaitStrategy = "load"
	WaitDOMContentLoaded WaitStrategy = "domcontentloaded"
	WaitNetworkIdle      WaitStrategy = "networkidle"
)

// RenderOptions configures browser rendering
type RenderOptions struct {
	WaitFor   WaitStrategy
	WaitMS    int
	Width     int
	Height    int
	UserAgent string
}

// DefaultRenderOptions returns sensible defaults
func DefaultRenderOptions() *RenderOptions {
	return &RenderOptions{
		WaitFor:   WaitLoad,
		WaitMS:    0,
		Width:     1280,
		Height:    720,
		UserAgent: DefaultUserAgent,
	}
}

// RenderResult holds the result of browser rendering
type RenderResult struct {
	HTML     string
	FinalURL string
}

// RenderPage fetches a URL using a headless browser with JavaScript execution
func (p *BrowserPool) RenderPage(ctx context.Context, url string, opts *RenderOptions) (*RenderResult, error) {
	// Validate URL for SSRF protection
	if err := security.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	if err := p.Initialize(); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultRenderOptions()
	}

	p.mu.Lock()
	browser := p.browser
	p.mu.Unlock()

	// Create a new page
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Set viewport
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  opts.Width,
		Height: opts.Height,
	}); err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	// Set user agent
	if opts.UserAgent != "" {
		if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: opts.UserAgent,
		}); err != nil {
			return nil, fmt.Errorf("failed to set user agent: %w", err)
		}
	}

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait based on strategy
	switch opts.WaitFor {
	case WaitDOMContentLoaded:
		if err := page.WaitDOMStable(time.Second, 0.5); err != nil {
			return nil, fmt.Errorf("failed waiting for DOM: %w", err)
		}
	case WaitNetworkIdle:
		wait := page.WaitRequestIdle(time.Second, nil, nil, nil)
		wait() // Wait for network to be idle
	default: // WaitLoad
		if err := page.WaitLoad(); err != nil {
			return nil, fmt.Errorf("failed waiting for load: %w", err)
		}
	}

	// Additional wait if specified
	if opts.WaitMS > 0 {
		time.Sleep(time.Duration(opts.WaitMS) * time.Millisecond)
	}

	// Get the rendered HTML
	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	return &RenderResult{
		HTML:     html,
		FinalURL: page.MustInfo().URL,
	}, nil
}

// ScreenshotOptions configures screenshot capture
type ScreenshotOptions struct {
	Width    int
	Height   int
	FullPage bool
	WaitMS   int
	WaitFor  WaitStrategy
}

// DefaultScreenshotOptions returns sensible defaults
func DefaultScreenshotOptions() *ScreenshotOptions {
	return &ScreenshotOptions{
		Width:    1280,
		Height:   720,
		FullPage: false,
		WaitMS:   1000,
		WaitFor:  WaitLoad,
	}
}

// CaptureScreenshot takes a screenshot of a URL
func (p *BrowserPool) CaptureScreenshot(ctx context.Context, url string, opts *ScreenshotOptions) ([]byte, error) {
	// Validate URL for SSRF protection
	if err := security.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	if err := p.Initialize(); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultScreenshotOptions()
	}

	p.mu.Lock()
	browser := p.browser
	p.mu.Unlock()

	// Create a new page
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Set viewport
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  opts.Width,
		Height: opts.Height,
	}); err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait based on strategy
	switch opts.WaitFor {
	case WaitDOMContentLoaded:
		if err := page.WaitDOMStable(time.Second, 0.5); err != nil {
			return nil, fmt.Errorf("failed waiting for DOM: %w", err)
		}
	case WaitNetworkIdle:
		wait := page.WaitRequestIdle(time.Second, nil, nil, nil)
		wait() // Wait for network to be idle
	default: // WaitLoad
		if err := page.WaitLoad(); err != nil {
			return nil, fmt.Errorf("failed waiting for load: %w", err)
		}
	}

	// Additional wait
	if opts.WaitMS > 0 {
		time.Sleep(time.Duration(opts.WaitMS) * time.Millisecond)
	}

	// Capture screenshot
	var screenshot []byte
	if opts.FullPage {
		screenshot, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	} else {
		screenshot, err = page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	return screenshot, nil
}

// FetchWithJS fetches HTML content using a headless browser
// This is useful for pages that require JavaScript to render
func FetchWithJS(ctx context.Context, url string, waitFor WaitStrategy, waitMS int) (string, string, error) {
	pool := GetBrowserPool()

	result, err := pool.RenderPage(ctx, url, &RenderOptions{
		WaitFor: waitFor,
		WaitMS:  waitMS,
	})
	if err != nil {
		return "", "", err
	}

	return result.HTML, result.FinalURL, nil
}
