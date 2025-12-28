package middleware

import (
	"context"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestAnnotateSpanWithClaimsAvoidsSensitiveFields(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(recorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "auth")
	claims := &types.Claims{
		UserID:    "user-1",
		TenantID:  "tenant-1",
		Role:      "admin",
		SessionID: "session-secret",
		Scopes:    []string{"read", "write"},
	}

	annotateSpanWithClaims(span, claims)
	span.End()

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrMap := map[string]attribute.Value{}
	for _, kv := range spans[0].Attributes() {
		attrMap[string(kv.Key)] = kv.Value
	}

	if v, ok := attrMap["auth.user_id"]; !ok || v.AsString() != "user-1" {
		t.Fatalf("user id attribute missing or unexpected: %v", attrMap)
	}
	if v, ok := attrMap["auth.tenant_id"]; !ok || v.AsString() != "tenant-1" {
		t.Fatalf("tenant id attribute missing or unexpected: %v", attrMap)
	}
	if v, ok := attrMap["auth.role"]; !ok || v.AsString() != "admin" {
		t.Fatalf("role attribute missing or unexpected: %v", attrMap)
	}
	if v, ok := attrMap["auth.scopes"]; !ok || len(v.AsStringSlice()) != 2 {
		t.Fatalf("scopes attribute missing or unexpected: %v", attrMap)
	}
	if _, ok := attrMap["auth.session_id"]; ok {
		t.Fatalf("session id should not be recorded in span attributes: %v", attrMap)
	}
}
