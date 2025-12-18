package consumer

import (
	"context"
	"fmt"
	"testing"

	"github.com/segmentio/kafka-go"
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

func (r *stubReader) Close() error             { return nil }
func (r *stubReader) Stats() kafka.ReaderStats { return kafka.ReaderStats{} }

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
