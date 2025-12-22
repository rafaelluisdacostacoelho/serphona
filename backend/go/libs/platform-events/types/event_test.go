package types

import (
	"encoding/json"
	"errors"
	"testing"
)

type samplePayload struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestNewEventAndFluentSetters(t *testing.T) {
	e := NewEvent("evt.type", "source", map[string]string{"foo": "bar"})

	if e.ID == "" {
		t.Fatalf("expected generated ID")
	}
	if e.Type != "evt.type" || e.Source != "source" {
		t.Fatalf("unexpected type/source: %+v", e)
	}
	if e.Timestamp.IsZero() {
		t.Fatalf("timestamp should be set")
	}
	if e.Version != "1.0" {
		t.Fatalf("version mismatch: %s", e.Version)
	}
	if e.Metadata == nil {
		t.Fatalf("metadata should be initialized")
	}

	e.Metadata = nil // force nil branch in WithMetadata
	e.WithTenantID("tenant-1").WithUserID("user-1").WithMetadata("k", "v").WithTrace("trace", "span")

	if e.TenantID != "tenant-1" || e.UserID != "user-1" {
		t.Fatalf("tenant/user not set: %+v", e)
	}
	if got := e.Metadata["k"]; got != "v" {
		t.Fatalf("metadata not set: %+v", e.Metadata)
	}
	if e.TraceID != "trace" || e.SpanID != "span" {
		t.Fatalf("trace not set: %+v", e)
	}
}

func TestJSONRoundtrip(t *testing.T) {
	original := NewEvent("evt", "src", samplePayload{Name: "n", Value: 1})
	original.WithTenantID("t").WithUserID("u").WithTrace("tr", "sp").WithMetadata("key", "val")

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if restored.ID != original.ID || restored.Type != original.Type || restored.Source != original.Source {
		t.Fatalf("basic fields mismatch: %+v vs %+v", restored, original)
	}
	if restored.Metadata["key"] != "val" || restored.TenantID != "t" || restored.UserID != "u" {
		t.Fatalf("metadata/ids mismatch: %+v", restored)
	}
	if restored.TraceID != "tr" || restored.SpanID != "sp" {
		t.Fatalf("trace mismatch: %+v", restored)
	}
}

func TestFromJSONError(t *testing.T) {
	if _, err := FromJSON([]byte("{invalid")); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestBindErrorsOnNil(t *testing.T) {
	if _, err := Bind[samplePayload](nil); err == nil {
		t.Fatalf("expected error on nil event")
	}

	e := NewEvent("evt", "src", nil)
	if _, err := Bind[samplePayload](e); err == nil {
		t.Fatalf("expected error on nil data")
	}
}

func TestBindRawMessage(t *testing.T) {
	e := NewEvent("evt", "src", json.RawMessage(`{"name":"foo","value":3}`))
	out, err := Bind[samplePayload](e)
	if err != nil {
		t.Fatalf("bind raw message failed: %v", err)
	}
	if out.Name != "foo" || out.Value != 3 {
		t.Fatalf("unexpected payload: %+v", out)
	}

	e.Data = json.RawMessage(`{"name":}`)
	if _, err := Bind[samplePayload](e); err == nil {
		t.Fatalf("expected error for invalid raw message")
	}
}

func TestBindBytes(t *testing.T) {
	e := NewEvent("evt", "src", []byte(`{"name":"bar","value":4}`))
	out, err := Bind[samplePayload](e)
	if err != nil {
		t.Fatalf("bind bytes failed: %v", err)
	}
	if out.Name != "bar" || out.Value != 4 {
		t.Fatalf("unexpected payload: %+v", out)
	}

	e.Data = []byte(`{"value":}`)
	if _, err := Bind[samplePayload](e); err == nil {
		t.Fatalf("expected error for invalid bytes")
	}
}

func TestBindDefaultCases(t *testing.T) {
	// Success path with struct
	e := NewEvent("evt", "src", samplePayload{Name: "ok", Value: 9})
	out, err := Bind[samplePayload](e)
	if err != nil {
		t.Fatalf("bind default struct failed: %v", err)
	}
	if out.Name != "ok" || out.Value != 9 {
		t.Fatalf("unexpected payload: %+v", out)
	}

	// Marshal error path
	e.Data = make(chan int)
	if _, err := Bind[samplePayload](e); err == nil {
		t.Fatalf("expected marshal error")
	}

	// Unmarshal error path (marshal succeeds, unmarshal fails)
	e.Data = map[string]string{"value": "not-int"}
	_, err = Bind[samplePayload](e)
	if err == nil {
		t.Fatalf("expected unmarshal error")
	}

	var jsonTypeError *json.UnmarshalTypeError
	if !errors.As(err, &jsonTypeError) {
		t.Fatalf("expected UnmarshalTypeError, got %v", err)
	}
}
