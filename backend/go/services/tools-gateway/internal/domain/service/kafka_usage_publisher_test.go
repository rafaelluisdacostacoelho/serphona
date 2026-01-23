package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeKafkaWriter struct {
	calls  int
	errSeq []error
}

func (f *fakeKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	f.calls++
	if len(f.errSeq) == 0 {
		return nil
	}
	err := f.errSeq[0]
	if len(f.errSeq) > 1 {
		f.errSeq = f.errSeq[1:]
	}
	return err
}

func (f *fakeKafkaWriter) Close() error { return nil }

func TestKafkaPublisherRetriesThenSucceeds(t *testing.T) {
	fw := &fakeKafkaWriter{errSeq: []error{errors.New("boom"), errors.New("boom"), nil}}
	pub := &KafkaUsagePublisher{writer: fw, backoff: 1 * time.Millisecond, retryMax: 3, topic: "tools.usage"}

	if err := pub.PublishUsage(context.Background(), UsageEvent{}); err != nil {
		t.Fatalf("expected success after retry: %v", err)
	}
	if fw.calls < 2 {
		t.Fatalf("expected at least 3 attempts, got %d", fw.calls)
	}
}

func TestKafkaPublisherStopsOnPermanentError(t *testing.T) {
	fw := &fakeKafkaWriter{errSeq: []error{errors.New("fail"), errors.New("fail")}}
	pub := &KafkaUsagePublisher{writer: fw, backoff: 1 * time.Millisecond, retryMax: 2, topic: "tools.usage"}

	if err := pub.PublishUsage(context.Background(), UsageEvent{}); err == nil {
		t.Fatalf("expected error on permanent failure")
	}
	if fw.calls != 2 { // initial + 1 retry
		t.Fatalf("expected 2 attempts, got %d", fw.calls)
	}
}
