package consumer

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/config"
)

// fakeReader minimal Stats provider to avoid network calls.
type fakeReader struct{}

func (f *fakeReader) FetchMessage(context.Context) (kafka.Message, error) {
	return kafka.Message{}, nil
}
func (f *fakeReader) CommitMessages(context.Context, ...kafka.Message) error { return nil }
func (f *fakeReader) Stats() kafka.ReaderStats                               { return kafka.ReaderStats{Messages: 42} }
func (f *fakeReader) Close() error                                           { return nil }

func TestStatsReturnsUnderlyingReaderStats(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.GroupID = "g"
	cfg.ClientID = "c"
	cfg.ServiceName = "s"

	c := &Consumer{
		reader: &fakeReader{},
		config: cfg,
	}

	stats := c.Stats()
	if stats.Messages != 42 {
		t.Fatalf("expected stats.Messages = 42, got %d", stats.Messages)
	}
}
