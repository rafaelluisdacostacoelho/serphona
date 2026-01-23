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

type quotaFake struct{ called int }

func (q *quotaFake) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	q.called++
	return [][]float32{{1}}, nil
}

func TestQuotaClientBlocksDailyTokens(t *testing.T) {
	inner := &quotaFake{}
	qc := &QuotaClient{Inner: inner, Config: QuotaConfig{TokensPerDay: 10, TPS: 10, Now: func() time.Time {
		return time.Unix(0, 0)
	}}}

	ctx := WithEmbeddingQuotaScope(context.Background(), "t1", "ns")
	if _, err := qc.Embed(ctx, []string{"aaaaaaaaaaaaaaaaaaaa"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := qc.Embed(ctx, []string{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
	if inner.called != 1 {
		t.Fatalf("expected 1 inner call, got %d", inner.called)
	}
}

func TestQuotaClientBlocksTps(t *testing.T) {
	inner := &quotaFake{}
	qc := &QuotaClient{Inner: inner, Config: QuotaConfig{TokensPerDay: 1000, TPS: 1, Now: func() time.Time {
		return time.Unix(0, 0)
	}}}

	ctx := WithEmbeddingQuotaScope(context.Background(), "t1", "ns")
	if _, err := qc.Embed(ctx, []string{"a"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := qc.Embed(ctx, []string{"b"}); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
	if inner.called != 1 {
		t.Fatalf("expected 1 inner call, got %d", inner.called)
	}
}
