package middleware

import (
	"context"

	"github.com/serphona/backend/go/libs/platform-observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

// startAuthSpan starts a span for auth middleware/interceptor flows and tags the request ID.
func startAuthSpan(ctx context.Context, spanName, requestID string) (context.Context, trace.Span) {
	tracer := tracing.Tracer()
	ctx, span := tracer.Start(ctx, spanName)
	if requestID != "" {
		span.SetAttributes(attribute.String("auth.request_id", requestID))
	}
	return ctx, span
}

func annotateSpanWithClaims(span trace.Span, claims *types.Claims) {
	if span == nil || claims == nil {
		return
	}

	attrs := []attribute.KeyValue{}
	if claims.UserID != "" {
		attrs = append(attrs, attribute.String("auth.user_id", claims.UserID))
	}
	if claims.TenantID != "" {
		attrs = append(attrs, attribute.String("auth.tenant_id", claims.TenantID))
	}
	if claims.Role != "" {
		attrs = append(attrs, attribute.String("auth.role", claims.Role))
	}
	if len(claims.Scopes) > 0 {
		attrs = append(attrs, attribute.StringSlice("auth.scopes", claims.Scopes))
	}

	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

func recordSpanError(span trace.Span, mapped authMappedError, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
	}
	span.SetAttributes(attribute.String("auth.error_code", mapped.code))
	span.SetStatus(codes.Error, mapped.message)
}

func recordSpanSuccess(span trace.Span) {
	if span == nil {
		return
	}
	span.SetStatus(codes.Ok, "")
}
