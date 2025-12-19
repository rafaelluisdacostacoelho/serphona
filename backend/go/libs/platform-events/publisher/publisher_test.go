package publisher

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/events"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// fakeWriter captures messages written without hitting a broker.
type fakeWriter struct {
	msgs     []kafka.Message
	err      error
	closeErr error
	closed   int
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.msgs = append(f.msgs, msgs...)
	return f.err
}

func (f *fakeWriter) Stats() kafka.WriterStats { return kafka.WriterStats{Writes: int64(len(f.msgs))} }
func (f *fakeWriter) Close() error             { f.closed++; return f.closeErr }

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

func TestPublishBatchOptionalHeaders(t *testing.T) {
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
		events.NewEvent("topic", "src", events.UserCreatedEvent{UserID: "u1"}).
			WithTenantID("tenant-1").WithUserID("user-1").WithTrace("trace-1", "span-1").WithMetadata("k", "v"),
		events.NewEvent("topic", "src", events.UserCreatedEvent{UserID: "u2"}),
	}

	if err := pub.PublishBatch(context.Background(), "topic", eventsBatch); err != nil {
		t.Fatalf("publish batch returned error: %v", err)
	}

	headers := map[string]string{}
	for _, h := range fw.msgs[0].Headers {
		headers[h.Key] = string(h.Value)
	}

	expected := map[string]string{
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"trace_id":  "trace-1",
		"span_id":   "span-1",
	}

	for k, v := range expected {
		if headers[k] != v {
			t.Fatalf("expected header %s=%s, got %s", k, v, headers[k])
		}
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
	createCalled  bool
	readCalled    bool
	partitions    []kafka.Partition
	partitionsErr error
	deadlineSet   bool
	createErr     error
}

func (f *fakeTopicConn) CreateTopics(...kafka.TopicConfig) error {
	f.createCalled = true
	return f.createErr
}
func (f *fakeTopicConn) ReadPartitions(...string) ([]kafka.Partition, error) {
	f.readCalled = true
	return f.partitions, f.partitionsErr
}
func (f *fakeTopicConn) SetDeadline(time.Time) error { f.deadlineSet = true; return nil }
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

func TestNewDialerScramMechanisms(t *testing.T) {
	cfg256 := config.DefaultConfig()
	cfg256.SASLMechanism = "scram-sha256"
	cfg256.SASLUsername = "u"
	cfg256.SASLPassword = "p"

	if _, err := newDialer(cfg256); err != nil {
		t.Fatalf("scram-sha256 dialer error: %v", err)
	}

	cfg512 := config.DefaultConfig()
	cfg512.SASLMechanism = "scram-sha512"
	cfg512.SASLUsername = "u"
	cfg512.SASLPassword = "p"

	if _, err := newDialer(cfg512); err != nil {
		t.Fatalf("scram-sha512 dialer error: %v", err)
	}
}

func TestNewDialerScramSHA512(t *testing.T) {
	originalScram := scramMechanism
	defer func() { scramMechanism = originalScram }()

	called := false
	scramMechanism = func(algo scram.Algorithm, user, pass string) (sasl.Mechanism, error) {
		if algo != scram.SHA512 {
			t.Fatalf("expected SHA512, got %v", algo.Name())
		}
		called = true
		return nil, nil
	}

	cfg := config.DefaultConfig()
	cfg.SASLMechanism = "scram-sha512"
	if _, err := newDialer(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatalf("scram mechanism not invoked")
	}
}

func TestNewDialerScramSHA512Error(t *testing.T) {
	originalScram := scramMechanism
	defer func() { scramMechanism = originalScram }()

	scramMechanism = func(algo scram.Algorithm, user, pass string) (sasl.Mechanism, error) {
		if algo != scram.SHA512 {
			t.Fatalf("expected SHA512, got %v", algo.Name())
		}
		return nil, errors.New("scram512 fail")
	}

	cfg := config.DefaultConfig()
	cfg.SASLMechanism = "scram-sha512"
	if _, err := newDialer(cfg); err == nil {
		t.Fatalf("expected scram error")
	}
}

