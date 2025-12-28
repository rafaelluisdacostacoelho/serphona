package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
)

func TestWriteSuccessEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"status": "ok"}

	response.WriteSuccess(nil, rec, http.StatusOK, payload)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if body.Data["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body.Data["status"])
	}
}

func TestWriteErrorEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()

	response.WriteError(nil, rec, http.StatusBadRequest, "INVALID", "bad input", nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if body.Error.Code != "INVALID" {
		t.Fatalf("expected error INVALID, got %s", body.Error.Code)
	}
	if body.Error.Message != "bad input" {
		t.Fatalf("expected message bad input, got %s", body.Error.Message)
	}
}
