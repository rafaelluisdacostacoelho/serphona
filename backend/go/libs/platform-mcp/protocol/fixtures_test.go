package protocol

import (
	"encoding/json"
	"testing"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
)

func TestInvocationFixtureRoundTrip(t *testing.T) {
	// ok fixture
	raw := []byte(`{"version":"v1","tenant_id":"t1","tool":{"name":"echo"},"input":{"msg":"hi"}}`)
	var req InvocationRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := ValidateInvocation(req); err != nil {
		t.Fatalf("validate: %v", err)
	}
	out, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected output bytes")
	}

	// error fixture
	errRaw := []byte(`{"type":"error","error":{"code":"invalid_request","message":"bad"}}`)
	var evt InvocationEvent
	if err := json.Unmarshal(errRaw, &evt); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if evt.Type != EventError || evt.Error == nil || evt.Error.Code != mcperrors.ErrInvalidRequest {
		t.Fatalf("unexpected error event: %+v", evt)
	}

	// cancel fixture (treated as error with cancelled code)
	cancelRaw := []byte(`{"type":"error","error":{"code":"cancelled","message":"client cancelled"}}`)
	var cancelEvt InvocationEvent
	if err := json.Unmarshal(cancelRaw, &cancelEvt); err != nil {
		t.Fatalf("unmarshal cancel: %v", err)
	}
	if cancelEvt.Error == nil || cancelEvt.Error.Code != mcperrors.ErrCancelled {
		t.Fatalf("unexpected cancel event: %+v", cancelEvt)
	}

	// unsupported version fixture
	if err := ValidateVersion("v2"); err == nil {
		t.Fatalf("expected version validation failure")
	}
}
