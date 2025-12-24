package retry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "connection refused",
			err:      &net.OpError{Err: syscall.ECONNREFUSED},
			expected: true,
		},
		{
			name:     "connection reset",
			err:      &net.OpError{Err: syscall.ECONNRESET},
			expected: true,
		},
		{
			name:     "broken pipe",
			err:      &net.OpError{Err: syscall.EPIPE},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			expected:   false,
		},
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			expected:   false,
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			expected:   false,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
		{
			name:       "408 Request Timeout",
			statusCode: http.StatusRequestTimeout,
			expected:   true,
		},
		{
			name:       "429 Too Many Requests",
			statusCode: http.StatusTooManyRequests,
			expected:   true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "502 Bad Gateway",
			statusCode: http.StatusBadGateway,
			expected:   true,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			expected:   true,
		},
		{
			name:       "504 Gateway Timeout",
			statusCode: http.StatusGatewayTimeout,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableHTTPStatus(tt.statusCode)
			if result != tt.expected {
				t.Errorf("IsRetryableHTTPStatus(%d) = %v, want %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestWithBackoff_Success(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	ctx := context.Background()
	err := WithBackoff(ctx, cfg, fn)

	if err != nil {
		t.Errorf("WithBackoff() error = %v, want nil", err)
	}

	if attempts != 1 {
		t.Errorf("WithBackoff() attempts = %d, want 1", attempts)
	}
}

func TestWithBackoff_RetryableError(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return &net.OpError{Err: syscall.ECONNREFUSED}
		}
		return nil
	}

	ctx := context.Background()
	err := WithBackoff(ctx, cfg, fn)

	if err != nil {
		t.Errorf("WithBackoff() error = %v, want nil", err)
	}

	if attempts != 3 {
		t.Errorf("WithBackoff() attempts = %d, want 3", attempts)
	}
}

func TestWithBackoff_NonRetryableError(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	expectedErr := errors.New("non-retryable error")
	fn := func() error {
		attempts++
		return expectedErr
	}

	ctx := context.Background()
	err := WithBackoff(ctx, cfg, fn)

	if err == nil {
		t.Error("WithBackoff() error = nil, want non-nil")
	}

	if attempts != 1 {
		t.Errorf("WithBackoff() attempts = %d, want 1 (should not retry non-retryable errors)", attempts)
	}
}

func TestWithBackoff_MaxRetriesExceeded(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return &net.OpError{Err: syscall.ECONNREFUSED}
	}

	ctx := context.Background()
	err := WithBackoff(ctx, cfg, fn)

	if err == nil {
		t.Error("WithBackoff() error = nil, want non-nil")
	}

	// Should attempt 4 times (initial + 3 retries)
	expectedAttempts := cfg.MaxRetries + 1
	if attempts != expectedAttempts {
		t.Errorf("WithBackoff() attempts = %d, want %d", attempts, expectedAttempts)
	}
}

func TestWithBackoff_ContextCanceled(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetries:      5,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return &net.OpError{Err: syscall.ECONNREFUSED}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := WithBackoff(ctx, cfg, fn)

	if err != context.Canceled {
		t.Errorf("WithBackoff() error = %v, want context.Canceled", err)
	}

	// Should have attempted at least once
	if attempts < 1 {
		t.Errorf("WithBackoff() attempts = %d, want at least 1", attempts)
	}
}

func TestWithBackoffHTTP_Success(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() (int, error) {
		attempts++
		return http.StatusOK, nil
	}

	ctx := context.Background()
	err := WithBackoffHTTP(ctx, cfg, fn)

	if err != nil {
		t.Errorf("WithBackoffHTTP() error = %v, want nil", err)
	}

	if attempts != 1 {
		t.Errorf("WithBackoffHTTP() attempts = %d, want 1", attempts)
	}
}

func TestWithBackoffHTTP_RetryableStatus(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() (int, error) {
		attempts++
		if attempts < 3 {
			return http.StatusServiceUnavailable, nil
		}
		return http.StatusOK, nil
	}

	ctx := context.Background()
	err := WithBackoffHTTP(ctx, cfg, fn)

	if err != nil {
		t.Errorf("WithBackoffHTTP() error = %v, want nil", err)
	}

	if attempts != 3 {
		t.Errorf("WithBackoffHTTP() attempts = %d, want 3", attempts)
	}
}

func TestWithBackoffHTTP_NonRetryableStatus(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() (int, error) {
		attempts++
		return http.StatusBadRequest, nil
	}

	ctx := context.Background()
	err := WithBackoffHTTP(ctx, cfg, fn)

	if err == nil {
		t.Error("WithBackoffHTTP() error = nil, want non-nil")
	}

	if attempts != 1 {
		t.Errorf("WithBackoffHTTP() attempts = %d, want 1 (should not retry 4xx errors)", attempts)
	}
}

func TestWithBackoffHTTP_RetryableErrorThenSuccess(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() (int, error) {
		attempts++
		if attempts < 2 {
			return 0, &net.OpError{Err: syscall.ECONNREFUSED}
		}
		return http.StatusOK, nil
	}

	ctx := context.Background()
	err := WithBackoffHTTP(ctx, cfg, fn)

	if err != nil {
		t.Errorf("WithBackoffHTTP() error = %v, want nil", err)
	}

	if attempts != 2 {
		t.Errorf("WithBackoffHTTP() attempts = %d, want 2", attempts)
	}
}

func TestWithBackoffHTTP_MaxRetriesWithStatus(t *testing.T) {
	cfg := BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxRetries:      3,
		Multiplier:      2.0,
	}

	attempts := 0
	fn := func() (int, error) {
		attempts++
		return http.StatusInternalServerError, nil
	}

	ctx := context.Background()
	err := WithBackoffHTTP(ctx, cfg, fn)

	if err == nil {
		t.Error("WithBackoffHTTP() error = nil, want non-nil")
	}

	// Should attempt 4 times (initial + 3 retries)
	expectedAttempts := cfg.MaxRetries + 1
	if attempts != expectedAttempts {
		t.Errorf("WithBackoffHTTP() attempts = %d, want %d", attempts, expectedAttempts)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.InitialInterval != 1*time.Second {
		t.Errorf("DefaultConfig().InitialInterval = %v, want 1s", cfg.InitialInterval)
	}

	if cfg.MaxInterval != 30*time.Second {
		t.Errorf("DefaultConfig().MaxInterval = %v, want 30s", cfg.MaxInterval)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultConfig().MaxRetries = %d, want 3", cfg.MaxRetries)
	}

	if cfg.Multiplier != 2.0 {
		t.Errorf("DefaultConfig().Multiplier = %f, want 2.0", cfg.Multiplier)
	}
}

func TestHTTPConfig(t *testing.T) {
	cfg := HTTPConfig()

	if cfg.InitialInterval != 1*time.Second {
		t.Errorf("HTTPConfig().InitialInterval = %v, want 1s", cfg.InitialInterval)
	}

	if cfg.MaxInterval != 30*time.Second {
		t.Errorf("HTTPConfig().MaxInterval = %v, want 30s", cfg.MaxInterval)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("HTTPConfig().MaxRetries = %d, want 3", cfg.MaxRetries)
	}

	if cfg.Multiplier != 2.0 {
		t.Errorf("HTTPConfig().Multiplier = %f, want 2.0", cfg.Multiplier)
	}
}
