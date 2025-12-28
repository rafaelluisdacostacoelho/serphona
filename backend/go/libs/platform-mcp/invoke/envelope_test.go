package invoke

import (
	"encoding/json"
	"testing"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
)

func TestOkEnvelope(t *testing.T) {
	e := OkEnvelope(map[string]string{"msg": "hi"})
	if e.Status != "ok" || e.Error != nil {
		t.Fatalf("unexpected ok envelope: %+v", e)
	}
	if _, err := json.Marshal(e); err != nil {
		t.Fatalf("marshal: %v", err)
	}
}

func TestErrorEnvelope(t *testing.T) {
	e := ErrorEnvelope(mcperrors.ErrInvalidRequest, "bad")
	if e.Status != "error" || e.Error == nil || e.Error.Code != mcperrors.ErrInvalidRequest {
		t.Fatalf("unexpected error envelope: %+v", e)
	}
	if _, err := json.Marshal(e); err != nil {
		t.Fatalf("marshal: %v", err)
	}
}
