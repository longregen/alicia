package langfuse

import (
	"context"
	"sync"
	"time"
)

// Default cache configuration
const (
	defaultRefreshInterval = 1 * time.Minute
	defaultMaxAge          = 24 * time.Hour
)

// cacheEntry holds a cached prompt with timing information.
type cacheEntry struct {
	prompt    *Prompt
	fetchedAt time.Time
}

// promptCache implements stale-while-revalidate caching for prompts.
type promptCache struct {
	client          *Client
	entries         sync.Map // map[string]*cacheEntry
	refreshInterval time.Duration
	maxAge          time.Duration
	refreshing      sync.Map // map[string]bool - tracks ongoing refreshes
}

// newPromptCache creates a new prompt cache.
func newPromptCache(client *Client) *promptCache {
	return &promptCache{
		client:          client,
		refreshInterval: defaultRefreshInterval,
		maxAge:          defaultMaxAge,
	}
}

func (c *promptCache) setRefreshInterval(d time.Duration) {
	c.refreshInterval = d
}

func (c *promptCache) setMaxAge(d time.Duration) {
	c.maxAge = d
}

func (c *promptCache) get(key string) (*Prompt, bool) {
	value, ok := c.entries.Load(key)
	if !ok {
		return nil, false
	}

	entry, ok := value.(*cacheEntry)
	if !ok {
		return nil, false
	}

	// Check if entry is expired (beyond max age)
	if time.Since(entry.fetchedAt) > c.maxAge {
		c.entries.Delete(key)
		return nil, false
	}

	return entry.prompt, true
}

func (c *promptCache) set(key string, prompt *Prompt) {
	c.entries.Store(key, &cacheEntry{
		prompt:    prompt,
		fetchedAt: time.Now(),
	})
}

// isStale returns true if the cached entry is older than the refresh interval.
func (c *promptCache) isStale(key string) bool {
	value, ok := c.entries.Load(key)
	if !ok {
		return true
	}

	entry, ok := value.(*cacheEntry)
	if !ok {
		return true
	}
	return time.Since(entry.fetchedAt) > c.refreshInterval
}

// triggerRefreshIfStale starts a background refresh if the entry is stale.
// Only one refresh per key will run at a time.
func (c *promptCache) triggerRefreshIfStale(key, name string, cfg *options) {
	if !c.isStale(key) {
		return
	}

	// Check if already refreshing
	if _, loaded := c.refreshing.LoadOrStore(key, true); loaded {
		return // Already refreshing
	}

	// Start background refresh
	go func() {
		defer c.refreshing.Delete(key)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		prompt, err := c.client.fetchPrompt(ctx, name, cfg)
		if err != nil {
			c.client.log.Printf("langfuse: background refresh failed for %s: %v", name, err)
			return
		}

		c.set(key, prompt)
	}()
}

// Preload fetches and caches a prompt synchronously.
// Useful for warming the cache at startup.
func (c *promptCache) Preload(ctx context.Context, name string, opts ...Option) error {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(cfg)
	}

	key := c.client.buildCacheKey(name, cfg)
	prompt, err := c.client.fetchPrompt(ctx, name, cfg)
	if err != nil {
		return err
	}

	c.set(key, prompt)
	return nil
}
