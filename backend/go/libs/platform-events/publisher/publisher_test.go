package publisher

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/events"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// fakeWriter captures messages written without hitting a broker.
type fakeWriter struct {
	msgs   []kafka.Message
	err    error
	closed int
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.msgs = append(f.msgs, msgs...)
	return f.err
}

func (f *fakeWriter) Stats() kafka.WriterStats { return kafka.WriterStats{Writes: int64(len(f.msgs))} }
func (f *fakeWriter) Close() error             { f.closed++; return nil }

// stub dialer path by replacing DialLeader usage with writer.WriteMessages in test via helper.
func TestPublishSetsRequiredAndOptionalHeaders(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ServiceName = "svc-test"
	cfg.ClientID = "client-test"
	cfg.GroupID = "group-test"

	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	// Swap writer for fake to avoid network
	fw := &fakeWriter{}
	pub.writer = fw
	pub.ready[topics.UserCreated] = true // skip ensureTopic network path

	ctx := context.Background()
	evt := events.NewEvent(topics.UserCreated, "auth-gateway", events.UserCreatedEvent{UserID: "u1"}).
		WithTenantID("tenant-1").WithUserID("user-1").WithTrace("trace-1", "span-1")

	if err := pub.Publish(ctx, topics.UserCreated, evt); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	headers := map[string]string{}
	for _, h := range fw.msgs[0].Headers {
		headers[h.Key] = string(h.Value)
	}

	// Required
	for _, key := range []string{"event_type", "source", "version"} {
		if headers[key] == "" {
			t.Fatalf("missing required header %s", key)
		}
	}

	// Optional
	if headers["tenant_id"] != "tenant-1" {
		t.Fatalf("tenant_id header mismatch: %s", headers["tenant_id"])
	}
	if headers["user_id"] != "user-1" {
		t.Fatalf("user_id header mismatch: %s", headers["user_id"])
	}
	if headers["trace_id"] != "trace-1" || headers["span_id"] != "span-1" {
		t.Fatalf("trace headers mismatch: trace_id=%s span_id=%s", headers["trace_id"], headers["span_id"])
	}
}

func TestPublishOmitsOptionalHeadersWhenEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ServiceName = "svc-test"
	cfg.ClientID = "client-test"
	cfg.GroupID = "group-test"

	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	fw := &fakeWriter{}
	pub.writer = fw
	pub.ready[topics.UserCreated] = true

	evt := events.NewEvent(topics.UserCreated, "auth-gateway", events.UserCreatedEvent{UserID: "u1"})
	if err := pub.Publish(context.Background(), topics.UserCreated, evt); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	headers := map[string]string{}
	for _, h := range fw.msgs[0].Headers {
		headers[h.Key] = string(h.Value)
	}

	if headers["tenant_id"] != "" || headers["user_id"] != "" || headers["trace_id"] != "" || headers["span_id"] != "" {
		t.Fatalf("optional headers should be empty when not set: %+v", headers)
	}
}

func TestStatsReturnsWriterStats(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ServiceName = "svc-test"
	cfg.ClientID = "client-test"
	cfg.GroupID = "group-test"

	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	fw := &fakeWriter{}
	pub.writer = fw
	pub.ready[topics.UserCreated] = true

	stats := pub.Stats()
	if stats.Messages != 0 {
		t.Fatalf("expected zeroed stats, got %v", stats.Messages)
	}
}

func TestPublishBatchWritesAllMessages(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	pub, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	pub.ready["topic"] = true

	eventsBatch := []*types.Event{
		events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u1"}),
		events.NewEvent(topics.UserUpdated, "src", events.UserUpdatedEvent{UserID: "u2"}),
	}

	if err := pub.PublishBatch(context.Background(), "topic", eventsBatch); err != nil {
		t.Fatalf("publish batch returned error: %v", err)
	}

	if len(fw.msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(fw.msgs))
	}
}

func TestPublishReturnsErrorWhenClosed(t *testing.T) {
	p := &Publisher{
		writer: &fakeWriter{},
		ready:  make(map[string]bool),
		config: config.DefaultConfig(),
		closed: true,
	}

	err := p.Publish(context.Background(), "topic", events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"}))
	if err != ErrPublisherClosed {
		t.Fatalf("expected ErrPublisherClosed, got %v", err)
	}
}

type fakeTopicConn struct {
	createCalled bool
	readCalled   bool
	partitions   []kafka.Partition
}

func (f *fakeTopicConn) CreateTopics(...kafka.TopicConfig) error { f.createCalled = true; return nil }
func (f *fakeTopicConn) ReadPartitions(...string) ([]kafka.Partition, error) {
	f.readCalled = true
	return f.partitions, nil
}
func (f *fakeTopicConn) SetDeadline(time.Time) error { return nil }
func (f *fakeTopicConn) Close() error                { return nil }

type fakeLeaderConn struct{ closed bool }

func (f *fakeLeaderConn) Close() error { f.closed = true; return nil }

func TestEnsureTopicMarksReady(t *testing.T) {
	originalDialer := newDialer
	originalDialContext := dialContext
	originalDialLeader := dialLeader
	defer func() {
		newDialer = originalDialer
		dialContext = originalDialContext
		dialLeader = originalDialLeader
	}()

	fc := &fakeTopicConn{partitions: []kafka.Partition{{}}}
	newDialer = func(*config.Config) (*kafka.Dialer, error) { return &kafka.Dialer{}, nil }
	dialContext = func(*kafka.Dialer, context.Context, string, string) (topicConn, error) { return fc, nil }
	dialLeader = func(*kafka.Dialer, context.Context, string, string, string, int) (leaderConn, error) {
		return &fakeLeaderConn{}, nil
	}

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}

	if err := p.ensureTopic(context.Background(), "t1"); err != nil {
		t.Fatalf("ensureTopic failed: %v", err)
	}

	if !fc.createCalled || !fc.readCalled {
		t.Fatalf("expected create and read partitions to be called")
	}
	if !p.ready["t1"] {
		t.Fatalf("topic should be marked ready")
	}
}

