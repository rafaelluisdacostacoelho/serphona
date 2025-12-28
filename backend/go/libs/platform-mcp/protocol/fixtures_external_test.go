package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
)

func TestExternalFixtures(t *testing.T) {
	base := "fixtures"

	// ok request
	okBytes := readFixture(t, base, "invocation_ok.json")
	var okReq InvocationRequest
	if err := json.Unmarshal(okBytes, &okReq); err != nil {
		t.Fatalf("unmarshal ok: %v", err)
	}
	if err := ValidateInvocation(okReq); err != nil {
		t.Fatalf("validate ok: %v", err)
	}

	// error event
	errBytes := readFixture(t, base, "invocation_error.json")
	var errEvt InvocationEvent
	if err := json.Unmarshal(errBytes, &errEvt); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errEvt.Type != EventError || errEvt.Error == nil || errEvt.Error.Code != mcperrors.ErrInvalidRequest {
		t.Fatalf("unexpected error fixture: %+v", errEvt)
	}

	// cancel event
	cancelBytes := readFixture(t, base, "invocation_cancel.json")
	var cancelEvt InvocationEvent
	if err := json.Unmarshal(cancelBytes, &cancelEvt); err != nil {
		t.Fatalf("unmarshal cancel: %v", err)
	}
	if cancelEvt.Error == nil || cancelEvt.Error.Code != mcperrors.ErrCancelled {
		t.Fatalf("unexpected cancel fixture: %+v", cancelEvt)
	}

	// unsupported version request
	unsupportedBytes := readFixture(t, base, "invocation_unsupported.json")
	var badReq InvocationRequest
	if err := json.Unmarshal(unsupportedBytes, &badReq); err != nil {
		t.Fatalf("unmarshal unsupported: %v", err)
	}
	if err := ValidateInvocation(badReq); err == nil {
		t.Fatalf("expected validation failure for unsupported version")
	}
}

func readFixture(t *testing.T, base, name string) []byte {
	t.Helper()
	path := filepath.Join(base, name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}
