package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPUsagePublisherRetriesOn500ThenSuccess(t *testing.T) {
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)

	pub := NewHTTPUsagePublisher(ts.URL, "", 2*time.Second, 3, 10*time.Millisecond, false, 0, 0)
	err := pub.PublishUsage(context.Background(), UsageEvent{})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests (500 then 200), got %d", hits)
	}
}

func TestHTTPUsagePublisherNoRetryOn429(t *testing.T) {
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(ts.Close)

	pub := NewHTTPUsagePublisher(ts.URL, "", 2*time.Second, 3, 10*time.Millisecond, false, 0, 0)
	err := pub.PublishUsage(context.Background(), UsageEvent{})
	if err == nil {
		t.Fatalf("expected error for 429 response")
	}
	if hits != 1 {
		t.Fatalf("expected no retries on 429, got %d hits", hits)
	}
}

func TestHTTPUsagePublisherCircuitBreakerOpens(t *testing.T) {
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(ts.Close)

	pub := NewHTTPUsagePublisher(ts.URL, "", 2*time.Second, 2, 10*time.Millisecond, true, 1, 10*time.Second)

	// First call should attempt and fail, opening the breaker on next execution.
	if err := pub.PublishUsage(context.Background(), UsageEvent{}); err == nil {
		t.Fatalf("expected error on first failure")
	}

	// Second call should be blocked immediately by breaker (no additional hit).
	err := pub.PublishUsage(context.Background(), UsageEvent{})
	if err == nil {
		t.Fatalf("expected breaker to block second call")
	}
	if hits != 1 {
		t.Fatalf("expected only one upstream call, got %d", hits)
	}
}
