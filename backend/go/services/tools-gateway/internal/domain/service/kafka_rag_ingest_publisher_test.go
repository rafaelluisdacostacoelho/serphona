package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
	"github.com/segmentio/kafka-go"
)

type fakeIngestWriter struct {
	msgs   int
	errSeq []error
}

func (f *fakeIngestWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	f.msgs += len(msgs)
	if len(f.errSeq) > 0 {
		err := f.errSeq[0]
		f.errSeq = f.errSeq[1:]
		return err
	}
	return nil
}

func (f *fakeIngestWriter) Close() error { return nil }

func TestKafkaRAGIngestionPublisherRetriesThenSucceeds(t *testing.T) {
	fw := &fakeIngestWriter{errSeq: []error{errors.New("boom"), nil}}
	pub := &KafkaRAGIngestionPublisher{writer: fw, backoff: time.Millisecond, retryMax: 2, topic: "rag.ingestion.requested"}
	evt := events.RAGIngestionRequestedEvent{TenantID: "t1", DocumentID: "doc", Namespace: "ns"}
	if err := pub.PublishIngestion(context.Background(), evt); err != nil {
		t.Fatalf("expected success after retry: %v", err)
	}
	if fw.msgs != 2 {
		t.Fatalf("expected 2 attempts, got %d", fw.msgs)
	}
}

func TestKafkaRAGIngestionPublisherNoWriter(t *testing.T) {
	pub := &KafkaRAGIngestionPublisher{}
	if err := pub.PublishIngestion(context.Background(), events.RAGIngestionRequestedEvent{}); err != nil {
		t.Fatalf("expected nil when writer missing, got %v", err)
	}
}
