package embedding

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubClient struct {
	outputs [][]float32
	errs    []error
	calls   int
}

func (s *stubClient) Embed(_ context.Context, _ []string) ([][]float32, error) {
	defer func() { s.calls++ }()
	if s.calls < len(s.errs) && s.errs[s.calls] != nil {
		return nil, s.errs[s.calls]
	}
	return s.outputs, nil
}

func TestRetryableClientSuccessAfterRetry(t *testing.T) {
	stub := &stubClient{outputs: [][]float32{{1}}, errs: []error{errors.New("boom")}}
	client := RetryableClient{Inner: stub, MaxRetries: 2, BackoffMin: time.Nanosecond, BackoffMax: time.Nanosecond}

	res, err := client.Embed(context.Background(), []string{"hi"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(res) != 1 || stub.calls != 2 {
		t.Fatalf("expected retry path, calls=%d res=%v", stub.calls, res)
	}
}

func TestRetryableClientNoInner(t *testing.T) {
	client := RetryableClient{}
	if _, err := client.Embed(context.Background(), []string{"hi"}); err != ErrNoInnerClient {
		t.Fatalf("expected ErrNoInnerClient, got %v", err)
	}
}

func TestRetryableClientExhaustsRetries(t *testing.T) {
	stub := &stubClient{errs: []error{errors.New("boom"), errors.New("boom")}}
	client := RetryableClient{Inner: stub, MaxRetries: 2, BackoffMin: time.Nanosecond, BackoffMax: time.Nanosecond}
	if _, err := client.Embed(context.Background(), []string{"hi"}); err == nil {
		t.Fatalf("expected final error")
	}
	if stub.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", stub.calls)
	}
}

func TestRetryableClientDefaultsApplied(t *testing.T) {
	stub := &stubClient{outputs: [][]float32{{1}}}
	client := RetryableClient{Inner: stub} // zero values trigger defaults

	res, err := client.Embed(context.Background(), []string{"ok"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(res) != 1 || stub.calls != 1 {
		t.Fatalf("expected single call with defaults, calls=%d", stub.calls)
	}
}
