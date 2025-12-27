package embedding

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeClient struct {
	calls int
	fail  int
}

func (f *fakeClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	f.calls++
	if f.fail > 0 {
		f.fail--
		return nil, errors.New("temporary")
	}
	return [][]float32{{1}}, nil
}

func TestRetryableClientRetries(t *testing.T) {
	fc := &fakeClient{fail: 1}
	r := RetryableClient{Inner: fc, MaxRetries: 3, BackoffMin: time.Millisecond, BackoffMax: 2 * time.Millisecond}
	if _, err := r.Embed(context.Background(), []string{"x"}); err != nil {
		t.Fatalf("expected success after retry: %v", err)
	}
	if fc.calls != 2 {
		t.Fatalf("expected 2 calls (1 fail + 1 success), got %d", fc.calls)
	}
}

func TestRetryableClientMissingInner(t *testing.T) {
	r := RetryableClient{}
	if _, err := r.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatalf("expected error for missing inner client")
	}
	if _, err := r.Embed(context.Background(), []string{"x"}); !errors.Is(err, ErrNoInnerClient) {
		t.Fatalf("expected ErrNoInnerClient, got %v", err)
	}
}
