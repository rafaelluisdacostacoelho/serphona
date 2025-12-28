package response

import (
	"context"
	"encoding/json"
	"testing"
)

func FuzzSuccessEnvelopeRoundtrip(f *testing.F) {
	f.Add("data", "req-1")
	f.Fuzz(func(t *testing.T, payload string, reqID string) {
		env := Success(context.Background(), payload, WithRequestID(reqID))
		b, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		var out SuccessEnvelope
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if out.Data != payload {
			t.Fatalf("data mismatch: %v vs %v", out.Data, payload)
		}
		if out.Meta != nil && out.Meta.RequestID != reqID {
			t.Fatalf("request_id mismatch: %s vs %s", out.Meta.RequestID, reqID)
		}
	})
}

func FuzzErrorEnvelopeRoundtrip(f *testing.F) {
	f.Add("CODE", "message", "req-2")
	f.Fuzz(func(t *testing.T, code string, msg string, reqID string) {
		env := Error(context.Background(), code, msg, map[string]string{"detail": "x"}, WithRequestID(reqID))
		b, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		var out ErrorEnvelope
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if out.Error.Code != code || out.Error.Message != msg {
			t.Fatalf("code/message mismatch: %#v", out.Error)
		}
		if out.Error.RequestID != reqID {
			t.Fatalf("request_id mismatch: %s vs %s", out.Error.RequestID, reqID)
		}
	})
}
