package consumer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func TestHydrateHeadersSetsCoreFieldsAndMetadata(t *testing.T) {
	evt := &types.Event{Metadata: map[string]string{"existing": "keep"}}
	msg := kafka.Message{
		Headers: []kafka.Header{
			{Key: "tenant_id", Value: []byte("tenant-1")},
			{Key: "user_id", Value: []byte("user-2")},
			{Key: "trace_id", Value: []byte("trace-3")},
			{Key: "span_id", Value: []byte("span-4")},
			{Key: "version", Value: []byte("1.1")},
			{Key: "source", Value: []byte("svc-a")},
			{Key: "custom", Value: []byte("meta")},
		},
	}

	hydrateHeaders(evt, msg)

	if evt.TenantID != "tenant-1" {
		t.Fatalf("TenantID mismatch: %s", evt.TenantID)
	}
	if evt.UserID != "user-2" {
		t.Fatalf("UserID mismatch: %s", evt.UserID)
	}
	if evt.TraceID != "trace-3" {
		t.Fatalf("TraceID mismatch: %s", evt.TraceID)
	}
	if evt.SpanID != "span-4" {
		t.Fatalf("SpanID mismatch: %s", evt.SpanID)
	}
	if evt.Version != "1.1" {
		t.Fatalf("Version mismatch: %s", evt.Version)
	}
	if evt.Source != "svc-a" {
		t.Fatalf("Source mismatch: %s", evt.Source)
	}
	if evt.Metadata["custom"] != "meta" {
		t.Fatalf("Metadata custom mismatch: %s", evt.Metadata["custom"])
	}
	if evt.Metadata["existing"] != "keep" {
		t.Fatalf("Existing metadata lost")
	}
}

func TestHydrateHeadersDoesNotOverrideExistingFields(t *testing.T) {
	evt := &types.Event{
		TenantID: "tenant-keep",
		UserID:   "user-keep",
		TraceID:  "trace-keep",
		SpanID:   "span-keep",
		Version:  "2.0",
		Source:   "svc-keep",
	}
	msg := kafka.Message{
		Headers: []kafka.Header{
			{Key: "tenant_id", Value: []byte("tenant-new")},
			{Key: "user_id", Value: []byte("user-new")},
			{Key: "trace_id", Value: []byte("trace-new")},
			{Key: "span_id", Value: []byte("span-new")},
			{Key: "version", Value: []byte("9.9")},
			{Key: "source", Value: []byte("svc-new")},
		},
	}

	hydrateHeaders(evt, msg)

	if evt.TenantID != "tenant-keep" || evt.UserID != "user-keep" || evt.TraceID != "trace-keep" || evt.SpanID != "span-keep" || evt.Version != "2.0" || evt.Source != "svc-keep" {
		t.Fatalf("Existing fields were overridden: %+v", evt)
	}
}

func TestProcessMessageFiltersAndRetries(t *testing.T) {
	event := types.NewEvent("event.test", "svc-a", map[string]string{"foo": "bar"})
	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	filterCalled := 0
	handler1Called := 0
	handler2Calls := 0

	filter := func(evt *types.Event) bool {
		filterCalled++
		return false // should skip handler1
	}

	handler1 := func(evt *types.Event) error {
		handler1Called++
		return nil
	}

	handler2 := func(evt *types.Event) error {
		handler2Calls++
		if handler2Calls == 1 {
			return fmt.Errorf("transient")
		}
		return nil
	}

	c := &Consumer{
		config: &config.Config{ConsumerMaxRetries: 2, ConsumerRetryInterval: 1},
		handlers: map[string][]types.EventHandler{
			event.Type: {handler1, handler2},
		},
		filters: map[string][]types.EventFilter{
			event.Type: {filter},
		},
	}

	msg := kafka.Message{Value: data, Headers: []kafka.Header{{Key: "tenant_id", Value: []byte("tenant-123")}}}

	if err := c.processMessage(msg); err != nil {
		t.Fatalf("processMessage returned error: %v", err)
	}

	if filterCalled != 1 {
		t.Fatalf("expected filter to be called once, got %d", filterCalled)
	}
	if handler1Called != 0 {
		t.Fatalf("handler1 should have been skipped by filter, got %d", handler1Called)
	}
	if handler2Calls != 2 {
		t.Fatalf("handler2 should be retried once, got %d calls", handler2Calls)
	}
}

type stubReader struct {
	msgs        []kafka.Message
	idx         int
	commitCount int
	closed      bool
	stats       kafka.ReaderStats
}

