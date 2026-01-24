package langfuse

import "time"

// options holds configuration for GetPrompt calls.
type options struct {
	label   string
	version int
}

// Option configures a GetPrompt call.
type Option func(*options)

// defaultOptions returns the default options.
func defaultOptions() *options {
	return &options{
		label: "production",
	}
}

// WithLabel sets the label to filter prompts by.
// Default is "production".
func WithLabel(label string) Option {
	return func(o *options) {
		o.label = label
	}
}

// WithVersion sets a specific version to fetch.
// If set, label is ignored.
func WithVersion(version int) Option {
	return func(o *options) {
		o.version = version
	}
}

// WithoutLabel removes the default label filter.
func WithoutLabel() Option {
	return func(o *options) {
		o.label = ""
	}
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithHTTPTimeout sets the HTTP client timeout.
func WithHTTPTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithRefreshInterval sets how often cached entries should be refreshed.
func WithRefreshInterval(d time.Duration) ClientOption {
	return func(c *Client) {
		c.cache.setRefreshInterval(d)
	}
}

// WithMaxAge sets the maximum age for cached entries.
func WithMaxAge(d time.Duration) ClientOption {
	return func(c *Client) {
		c.cache.setMaxAge(d)
	}
}

// NewWithOptions creates a new client with additional configuration options.
func NewWithOptions(host, publicKey, secretKey string, opts ...ClientOption) *Client {
	c := New(host, publicKey, secretKey)
	for _, opt := range opts {
		opt(c)
	}
	return c
}
