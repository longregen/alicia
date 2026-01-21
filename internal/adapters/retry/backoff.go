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

func HTTPConfig() BackoffConfig {
	return BackoffConfig{
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxRetries:      3,
		Multiplier:      2.0,
	}
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// IsNotFound indicates a definitive NXDOMAIN, which shouldn't be retried
		return !dnsErr.IsNotFound
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return true
		}
		if errors.Is(opErr.Err, syscall.ECONNRESET) {
			return true
		}
		if errors.Is(opErr.Err, syscall.EPIPE) {
			return true
		}
	}

	return false
}

func IsRetryableHTTPStatus(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}

	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	if statusCode == http.StatusRequestTimeout {
		return true
	}

	return false
}

func ShouldRetry(err error, statusCode int) bool {
	if err == nil && statusCode > 0 {
		return IsRetryableHTTPStatus(statusCode)
	}
	return IsRetryableError(err)
}

func WithBackoff(ctx context.Context, cfg BackoffConfig, fn func() error) error {
	var lastErr error
	interval := cfg.InitialInterval

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
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

func WithBackoffHTTP(ctx context.Context, cfg BackoffConfig, fn func() (int, error)) error {
	var lastErr error
	var lastStatus int
	interval := cfg.InitialInterval

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		statusCode, err := fn()
		lastStatus = statusCode
		lastErr = err

		if err == nil && statusCode >= 200 && statusCode < 300 {
			return nil
		}

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
