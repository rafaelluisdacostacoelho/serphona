package embedding

import (
	"context"
	"errors"
	"time"
)

// Client computes embeddings for given inputs.
type Client interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

// RetryableClient decorates a Client with retries/backoff for transient failures.
type RetryableClient struct {
	Inner      Client
	MaxRetries int
	BackoffMin time.Duration
	BackoffMax time.Duration
}

// Embed proxies to the inner client with simple exponential backoff.
func (r RetryableClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if r.Inner == nil {
		return nil, ErrNoInnerClient
	}
	if r.MaxRetries <= 0 {
		r.MaxRetries = 3
	}
	if r.BackoffMin <= 0 {
		r.BackoffMin = 100 * time.Millisecond
	}
	if r.BackoffMax <= 0 {
		r.BackoffMax = 2 * time.Second
	}

	backoff := r.BackoffMin
	var lastErr error
	for attempt := 0; attempt < r.MaxRetries; attempt++ {
		res, err := r.Inner.Embed(ctx, inputs)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if attempt == r.MaxRetries-1 {
			break
		}
		t := backoff
		backoff *= 2
		if backoff > r.BackoffMax {
			backoff = r.BackoffMax
		}
		time.Sleep(t)
	}
	return nil, lastErr
}

// ErrNoInnerClient indicates the decorator has no inner client set.
var ErrNoInnerClient = errors.New("retryable client missing inner client")
