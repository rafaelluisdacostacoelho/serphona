package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
)

// HTTPUsagePublisher posts usage events to an HTTP endpoint with retry.
type HTTPUsagePublisher struct {
	client   *http.Client
	endpoint string
	token    string
	retryMax uint
	backoff  time.Duration
	breaker  *gobreaker.CircuitBreaker
}

var (
	usagePublishAttempts *prometheus.CounterVec
	usagePublishBreaker  *prometheus.GaugeVec
	usageMetricsOnce     sync.Once
)

func initUsageMetrics() {
	usageMetricsOnce.Do(func() {
		usagePublishAttempts = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tools_gateway_usage_publish_attempts_total",
				Help: "Usage publish attempts by outcome",
			},
			[]string{"sink", "target", "outcome", "status"},
		)

		usagePublishBreaker = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tools_gateway_usage_publish_breaker_state",
				Help: "Circuit breaker state for usage publisher (0 closed, 1 open)",
			},
			[]string{"sink", "target"},
		)

		prometheus.MustRegister(usagePublishAttempts)
		prometheus.MustRegister(usagePublishBreaker)
	})
}

// NewHTTPUsagePublisher builds an HTTP usage publisher.
func NewHTTPUsagePublisher(
	endpoint, token string,
	timeout time.Duration,
	retryMax uint,
	backoff time.Duration,
	breakerEnabled bool,
	breakerFailures uint,
	breakerReset time.Duration,
) UsagePublisher {
	if endpoint == "" {
		return NewNoopUsagePublisher()
	}
	initUsageMetrics()

	var breaker *gobreaker.CircuitBreaker
	if breakerEnabled {
		if breakerFailures == 0 {
			breakerFailures = 3
		}
		bf := int(breakerFailures)
		breaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name: "usage_publisher",
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return int(counts.ConsecutiveFailures) >= bf
			},
			Interval: breakerReset,
		})
	}

	return &HTTPUsagePublisher{
		client:   &http.Client{Timeout: timeout},
		endpoint: endpoint,
		token:    token,
		retryMax: retryMax,
		backoff:  backoff,
		breaker:  breaker,
	}
}

// PublishUsage sends the usage event as JSON.
func (p *HTTPUsagePublisher) PublishUsage(ctx context.Context, evt UsageEvent) error {
	if p == nil || p.endpoint == "" {
		return nil
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal usage event: %w", err)
	}

	return retry.Do(
		func() error {
			req, err := p.buildRequest(ctx, payload)
			if err != nil {
				return err
			}
			return p.sendWithBreaker(req)
		},
		retry.Attempts(p.retryMax),
		retry.Delay(p.backoff),
		retry.Context(ctx),
	)
}

func (p *HTTPUsagePublisher) buildRequest(ctx context.Context, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("build usage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	return req, nil
}

func (p *HTTPUsagePublisher) sendWithBreaker(req *http.Request) error {
	doReq := func() error {
		usagePublishAttempts.WithLabelValues("http", p.endpoint, "attempt", "").Inc()
		resp, err := p.client.Do(req)
		if err != nil {
			usagePublishAttempts.WithLabelValues("http", p.endpoint, "transport_error", "").Inc()
			return err
		}
		defer resp.Body.Close()

		switch {
		case resp.StatusCode >= 500:
			usagePublishAttempts.WithLabelValues("http", p.endpoint, "retryable", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			log.Printf("usage_publish sink=http outcome=retryable status=%d target=%s", resp.StatusCode, p.endpoint)
			return fmt.Errorf("usage publish: upstream error %d", resp.StatusCode)
		case resp.StatusCode == http.StatusTooManyRequests:
			usagePublishAttempts.WithLabelValues("http", p.endpoint, "rate_limited", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			log.Printf("usage_publish sink=http outcome=rate_limited status=%d target=%s", resp.StatusCode, p.endpoint)
			return retry.Unrecoverable(fmt.Errorf("usage publish: rate limited %d", resp.StatusCode))
		case resp.StatusCode >= 400:
			usagePublishAttempts.WithLabelValues("http", p.endpoint, "client_error", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			log.Printf("usage_publish sink=http outcome=client_error status=%d target=%s", resp.StatusCode, p.endpoint)
			return retry.Unrecoverable(fmt.Errorf("usage publish: bad request %d", resp.StatusCode))
		default:
			usagePublishAttempts.WithLabelValues("http", p.endpoint, "success", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			return nil
		}
	}

	if p.breaker != nil {
		_, err := p.breaker.Execute(func() (interface{}, error) {
			return nil, doReq()
		})
		if err == gobreaker.ErrOpenState {
			usagePublishBreaker.WithLabelValues("http", p.endpoint).Set(1)
			usagePublishAttempts.WithLabelValues("http", p.endpoint, "breaker_open", "").Inc()
			log.Printf("usage_publish sink=http outcome=breaker_open target=%s", p.endpoint)
		}
		if err == nil {
			usagePublishBreaker.WithLabelValues("http", p.endpoint).Set(0)
		}
		return err
	}

	return doReq()
}
