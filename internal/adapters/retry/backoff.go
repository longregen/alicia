package retry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

type BackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxRetries      int
	Multiplier      float64
}

func DefaultConfig() BackoffConfig {
	return BackoffConfig{
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxRetries:      3,
		Multiplier:      2.0,
	}
}

// HTTPConfig returns a configuration suitable for HTTP requests
func HTTPConfig() BackoffConfig {
	return BackoffConfig{
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxRetries:      3,
		Multiplier:      2.0,
	}
}

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors should not be retried
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors (timeout errors)
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Retry on timeout errors
		if netErr.Timeout() {
			return true
		}
	}

	// DNS errors (temporary lookup failures)
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// Retry on DNS lookup failures that aren't permanent
		// IsNotFound indicates a definitive NXDOMAIN, which shouldn't be retried
		return !dnsErr.IsNotFound
	}

	// Connection errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Retry connection refused (service might be temporarily down)
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return true
		}
		// Retry connection reset
		if errors.Is(opErr.Err, syscall.ECONNRESET) {
			return true
		}
		// Retry broken pipe
		if errors.Is(opErr.Err, syscall.EPIPE) {
			return true
		}
	}

	return false
}

// IsRetryableHTTPStatus determines if an HTTP status code should be retried
func IsRetryableHTTPStatus(statusCode int) bool {
	// 429 Too Many Requests - rate limiting
	if statusCode == http.StatusTooManyRequests {
		return true
	}

	// 5xx Server errors - temporary server issues
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	// 408 Request Timeout
	if statusCode == http.StatusRequestTimeout {
		return true
	}

	// 503 Service Unavailable, 502 Bad Gateway, 504 Gateway Timeout are already covered by 5xx

	return false
}

// ShouldRetry combines error checking and optional HTTP status checking
func ShouldRetry(err error, statusCode int) bool {
	if err == nil && statusCode > 0 {
		return IsRetryableHTTPStatus(statusCode)
	}
	return IsRetryableError(err)
}

// WithBackoff executes the function with exponential backoff retry logic
func WithBackoff(ctx context.Context, cfg BackoffConfig, fn func() error) error {
	var lastErr error
	interval := cfg.InitialInterval

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			// Check if we should retry this error
			if !IsRetryableError(err) {
				return fmt.Errorf("non-retryable error on attempt %d: %w", attempt+1, err)
			}
		}

		if attempt == cfg.MaxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		interval = time.Duration(float64(interval) * cfg.Multiplier)
		if interval > cfg.MaxInterval {
			interval = cfg.MaxInterval
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxRetries, lastErr)
}

// WithBackoffHTTP executes the function with exponential backoff retry logic for HTTP requests
// The function should return both an error and an HTTP status code
func WithBackoffHTTP(ctx context.Context, cfg BackoffConfig, fn func() (int, error)) error {
	var lastErr error
	var lastStatus int
	interval := cfg.InitialInterval

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		statusCode, err := fn()
		lastStatus = statusCode
		lastErr = err

		// Success case: no error and 2xx status code
		if err == nil && statusCode >= 200 && statusCode < 300 {
			return nil
		}

		// Check if we should retry
		shouldRetry := false
		if err != nil {
			shouldRetry = IsRetryableError(err)
		} else if statusCode > 0 {
			shouldRetry = IsRetryableHTTPStatus(statusCode)
		}

		if !shouldRetry {
			if err != nil {
				return fmt.Errorf("non-retryable error on attempt %d (status %d): %w", attempt+1, statusCode, err)
			}
			return fmt.Errorf("non-retryable status code %d on attempt %d", statusCode, attempt+1)
		}

		if attempt == cfg.MaxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		interval = time.Duration(float64(interval) * cfg.Multiplier)
		if interval > cfg.MaxInterval {
			interval = cfg.MaxInterval
		}
	}

	if lastErr != nil {
		return fmt.Errorf("max retries (%d) exceeded (status %d): %w", cfg.MaxRetries, lastStatus, lastErr)
	}
	return fmt.Errorf("max retries (%d) exceeded with status code %d", cfg.MaxRetries, lastStatus)
}