func TestNewDialerScramError(t *testing.T) {
	originalScram := scramMechanism
	defer func() { scramMechanism = originalScram }()

	scramMechanism = func(scram.Algorithm, string, string) (sasl.Mechanism, error) {
		return nil, errors.New("scram fail")
	}

	cfg := config.DefaultConfig()
	cfg.SASLMechanism = "scram-sha256"
	if _, err := newDialer(cfg); err == nil {
		t.Fatalf("expected scram error")
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

func TestPublishSerializationError(t *testing.T) {
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

	if err := pub.Publish(context.Background(), "topic", bad); err == nil {
		t.Fatalf("expected serialization error")
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

func TestEnsureTopicCreateError(t *testing.T) {
	originalDialer := newDialer
	originalDialContext := dialContext
	defer func() {
		newDialer = originalDialer
		dialContext = originalDialContext
	}()

	fc := &fakeTopicConn{createErr: errors.New("boom")}
	newDialer = func(*config.Config) (*kafka.Dialer, error) { return &kafka.Dialer{}, nil }
	dialContext = func(*kafka.Dialer, context.Context, string, string) (topicConn, error) { return fc, nil }

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}
	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected create topic error")
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

func TestNewWriterDialerFailure(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) { return nil, errors.New("dialer fail") }

	if _, err := New(config.DefaultConfig()); err == nil {
		t.Fatalf("expected new dialer error to bubble up")
	}
}

func TestDialHelpersDefaultFunctions(t *testing.T) {
	d := &kafka.Dialer{Timeout: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	if _, err := dialContext(d, ctx, "tcp", "127.0.0.1:0"); err == nil {
		t.Fatalf("expected dialContext to error on invalid address")
	}

	if _, err := dialLeader(d, ctx, "tcp", "127.0.0.1:0", "t", 0); err == nil {
		t.Fatalf("expected dialLeader to error on invalid address")
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

func TestNewValidationFailure(t *testing.T) {
	bad := config.DefaultConfig()
	bad.Brokers = nil
	if _, err := New(bad); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestNewWithDebugFlag(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	w := &fakeWriter{}
	newWriter = func(*config.Config) (writerInterface, error) { return w, nil }

	cfg := config.DefaultConfig()
	cfg.Debug = true

	if _, err := New(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishEnsureTopicErrorNoBrokers(t *testing.T) {
	p := &Publisher{config: &config.Config{Brokers: []string{}}, ready: make(map[string]bool), writer: &fakeWriter{}}

	err := p.Publish(context.Background(), "topic", events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"}))
	if err == nil {
		t.Fatalf("expected error when no brokers")
	}
}

func TestPublishBatchWhenClosed(t *testing.T) {
	p := &Publisher{config: config.DefaultConfig(), writer: &fakeWriter{}, ready: make(map[string]bool), closed: true}

	if err := p.PublishBatch(context.Background(), "topic", []*types.Event{}); err != ErrPublisherClosed {
		t.Fatalf("expected ErrPublisherClosed, got %v", err)
	}
}

func TestPublishBatchWriterError(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{err: errors.New("write fail")}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	pub, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pub.ready["topic"] = true

	batch := []*types.Event{events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"})}
	if err := pub.PublishBatch(context.Background(), "topic", batch); err == nil {
		t.Fatalf("expected write error")
	}
}

func TestPublishBatchEnsureTopicError(t *testing.T) {
	p := &Publisher{config: &config.Config{Brokers: []string{}}, writer: &fakeWriter{}, ready: make(map[string]bool)}
	err := p.PublishBatch(context.Background(), "topic", []*types.Event{events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"})})
	if err == nil {
		t.Fatalf("expected ensureTopic error")
	}
}

func TestPublisherCloseWithNilWriterAndDebug(t *testing.T) {
	p := &Publisher{config: &config.Config{Debug: true}, ready: make(map[string]bool)}
	if err := p.Close(); err != nil {
		t.Fatalf("close should succeed even without writer: %v", err)
	}
}

func TestPublisherCloseWithWriterError(t *testing.T) {
	fw := &fakeWriter{closeErr: errors.New("close boom")}
	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: fw}
	if err := p.Close(); err == nil {
		t.Fatalf("expected close error")
	}
}

func TestEnsureTopicDialError(t *testing.T) {
	originalDialer := newDialer
	originalDialContext := dialContext
	defer func() {
		newDialer = originalDialer
		dialContext = originalDialContext
	}()

	newDialer = func(*config.Config) (*kafka.Dialer, error) { return &kafka.Dialer{}, nil }
	dialContext = func(*kafka.Dialer, context.Context, string, string) (topicConn, error) {
		return nil, errors.New("dial fail")
	}

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}
	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected dial error")
	}
}

func TestEnsureTopicDialerBuildError(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) { return nil, errors.New("bad dialer") }

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}
	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected dialer build error")
	}
}

func TestEnsureTopicReadPartitionsError(t *testing.T) {
	originalDialer := newDialer
	originalDialContext := dialContext
	defer func() {
		newDialer = originalDialer
		dialContext = originalDialContext
	}()

	fc := &fakeTopicConn{partitionsErr: errors.New("read fail")}
	newDialer = func(*config.Config) (*kafka.Dialer, error) { return &kafka.Dialer{}, nil }
	dialContext = func(*kafka.Dialer, context.Context, string, string) (topicConn, error) { return fc, nil }

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}
	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected read partitions error")
	}
}

func TestEnsureTopicDialLeaderError(t *testing.T) {
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
		return nil, errors.New("leader fail")
	}

	p := &Publisher{config: config.DefaultConfig(), ready: make(map[string]bool), writer: &fakeWriter{}}
	if err := p.ensureTopic(context.Background(), "t1"); err == nil {
		t.Fatalf("expected leader dial error")
	}
}

func TestEnsureTopicSetsDeadlineWhenContextHasTimeout(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := p.ensureTopic(ctx, "t1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fc.deadlineSet {
		t.Fatalf("expected SetDeadline to be invoked")
	}
}

func TestNewTransportWithTLSAndSASL(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) {
		return &kafka.Dialer{TLS: &tls.Config{}, SASLMechanism: plain.Mechanism{Username: "u", Password: "p"}}, nil
	}

	tr := newTransport(config.DefaultConfig())
	if tr.TLS == nil {
		t.Fatalf("expected TLS to be set")
	}
	if tr.SASL == nil {
		t.Fatalf("expected SASL to be set")
	}
}

func TestPublishWithDebug(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	cfg := config.DefaultConfig()
	cfg.Debug = true
	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pub.ready["topic"] = true

	if err := pub.Publish(context.Background(), "topic", events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"})); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
}

func TestPublishBatchWithDebug(t *testing.T) {
	originalWriter := newWriter
	defer func() { newWriter = originalWriter }()

	fw := &fakeWriter{}
	newWriter = func(*config.Config) (writerInterface, error) { return fw, nil }

	cfg := config.DefaultConfig()
	cfg.Debug = true
	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pub.ready["topic"] = true

	batch := []*types.Event{events.NewEvent(topics.UserCreated, "src", events.UserCreatedEvent{UserID: "u"})}
	if err := pub.PublishBatch(context.Background(), "topic", batch); err != nil {
		t.Fatalf("publish batch failed: %v", err)
	}
}

func TestPublisherCloseWithWriterDebug(t *testing.T) {
	fw := &fakeWriter{}
	p := &Publisher{config: &config.Config{Debug: true}, ready: make(map[string]bool), writer: fw}
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if fw.closed != 1 {
		t.Fatalf("writer should be closed")
	}
}

func TestNewTransportDialErrorPath(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) {
		return &kafka.Dialer{
			DialFunc: func(context.Context, string, string) (net.Conn, error) {
				return nil, errors.New("dial-fail")
			},
		}, nil
	}

	tr := newTransport(config.DefaultConfig())
	if _, err := tr.Dial(context.Background(), "tcp", "addr"); err == nil {
		t.Fatalf("expected dial error")
	}
}
