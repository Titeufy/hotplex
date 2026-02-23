package chatapps

import (
	"context"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			delay := config.BaseDelay * time.Duration(1<<attempt)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		return nil
	}
	return lastErr
}

// IsRetryableError classifies errors as retryable or non-retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	// Non-retryable: auth, validation, client errors
	nonRetryable := []string{"401", "403", "404", "422", "unauthorized", "forbidden", "not found", "validation", "invalid", "malformed"}
	for _, n := range nonRetryable {
		if strings.Contains(errStr, n) {
			return false
		}
	}

	// Retryable: timeouts, rate limits, server errors
	retryable := []string{"timeout", "temporary", "unavailable", "429", "500", "502", "503", "504", "rate limit", "too many requests", "server error", "connection refused", "connection reset", "i/o timeout"}
	for _, r := range retryable {
		if strings.Contains(errStr, r) {
			return true
		}
	}

	// Default: retry (conservative)
	return true
}
