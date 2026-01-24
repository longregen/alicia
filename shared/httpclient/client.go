// Package httpclient provides a shared HTTP client factory with common configurations.
package httpclient

import (
	"net/http"
	"time"
)

type Config struct {
	Timeout       time.Duration
	Transport     http.RoundTripper
	CheckRedirect func(req *http.Request, via []*http.Request) error
}

type Option func(*Config)

func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

// WithTransport sets a custom transport (e.g., for OTEL tracing).
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Config) {
		c.Transport = rt
	}
}

func WithMaxRedirects(max int) Option {
	return func(c *Config) {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= max {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}
}

func WithNoRedirects() Option {
	return func(c *Config) {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
}

func WithCheckRedirect(fn func(req *http.Request, via []*http.Request) error) Option {
	return func(c *Config) {
		c.CheckRedirect = fn
	}
}

const (
	TimeoutShort  = 10 * time.Second
	TimeoutMedium = 30 * time.Second
	TimeoutLong   = 60 * time.Second
)

func New(opts ...Option) *http.Client {
	cfg := &Config{
		Timeout:   TimeoutMedium,
		Transport: http.DefaultTransport,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &http.Client{
		Timeout:       cfg.Timeout,
		Transport:     cfg.Transport,
		CheckRedirect: cfg.CheckRedirect,
	}
}

func NewShort(opts ...Option) *http.Client {
	return New(append([]Option{WithTimeout(TimeoutShort)}, opts...)...)
}

func NewLong(opts ...Option) *http.Client {
	return New(append([]Option{WithTimeout(TimeoutLong)}, opts...)...)
}
