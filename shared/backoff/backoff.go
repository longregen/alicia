// Package backoff provides exponential backoff utilities for retry logic.
package backoff

import (
	"context"
	"fmt"
	"time"
)

type Strategy struct {
	Delays []time.Duration
}

var (
	Quick = Strategy{
		Delays: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			16 * time.Second,
		},
	}

	Standard = Strategy{
		Delays: []time.Duration{
			500 * time.Millisecond,
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			16 * time.Second,
			30 * time.Second,
		},
	}

	Aggressive = Strategy{
		Delays: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			16 * time.Second,
			30 * time.Second,
			60 * time.Second,
			120 * time.Second,
			240 * time.Second,
			300 * time.Second,
		},
	}
)

type RetryFunc func(ctx context.Context, attempt int) error

func Retry(ctx context.Context, strategy Strategy, fn RetryFunc) error {
	var lastErr error

	for i := 0; i < len(strategy.Delays); i++ {
		if err := fn(ctx, i+1); err != nil {
			lastErr = err

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(strategy.Delays[i]):
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", len(strategy.Delays), lastErr)
}

func RetryWithCallback(ctx context.Context, strategy Strategy, fn RetryFunc, onRetry func(attempt int, err error, delay time.Duration)) error {
	var lastErr error

	for i := 0; i < len(strategy.Delays); i++ {
		if err := fn(ctx, i+1); err != nil {
			lastErr = err

			if onRetry != nil {
				onRetry(i+1, err, strategy.Delays[i])
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(strategy.Delays[i]):
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", len(strategy.Delays), lastErr)
}

func Custom(delays ...time.Duration) Strategy {
	return Strategy{Delays: delays}
}

func Exponential(initial time.Duration, multiplier float64, maxRetries int) Strategy {
	delays := make([]time.Duration, maxRetries)
	current := initial
	for i := 0; i < maxRetries; i++ {
		delays[i] = current
		current = time.Duration(float64(current) * multiplier)
	}
	return Strategy{Delays: delays}
}