func TestEnsureTopicFailsWithoutBrokers(t *testing.T) {
	p := &Publisher{config: &config.Config{Brokers: []string{}}, ready: make(map[string]bool), writer: &fakeWriter{}}

	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected error for missing brokers")
	}
}

func TestNewDialerTLSAndSASL(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UseTLS = true
	cfg.TLSInsecureSkipVerify = true
	cfg.SASLMechanism = "plain"
	cfg.SASLUsername = "u"
	cfg.SASLPassword = "p"

	d, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.TLS == nil || !d.TLS.InsecureSkipVerify {
		t.Fatalf("tls not configured")
	}
	if d.SASLMechanism == nil {
		t.Fatalf("sasl not configured")
	}
}

func TestPublishBatchEmptyReturnsNil(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	pub, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	pub.ready["topic"] = true

	if err := pub.PublishBatch(context.Background(), "topic", nil); err != nil {
		t.Fatalf("expected nil error for empty batch, got %v", err)
	}
	if len(fw.msgs) != 0 {
		t.Fatalf("no messages should be written on empty batch")
	}
}

func TestPublishPropagatesWriterError(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{err: errors.New("fail")}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	pub, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	pub.ready["topic"] = true

	err = pub.Publish(context.Background(), "topic", events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"}))
	if err == nil {
		t.Fatalf("expected writer error")
	}
}

func TestPublisherCloseIdempotent(t *testing.T) {
	p := &Publisher{writer: &fakeWriter{}, ready: make(map[string]bool), config: config.DefaultConfig()}

	if err := p.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second close should be nil: %v", err)
	}

	fw := p.writer.(*fakeWriter)
	if fw.closed != 1 {
		t.Fatalf("writer should be closed once, got %d", fw.closed)
	}
}

func TestNewTransportUsesDialer(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) {
		return &kafka.Dialer{
			DialFunc: func(context.Context, string, string) (net.Conn, error) {
				c1, c2 := net.Pipe()
				_ = c2.Close()
				return c1, nil
			},
		}, nil
	}

	tr := newTransport(config.DefaultConfig())
	conn, err := tr.Dial(context.Background(), "tcp", "addr")
	if err != nil {
		t.Fatalf("dial should succeed with fake dialer: %v", err)
	}
	_ = conn.Close()
}

func TestEnsureTopicSkipsWhenReady(t *testing.T) {
	p := &Publisher{config: config.DefaultConfig(), ready: map[string]bool{"t1": true}, writer: &fakeWriter{}}

	if err := p.ensureTopic(context.Background(), "t1"); err != nil {
		t.Fatalf("expected no error when topic already ready: %v", err)
	}
}

func TestEnsureTopicErrorsWhenNoPartitions(t *testing.T) {
	originalDialer := newDialer
	originalDialContext := dialContext
	defer func() {
		newDialer = originalDialer
		dialContext = originalDialContext
	}()

	fc := &fakeTopicConn{partitions: []kafka.Partition{}}
	newDialer = func(*config.Config) (*kafka.Dialer, error) { return &kafka.Dialer{}, nil }
	dialContext = func(*kafka.Dialer, context.Context, string, string) (topicConn, error) { return fc, nil }

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}
	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected error for missing partitions")
	}
}

func TestNewWithWriterError(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	newWriter = func(*config.Config) (writerInterface, error) { return nil, errors.New("boom") }

	if _, err := New(config.DefaultConfig()); err == nil {
		t.Fatalf("expected error when writer fails")
	}
}

func TestStatsWhenWriterNil(t *testing.T) {
	p := &Publisher{config: config.DefaultConfig()}
	stats := p.Stats()
	if stats.Writes != 0 {
		t.Fatalf("expected zero stats when writer nil")
	}
}

func TestPublishBatchSerializationError(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	pub, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	pub.ready["topic"] = true

	bad := events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"})
	bad.Data = make(chan int)

	if err := pub.PublishBatch(context.Background(), "topic", []*types.Event{bad}); err == nil {
		t.Fatalf("expected serialization error")
	}
}

func TestNewTransportFallbackOnDialerError(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) { return nil, errors.New("fail") }

	tr := newTransport(config.DefaultConfig())
	if tr == nil {
		t.Fatalf("transport should not be nil even on dialer error")
	}
}
