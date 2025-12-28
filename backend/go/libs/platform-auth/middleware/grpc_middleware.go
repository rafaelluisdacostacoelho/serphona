package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryAuthInterceptor enforces authentication (and optionally all required scopes) on unary RPCs.
func UnaryAuthInterceptor(requiredScopes ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		reqID := extractOrGenerateRequestID(extractRequestID(ctx))
		ctx = WithRequestID(ctx, reqID)
		ctx = WithSafeRequestFields(ctx, SafeGRPCRequestFields(ctx, info.FullMethod))
		ctx, span := startAuthSpan(ctx, "platform-auth.grpc.unary-auth", reqID)
		defer span.End()
		_ = grpc.SetHeader(ctx, grpcResponseMetadata(ctx, reqID))
		cfg := authjwt.GetValidationConfig()
		if needsSecret(cfg) {
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				mapped := mapAuthError(err)
				recordAuthError("grpc", mapped, start)
				recordSpanError(span, mapped, err)
				return nil, grpcError(mapped)
			}
		} else {
			_ = authjwt.EnsureSecretLoaded()
		}

		authHeader := extractAuthorization(ctx)
		claims, err := authjwt.ValidateTokenFromHeader(authHeader)
		if err != nil {
			mapped := mapAuthError(err)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, err)
			return nil, grpcError(mapped)
		}

		if len(requiredScopes) > 0 && !claims.HasAllScopes(requiredScopes...) {
			mapped := mapAuthError(autherrors.ErrInsufficientPermissions)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, nil)
			return nil, grpcError(mapped)
		}

		annotateSpanWithClaims(span, claims)
		ctxWithClaims := WithClaims(ctx, claims)
		recordAuthSuccess("grpc", start)
		recordSpanSuccess(span)
		return handler(ctxWithClaims, req)
	}
}

// UnaryAnyScopeInterceptor enforces that the caller has at least one of the specified scopes.
func UnaryAnyScopeInterceptor(scopes ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		reqID := extractOrGenerateRequestID(extractRequestID(ctx))
		ctx = WithRequestID(ctx, reqID)
		ctx = WithSafeRequestFields(ctx, SafeGRPCRequestFields(ctx, info.FullMethod))
		ctx, span := startAuthSpan(ctx, "platform-auth.grpc.unary-any-scope", reqID)
		defer span.End()
		_ = grpc.SetHeader(ctx, grpcResponseMetadata(ctx, reqID))
		cfg := authjwt.GetValidationConfig()
		if needsSecret(cfg) {
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				mapped := mapAuthError(err)
				recordAuthError("grpc", mapped, start)
				recordSpanError(span, mapped, err)
				return nil, grpcError(mapped)
			}
		} else {
			_ = authjwt.EnsureSecretLoaded()
		}

		authHeader := extractAuthorization(ctx)
		claims, err := authjwt.ValidateTokenFromHeader(authHeader)
		if err != nil {
			mapped := mapAuthError(err)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, err)
			return nil, grpcError(mapped)
		}

		if len(scopes) > 0 && !claims.HasAnyScope(scopes...) {
			mapped := mapAuthError(autherrors.ErrInsufficientPermissions)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, nil)
			return nil, grpcError(mapped)
		}

		annotateSpanWithClaims(span, claims)
		ctxWithClaims := WithClaims(ctx, claims)
		recordAuthSuccess("grpc", start)
		recordSpanSuccess(span)
		return handler(ctxWithClaims, req)
	}
}

