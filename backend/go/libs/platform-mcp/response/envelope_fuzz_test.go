package response

import (
	"context"
	"encoding/json"
	"testing"
)

func FuzzSuccessEnvelopeRoundTrip(f *testing.F) {
	f.Add("trace-123", "req-123")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, traceID, requestID string) {
		env := Success(context.Background(), map[string]string{"foo": "bar"}, WithTraceID(traceID), WithRequestID(requestID))
		b, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var round SuccessEnvelope
		if err := json.Unmarshal(b, &round); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if round.Data == nil {
			t.Fatalf("data missing after round trip")
		}
		if traceID != "" && (round.Meta == nil || round.Meta.TraceID != traceID) {
			t.Fatalf("trace_id lost: %#v", round.Meta)
		}
		if requestID != "" && (round.Meta == nil || round.Meta.RequestID != requestID) {
			t.Fatalf("request_id lost: %#v", round.Meta)
		}
	})
}

func FuzzErrorEnvelopeRoundTrip(f *testing.F) {
	f.Add("CODE", "message", "req-123")

	f.Fuzz(func(t *testing.T, code, msg, requestID string) {
		env := Error(context.Background(), code, msg, map[string]string{"detail": "x"}, WithRequestID(requestID))
		b, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var round ErrorEnvelope
		if err := json.Unmarshal(b, &round); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if round.Error.Code == "" || round.Error.Message == "" {
			t.Fatalf("missing code/message after round trip: %#v", round.Error)
		}
		if requestID != "" && round.Error.RequestID != requestID {
			t.Fatalf("request_id lost: %#v", round.Error)
		}
	})
}
