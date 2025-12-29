package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type recordingObserver struct {
	events []protocol.InvocationEvent
	errs   []error
	seen   int
}

func (r *recordingObserver) OnInvocationEvent(_ context.Context, _ protocol.InvocationRequest, evt protocol.InvocationEvent, err error, _ time.Duration) {
	r.seen++
	if err != nil {
		r.errs = append(r.errs, err)
	}
	if evt.Type != "" {
		r.events = append(r.events, evt)
	}
}

func TestObservedExecutorForwardsEvents(t *testing.T) {
	inner := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{Type: protocol.EventResult}, nil
		},
	})
	obs := &recordingObserver{}
	o := NewObservedExecutor(inner, obs)

	out, err := o.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-out; evt.Type != protocol.EventResult {
		t.Fatalf("unexpected event: %+v", evt)
	}
	if obs.seen == 0 || len(obs.events) != 1 {
		t.Fatalf("observer not called: %+v", obs)
	}
}

func TestObservedExecutorReportsImmediateError(t *testing.T) {
	inner := NoopExecutor{}
	obs := &recordingObserver{}
	o := NewObservedExecutor(inner, obs)

	_, err := o.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: ""}, Input: []byte(`{}`)})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if obs.seen == 0 || len(obs.errs) == 0 {
		t.Fatalf("expected observer to record error: %+v", obs)
	}
}

func TestObservedExecutorWithNilObserverBypassesWrapping(t *testing.T) {
	innerCh := make(chan protocol.InvocationEvent, 1)
	innerCh <- protocol.InvocationEvent{Type: protocol.EventResult}
	close(innerCh)
	inner := passThroughExecutor{ch: innerCh}

	o := NewObservedExecutor(inner, nil)
	out, err := o.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out != innerCh {
		t.Fatalf("expected direct channel passthrough")
	}
	if evt := <-out; evt.Type != protocol.EventResult {
		t.Fatalf("unexpected event: %+v", evt)
	}
}