// StreamAuthInterceptor enforces authentication (and optionally required scopes) on stream RPCs.
func StreamAuthInterceptor(requiredScopes ...string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()
		reqID := extractOrGenerateRequestID(extractRequestID(ctx))
		ctx = WithRequestID(ctx, reqID)
		ctx = WithSafeRequestFields(ctx, SafeGRPCRequestFields(ctx, info.FullMethod))
		ctx, span := startAuthSpan(ctx, "platform-auth.grpc.stream-auth", reqID)
		defer span.End()
		_ = ss.SetHeader(grpcResponseMetadata(ctx, reqID))
		cfg := authjwt.GetValidationConfig()
		if needsSecret(cfg) {
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				mapped := mapAuthError(err)
				recordAuthError("grpc", mapped, start)
				recordSpanError(span, mapped, err)
				return grpcError(mapped)
			}
		} else {
			_ = authjwt.EnsureSecretLoaded()
		}

		authHeader := extractAuthorization(ctx)
		claims, err := authjwt.ValidateTokenFromHeader(authHeader)
		if err != nil {
			mapped := mapAuthError(err)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, err)
			return grpcError(mapped)
		}

		if len(requiredScopes) > 0 && !claims.HasAllScopes(requiredScopes...) {
			mapped := mapAuthError(autherrors.ErrInsufficientPermissions)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, nil)
			return grpcError(mapped)
		}

		annotateSpanWithClaims(span, claims)
		ctx = WithClaims(ctx, claims)
		recordAuthSuccess("grpc", start)
		recordSpanSuccess(span)
		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
	}
}

// StreamAnyScopeInterceptor enforces at least one scope on stream RPCs.
func StreamAnyScopeInterceptor(scopes ...string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()
		reqID := extractOrGenerateRequestID(extractRequestID(ctx))
		ctx = WithRequestID(ctx, reqID)
		ctx = WithSafeRequestFields(ctx, SafeGRPCRequestFields(ctx, info.FullMethod))
		ctx, span := startAuthSpan(ctx, "platform-auth.grpc.stream-any-scope", reqID)
		defer span.End()
		_ = ss.SetHeader(grpcResponseMetadata(ctx, reqID))
		cfg := authjwt.GetValidationConfig()
		if needsSecret(cfg) {
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				mapped := mapAuthError(err)
				recordAuthError("grpc", mapped, start)
				recordSpanError(span, mapped, err)
				return grpcError(mapped)
			}
		} else {
			_ = authjwt.EnsureSecretLoaded()
		}

		authHeader := extractAuthorization(ctx)
		claims, err := authjwt.ValidateTokenFromHeader(authHeader)
		if err != nil {
			mapped := mapAuthError(err)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, err)
			return grpcError(mapped)
		}

		if len(scopes) > 0 && !claims.HasAnyScope(scopes...) {
			mapped := mapAuthError(autherrors.ErrInsufficientPermissions)
			recordAuthError("grpc", mapped, start)
			recordSpanError(span, mapped, nil)
			return grpcError(mapped)
		}

		annotateSpanWithClaims(span, claims)
		ctx = WithClaims(ctx, claims)
		recordAuthSuccess("grpc", start)
		recordSpanSuccess(span)
		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
	}
}

func extractAuthorization(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("authorization"); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func extractRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(strings.ToLower(requestIDHeader)); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func grpcError(mapped authMappedError) error {
	grpcCode := codes.Unauthenticated

	switch mapped.status {
	case httpStatusForbidden:
		grpcCode = codes.PermissionDenied
	case httpStatusInternal:
		grpcCode = codes.Internal
	}

	st := status.New(grpcCode, mapped.message)
	if withDetails, err := st.WithDetails(&errdetails.ErrorInfo{Reason: mapped.code}); err == nil {
		return withDetails.Err()
	}

	return st.Err()
}

const (
	httpStatusForbidden = 403
	httpStatusInternal  = 500
)

func grpcResponseMetadata(ctx context.Context, reqID string) metadata.MD {
	pairs := []string{strings.ToLower(requestIDHeader), reqID}
	if tp := traceparentFromContext(ctx); tp != "" {
		pairs = append(pairs, "traceparent", tp)
	}

	return metadata.Pairs(pairs...)
}

func traceparentFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}

	flags := sc.TraceFlags().String()
	if len(flags) == 1 {
		flags = "0" + flags
	}

	return fmt.Sprintf("00-%s-%s-%s", sc.TraceID().String(), sc.SpanID().String(), flags)
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