func (r *stubReader) FetchMessage(context.Context) (kafka.Message, error) {
	if r.idx >= len(r.msgs) {
		return kafka.Message{}, context.Canceled
	}
	m := r.msgs[r.idx]
	r.idx++
	return m, nil
}

func (r *stubReader) CommitMessages(context.Context, ...kafka.Message) error {
	r.commitCount++
	return nil
}

func (r *stubReader) Close() error {
	r.closed = true
	return nil
}
func (r *stubReader) Stats() kafka.ReaderStats { return r.stats }

func TestWorkerCommitsWhenAutoCommitDisabled(t *testing.T) {
	event := types.NewEvent("event.test", "svc-a", map[string]string{"foo": "bar"})
	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	r := &stubReader{msgs: []kafka.Message{{Value: data}}}
	c := &Consumer{
		reader: r,
		config: &config.Config{EnableAutoCommit: false, ConsumerMaxRetries: 1, ConsumerRetryInterval: 1},
		handlers: map[string][]types.EventHandler{
			event.Type: {func(*types.Event) error { return nil }},
		},
		filters: make(map[string][]types.EventFilter),
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go c.worker(0)
	c.wg.Wait()

	if r.commitCount != 1 {
		t.Fatalf("expected manual commit to be called once, got %d", r.commitCount)
	}
}

func TestWorkerSkipsCommitWhenAutoCommitEnabled(t *testing.T) {
	event := types.NewEvent("event.test", "svc-a", map[string]string{"foo": "bar"})
	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	r := &stubReader{msgs: []kafka.Message{{Value: data}}}
	c := &Consumer{
		reader: r,
		config: &config.Config{EnableAutoCommit: true, ConsumerMaxRetries: 1, ConsumerRetryInterval: 1},
		handlers: map[string][]types.EventHandler{
			event.Type: {func(*types.Event) error { return nil }},
		},
		filters: make(map[string][]types.EventFilter),
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go c.worker(0)
	c.wg.Wait()

	if r.commitCount != 0 {
		t.Fatalf("expected no manual commits when auto-commit is enabled, got %d", r.commitCount)
	}
}

type errorReader struct{}

func (e *errorReader) FetchMessage(context.Context) (kafka.Message, error) {
	return kafka.Message{}, errors.New("fetch")
}
func (e *errorReader) CommitMessages(context.Context, ...kafka.Message) error {
	return errors.New("commit")
}
func (e *errorReader) Close() error             { return errors.New("close") }
func (e *errorReader) Stats() kafka.ReaderStats { return kafka.ReaderStats{} }

func TestStartFailsWhenClosed(t *testing.T) {
	c := &Consumer{closed: true, config: config.DefaultConfig(), reader: &stubReader{}}
	if err := c.Start(); err != ErrConsumerClosed {
		t.Fatalf("expected ErrConsumerClosed, got %v", err)
	}
}

func TestClosePropagatesReaderError(t *testing.T) {
	c := &Consumer{config: config.DefaultConfig(), reader: &errorReader{}}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if err := c.Close(); err == nil {
		t.Fatalf("expected close error")
	}
}

func TestNewWithDebugUsesInjectedReader(t *testing.T) {
	originalReader := newReader
	defer func() { newReader = originalReader }()

	fake := &stubReader{}
	newReader = func(kafka.ReaderConfig) reader { return fake }

	cfg := config.DefaultConfig()
	cfg.Debug = true
	cfg.Brokers = []string{"localhost:9092"}

	cfg.GroupID = "g"
	cfg.ClientID = "c"

	c, err := New(cfg, []string{"t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ensure cleanup works with injected reader
	_ = c.Close()
}

func TestNewDialerErrorPath(t *testing.T) {
	originalDialer := newDialer
	defer func() { newDialer = originalDialer }()

	newDialer = func(*config.Config) (*kafka.Dialer, error) { return nil, errors.New("dialer fail") }
	cfg := config.DefaultConfig()

	if _, err := New(cfg, []string{"t"}); err == nil {
		t.Fatalf("expected dialer error")
	}
}

func TestNewValidationError(t *testing.T) {
	cfg := &config.Config{}
	if _, err := New(cfg, []string{"t"}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestStartDebugBranch(t *testing.T) {
	originalReader := newReader
	defer func() { newReader = originalReader }()

	newReader = func(cfg kafka.ReaderConfig) reader {
		return &seqReader{responses: []struct {
			msg kafka.Message
			err error
		}{
			{err: context.Canceled},
		}}
	}

	cfg := config.DefaultConfig()
	cfg.Debug = true
	cfg.ConsumerConcurrency = 1

	c, err := New(cfg, []string{"t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	_ = c.Close()
}

func TestWorkerContextCanceledExit(t *testing.T) {
	r := &seqReader{responses: []struct {
		msg kafka.Message
		err error
	}{
		{err: context.Canceled},
	}}

	c := &Consumer{reader: r, config: config.DefaultConfig()}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go c.worker(0)
	c.wg.Wait()
}

func TestNewNoTopicsError(t *testing.T) {
	cfg := config.DefaultConfig()
	if _, err := New(cfg, nil); !errors.Is(err, ErrNoTopics) {
		t.Fatalf("expected ErrNoTopics, got %v", err)
	}
}

func TestExecuteWithRetryDebugLogs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Debug = true
	cfg.ConsumerMaxRetries = 2
	cfg.ConsumerRetryInterval = time.Millisecond

	c := &Consumer{config: cfg}
	handlerCalls := 0
	err := c.executeWithRetry(func(*types.Event) error {
		handlerCalls++
		return errors.New("fail")
	}, &types.Event{ID: "1"})

	if err == nil {
		t.Fatalf("expected error after retries")
	}
	if handlerCalls != cfg.ConsumerMaxRetries {
		t.Fatalf("expected %d calls, got %d", cfg.ConsumerMaxRetries, handlerCalls)
	}
}

func TestProcessMessageDebugNoHandlers(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Debug = true

	c := &Consumer{config: cfg, handlers: make(map[string][]types.EventHandler), filters: make(map[string][]types.EventFilter)}
	evt := &types.Event{ID: "1", Type: "t1", Source: "src", Version: "v1", Timestamp: time.Now()}
	data, _ := evt.ToJSON()

	msg := kafka.Message{Value: data, Topic: "t1"}

	if err := c.processMessage(msg); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestNewSetsManualCommitInterval(t *testing.T) {
	originalReader := newReader
	defer func() { newReader = originalReader }()

	var commitInterval time.Duration
	newReader = func(cfg kafka.ReaderConfig) reader {
		commitInterval = cfg.CommitInterval
		return &seqReader{}
	}

	cfg := config.DefaultConfig()
	cfg.EnableAutoCommit = false

	if _, err := New(cfg, []string{"t"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if commitInterval != 0 {
		t.Fatalf("expected manual commit interval 0, got %v", commitInterval)
	}
}

func TestNewSetsAutoCommitInterval(t *testing.T) {
	originalReader := newReader
	defer func() { newReader = originalReader }()

	var commitInterval time.Duration
	newReader = func(cfg kafka.ReaderConfig) reader {
		commitInterval = cfg.CommitInterval
		return &seqReader{}
	}

	cfg := config.DefaultConfig()
	cfg.EnableAutoCommit = true
	cfg.CommitInterval = time.Second

	if _, err := New(cfg, []string{"t"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if commitInterval != time.Second {
		t.Fatalf("expected commit interval to match config, got %v", commitInterval)
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
		t.Fatalf("scram mechanism was not invoked")
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

func TestProcessMessageFilterDebugSkip(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Debug = true

	handlerCalls := 0
	c := &Consumer{config: cfg, handlers: make(map[string][]types.EventHandler), filters: make(map[string][]types.EventFilter)}
	c.handlers["t1"] = []types.EventHandler{func(*types.Event) error { handlerCalls++; return nil }}
	c.filters["t1"] = []types.EventFilter{func(*types.Event) bool { return false }}

	evt := &types.Event{ID: "1", Type: "t1", Source: "src", Version: "v1", Timestamp: time.Now()}
	data, _ := evt.ToJSON()
	msg := kafka.Message{Value: data, Topic: "t1"}

	if err := c.processMessage(msg); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if handlerCalls != 0 {
		t.Fatalf("handler should not run when filter blocks")
	}
}

func TestProcessMessageNoHandlersDebug(t *testing.T) {
	evt := types.NewEvent("evt.none", "src", map[string]string{"k": "v"})
	msg := mustMessage(evt)

	c := &Consumer{config: &config.Config{Debug: true}, handlers: make(map[string][]types.EventHandler), filters: make(map[string][]types.EventFilter)}

	if err := c.processMessage(msg); err != nil {
		t.Fatalf("expected nil error: %v", err)
	}
}
func TestProcessMessageContinuesOnHandlerError(t *testing.T) {
	event := types.NewEvent("event.test", "svc-a", map[string]string{"foo": "bar"})
	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	errCount := 0
	successCount := 0

	c := &Consumer{
		config: &config.Config{ConsumerMaxRetries: 1, ConsumerRetryInterval: 1},
		handlers: map[string][]types.EventHandler{
			event.Type: {
				func(*types.Event) error {
					errCount++
					return fmt.Errorf("boom")
				},
				func(*types.Event) error {
					successCount++
					return nil
				},
			},
		},
		filters: make(map[string][]types.EventFilter),
	}

	msg := kafka.Message{Value: data}

	if err := c.processMessage(msg); err != nil {
		t.Fatalf("processMessage returned error: %v", err)
	}

	if errCount != 1 {
		t.Fatalf("expected first handler to fail once, got %d", errCount)
	}
	if successCount != 1 {
		t.Fatalf("expected second handler to run despite first failing, got %d", successCount)
	}
}

func TestNewWithoutTopics(t *testing.T) {
	cfg := config.DefaultConfig()
	if _, err := New(cfg, nil); err != ErrNoTopics {
		t.Fatalf("expected ErrNoTopics, got %v", err)
	}
}

func TestExecuteWithRetryFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ConsumerMaxRetries = 2
	cfg.ConsumerRetryInterval = time.Millisecond

	c := &Consumer{config: cfg}
	attempts := 0
	err := c.executeWithRetry(func(*types.Event) error {
		attempts++
		return errors.New("boom")
	}, &types.Event{ID: "id"})

	if err == nil {
		t.Fatalf("expected error")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestProcessMessageWithNoHandlers(t *testing.T) {
	event := types.NewEvent("evt.none", "src", map[string]string{"k": "v"})
	msg := mustMessage(event)

	c := &Consumer{
		config:   config.DefaultConfig(),
		handlers: make(map[string][]types.EventHandler),
		filters:  make(map[string][]types.EventFilter),
	}

	if err := c.processMessage(msg); err != nil {
		t.Fatalf("expected nil when no handlers, got %v", err)
	}
}

func TestStartCloseAndStats(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ConsumerConcurrency = 1
	cfg.EnableAutoCommit = false

	r := &stubReader{msgs: []kafka.Message{mustMessage(types.NewEvent("evt", "src", map[string]string{"k": "v"}))}, stats: kafka.ReaderStats{Fetches: 42}}
	c := &Consumer{
		reader:   r,
		config:   cfg,
		handlers: map[string][]types.EventHandler{"evt": {func(*types.Event) error { return nil }}},
		filters:  make(map[string][]types.EventFilter),
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	if err := c.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := c.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if !r.closed {
		t.Fatalf("reader should be closed")
	}
	if r.commitCount == 0 {
		t.Fatalf("expected at least one commit")
	}

	stats := c.Stats()
	if stats.Fetches != 42 {
		t.Fatalf("stats not forwarded")
	}

	if err := c.Start(); err != ErrConsumerClosed {
		t.Fatalf("expected ErrConsumerClosed after close, got %v", err)
	}
}

func TestNewDialerBuildsTLSAndSASL(t *testing.T) {
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

func TestCloseIsIdempotent(t *testing.T) {
	r := &stubReader{msgs: []kafka.Message{mustMessage(types.NewEvent("evt", "src", map[string]string{"k": "v"}))}}
	c := &Consumer{reader: r, config: config.DefaultConfig()}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	if err := c.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second close should be nil, got %v", err)
	}
}

func TestSubscribeHelpers(t *testing.T) {
	c := &Consumer{
		config:   config.DefaultConfig(),
		reader:   &stubReader{},
		handlers: make(map[string][]types.EventHandler),
		filters:  make(map[string][]types.EventFilter),
	}
	c.Subscribe("evt", func(*types.Event) error { return nil })
	c.SubscribeWithFilter("evt", func(*types.Event) bool { return true }, func(*types.Event) error { return nil })

	if len(c.handlers["evt"]) != 2 {
		t.Fatalf("handlers not registered")
	}
	if len(c.filters["evt"]) != 1 {
		t.Fatalf("filters not registered")
	}
}

func TestNewValidationPaths(t *testing.T) {
	bad := config.DefaultConfig()
	bad.Brokers = nil
	if _, err := New(bad, []string{"t"}); err == nil {
		t.Fatalf("expected validation error")
	}

	good := config.DefaultConfig()
	c, err := New(good, []string{"t"})
	if err != nil {
		t.Fatalf("expected consumer, got error: %v", err)
	}
	// Close immediately to avoid goroutine leaks
	_ = c.Close()
}

func TestNewDialerScramVariants(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SASLUsername = "u"
	cfg.SASLPassword = "p"

	cfg.SASLMechanism = "scram-sha256"
	if _, err := newDialer(cfg); err != nil {
		t.Fatalf("scram-sha256 should succeed: %v", err)
	}

	cfg.SASLMechanism = "scram-sha512"
	if _, err := newDialer(cfg); err != nil {
		t.Fatalf("scram-sha512 should succeed: %v", err)
	}
}

func TestNewDialerDefaultsNoTLSNoSASL(t *testing.T) {
	cfg := config.DefaultConfig()
	d, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.TLS != nil {
		t.Fatalf("expected nil TLS")
	}
	if d.SASLMechanism != nil {
		t.Fatalf("expected nil SASL mechanism")
	}
}

func TestSubscribeDebugLoggingBranches(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Debug = true
	c := &Consumer{
		config:   cfg,
		reader:   &stubReader{},
		handlers: make(map[string][]types.EventHandler),
		filters:  make(map[string][]types.EventFilter),
	}

	c.Subscribe("evt", func(*types.Event) error { return nil })
	c.SubscribeWithFilter("evt", func(*types.Event) bool { return true }, func(*types.Event) error { return nil })

	if len(c.handlers["evt"]) != 2 {
		t.Fatalf("handlers not registered")
	}
}

type seqReader struct {
	responses []struct {
		msg kafka.Message
		err error
	}
	idx         int
	commitCount int
	closed      bool
}

func (s *seqReader) FetchMessage(context.Context) (kafka.Message, error) {
	if s.idx >= len(s.responses) {
		return kafka.Message{}, context.Canceled
	}
	r := s.responses[s.idx]
	s.idx++
	return r.msg, r.err
}

func (s *seqReader) CommitMessages(context.Context, ...kafka.Message) error {
	s.commitCount++
	return nil
}

func (s *seqReader) Close() error {
	s.closed = true
	return nil
}

func (s *seqReader) Stats() kafka.ReaderStats { return kafka.ReaderStats{} }

func TestWorkerCoversErrorPaths(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.EnableAutoCommit = false
	cfg.ConsumerMaxRetries = 1

	event := types.NewEvent("evt", "src", map[string]string{"ok": "1"})
	good := mustMessage(event)
	bad := kafka.Message{Value: []byte("{oops")}

	r := &seqReader{responses: []struct {
		msg kafka.Message
		err error
	}{
		{msg: good},                     // success -> commit
		{msg: bad},                      // deserialization error
		{err: errors.New("fetch-fail")}, // fetch error (non-cancel)
		{err: context.Canceled},         // stop
	}}

	c := &Consumer{
		reader:   r,
		config:   cfg,
		handlers: map[string][]types.EventHandler{"evt": {func(*types.Event) error { return nil }}},
		filters:  make(map[string][]types.EventFilter),
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.wg.Add(1)
	go c.worker(0)
	c.wg.Wait()

	if r.commitCount != 1 {
		t.Fatalf("expected one commit, got %d", r.commitCount)
	}
}

type commitErrReader struct {
	msg         kafka.Message
	fetched     bool
	commitCalls int
}

func (r *commitErrReader) FetchMessage(context.Context) (kafka.Message, error) {
	if r.fetched {
		return kafka.Message{}, context.Canceled
	}
	r.fetched = true
	return r.msg, nil
}
func (r *commitErrReader) CommitMessages(context.Context, ...kafka.Message) error {
	r.commitCalls++
	return errors.New("commit fail")
}
func (r *commitErrReader) Close() error             { return nil }
func (r *commitErrReader) Stats() kafka.ReaderStats { return kafka.ReaderStats{} }

func TestWorkerCommitErrorBranch(t *testing.T) {
	evt := types.NewEvent("evt", "src", map[string]string{})
	msg := mustMessage(evt)
	r := &commitErrReader{msg: msg}

	c := &Consumer{
		reader:   r,
		config:   &config.Config{EnableAutoCommit: false, ConsumerMaxRetries: 1, ConsumerRetryInterval: 1},
		handlers: map[string][]types.EventHandler{"evt": {func(*types.Event) error { return nil }}},
		filters:  make(map[string][]types.EventFilter),
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go c.worker(0)
	c.wg.Wait()

	if r.commitCalls != 1 {
		t.Fatalf("expected commit to be attempted once, got %d", r.commitCalls)
	}
}

// mustMessage helper builds a kafka.Message from Event or fails the test
func mustMessage(evt *types.Event) kafka.Message {
	data, err := evt.ToJSON()
	if err != nil {
		panic(err)
	}
	return kafka.Message{Value: data}
}
