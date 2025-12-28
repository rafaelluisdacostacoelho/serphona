package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

var sensitiveHeaderKeys = map[string]struct{}{
	"authorization":             {},
	"proxy-authorization":       {},
	"cookie":                    {},
	"set-cookie":                {},
	"x-api-key":                 {},
	"x-auth-token":              {},
	"x-id-token":                {},
	"x-refresh-token":           {},
	"x-access-token":            {},
	"x-forwarded-authorization": {},
}

// RedactHeaders returns a copy of the headers with sensitive keys redacted to avoid leaking secrets in logs.
func RedactHeaders(src http.Header) http.Header {
	if src == nil {
		return http.Header{}
	}

	out := http.Header{}
	for k, vals := range src {
		lower := strings.ToLower(k)
		if _, ok := sensitiveHeaderKeys[lower]; ok {
			out[k] = []string{"[REDACTED]"}
			continue
		}
		copyVals := make([]string, len(vals))
		copy(copyVals, vals)
		out[k] = copyVals
	}

	return out
}

// RedactMetadata returns a copy of the metadata with sensitive keys redacted to avoid leaking secrets in logs.
func RedactMetadata(src metadata.MD) metadata.MD {
	if src == nil {
		return metadata.MD{}
	}

	out := metadata.MD{}
	for k, vals := range src {
		lower := strings.ToLower(k)
		if _, ok := sensitiveHeaderKeys[lower]; ok {
			out[k] = []string{"[REDACTED]"}
			continue
		}
		copyVals := make([]string, len(vals))
		copy(copyVals, vals)
		out[k] = copyVals
	}

	return out
}

// SafeRequestFields builds a minimal, sanitized map for logging HTTP requests without secrets.
func SafeRequestFields(r *http.Request) map[string]any {
	if r == nil {
		return map[string]any{}
	}

	fields := map[string]any{
		"method": r.Method,
		"path":   r.URL.Path,
	}

	if reqID, err := RequestIDFromContext(r.Context()); err == nil && reqID != "" {
		fields["request_id"] = reqID
	}

	if r.RemoteAddr != "" {
		fields["remote_addr"] = r.RemoteAddr
	}

	fields["headers"] = RedactHeaders(r.Header)

	return fields
}

// SafeRequestFieldsFromGin builds sanitized fields from a Gin context.
func SafeRequestFieldsFromGin(c *gin.Context) map[string]any {
	if c == nil {
		return map[string]any{}
	}
	return SafeRequestFields(c.Request)
}

// SafeGRPCRequestFields builds sanitized fields from an incoming gRPC context and method.
func SafeGRPCRequestFields(ctx context.Context, fullMethod string) map[string]any {
	if ctx == nil {
		return map[string]any{}
	}

	fields := map[string]any{
		"method": fullMethod,
	}

	if reqID, err := RequestIDFromContext(ctx); err == nil && reqID != "" {
		fields["request_id"] = reqID
	}

	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		fields["remote_addr"] = p.Addr.String()
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		fields["metadata"] = RedactMetadata(md)
	} else {
		fields["metadata"] = metadata.MD{}
	}

	return fields
}

const safeRequestFieldsContextKey contextKey = "platform-auth-safe-request-fields"

// WithSafeRequestFields attaches sanitized request fields to context for logging.
func WithSafeRequestFields(ctx context.Context, fields map[string]any) context.Context {
	return context.WithValue(ctx, safeRequestFieldsContextKey, fields)
}

// SafeRequestFieldsFromContext retrieves sanitized request fields from context.
func SafeRequestFieldsFromContext(ctx context.Context) (map[string]any, bool) {
	if ctx == nil {
		return nil, false
	}
	if fields, ok := ctx.Value(safeRequestFieldsContextKey).(map[string]any); ok {
		return fields, true
	}
	return nil, false
}

// SafeRequestKeyValues flattens safe request fields into key/value pairs for structured logging.
// Returns nil when no safe fields are present.
func SafeRequestKeyValues(ctx context.Context) []any {
	fields, ok := SafeRequestFieldsFromContext(ctx)
	if !ok {
		return nil
	}

	kvs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		kvs = append(kvs, k, v)
	}
	return kvs
}
