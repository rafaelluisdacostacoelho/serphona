package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryAuthInterceptorSuccess(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	interceptor := middleware.UnaryAuthInterceptor()

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		claims, err := middleware.ClaimsFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return claims.UserID, nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if resp.(string) != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected response %v", resp)
	}
}

func TestUnaryAuthInterceptorMissingToken(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	interceptor := middleware.UnaryAuthInterceptor()

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestUnaryAuthInterceptorScopesDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices"})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	interceptor := middleware.UnaryAuthInterceptor("read:reports")

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if status.Convert(err).Message() != "Insufficient permissions" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func TestUnaryAnyScopeInterceptorSuccess(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:reports"})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	interceptor := middleware.UnaryAnyScopeInterceptor("read:invoices", "read:reports")

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestUnaryAnyScopeInterceptorUnauthorizedWithoutClaims(t *testing.T) {
	interceptor := middleware.UnaryAnyScopeInterceptor("read:reports")

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if status.Convert(err).Message() != "Missing authentication token" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func TestUnaryAnyScopeInterceptorInsufficientScopes(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices"})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	interceptor := middleware.UnaryAnyScopeInterceptor("read:reports")

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if status.Convert(err).Message() != "Insufficient permissions" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func TestUnaryAuthInterceptorAddsSafeRequestFields(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+token,
		"x-request-id", "req-grpc",
		"x-tenant", "t-1",
	))

	interceptor := middleware.UnaryAuthInterceptor()

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		fields, ok := middleware.SafeRequestFieldsFromContext(ctx)
		if !ok {
			return nil, errors.New("safe request fields missing")
		}
		if fields["method"] != "/svc/Method" {
			return nil, errors.New("method not set in safe fields")
		}
		if fields["request_id"] != "req-grpc" {
			return nil, errors.New("request_id not set in safe fields")
		}
		md, ok := fields["metadata"].(metadata.MD)
		if !ok {
			return nil, errors.New("metadata not stored in safe fields")
		}
		if got := md.Get("authorization"); len(got) == 0 || got[0] != "[REDACTED]" {
			return nil, errors.New("authorization not redacted")
		}
		if got := md.Get("x-request-id"); len(got) == 0 || got[0] != "req-grpc" {
			return nil, errors.New("request id not preserved in metadata")
		}

		return "ok", nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestUnaryAuthInterceptorSetsTraceparentAndRequestIDHeaders(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()
	useTestTracerProvider(t)

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))
	md := metadata.Pairs(
		"authorization", "Bearer "+token,
		"x-request-id", "req-grpc-meta",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	transport := &mockServerTransportStream{}
	ctx = grpc.NewContextWithServerTransportStream(ctx, transport)

	interceptor := middleware.UnaryAuthInterceptor()

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if got := transport.header.Get("x-request-id"); len(got) == 0 || got[0] != "req-grpc-meta" {
		t.Fatalf("expected x-request-id header to be propagated")
	}
	if got := transport.header.Get("traceparent"); len(got) == 0 {
		t.Fatalf("expected traceparent header to be set")
	}
}

func TestUnaryAuthInterceptorErrorContainsErrorInfo(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()
	useTestTracerProvider(t)

	md := metadata.Pairs("x-request-id", "req-grpc-error")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	transport := &mockServerTransportStream{}
	ctx = grpc.NewContextWithServerTransportStream(ctx, transport)

	interceptor := middleware.UnaryAuthInterceptor()

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if err == nil {
		t.Fatalf("expected error for missing token")
	}

	st, _ := status.FromError(err)
	var info *errdetails.ErrorInfo
	for _, d := range st.Details() {
		if ei, ok := d.(*errdetails.ErrorInfo); ok {
			info = ei
			break
		}
	}

	if info == nil {
		t.Fatalf("expected ErrorInfo details to be present")
	}
	if info.Reason != autherrors.CodeMissingToken {
		t.Fatalf("expected reason %s, got %s", autherrors.CodeMissingToken, info.Reason)
	}

	if got := transport.header.Get("x-request-id"); len(got) == 0 || got[0] != "req-grpc-error" {
		t.Fatalf("expected x-request-id header to be set on error path")
	}
	if got := transport.header.Get("traceparent"); len(got) == 0 {
		t.Fatalf("expected traceparent header to be set on error path")
	}
}

func TestStreamAuthInterceptorSuccess(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	stream := &mockServerStream{ctx: ctx}

	interceptor := middleware.StreamAuthInterceptor()
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		claims, err := middleware.ClaimsFromContext(ss.Context())
		if err != nil {
			return err
		}
		if claims.UserID == "" {
			return errors.New("missing claims in stream context")
		}
		if _, err := middleware.RequestIDFromContext(ss.Context()); err != nil {
			return errors.New("missing request id in stream context")
		}
		return nil
	}

	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream", IsServerStream: true}, handler)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(stream.header.Get("x-request-id")) == 0 {
		t.Fatalf("expected x-request-id header to be set")
	}
}

func TestStreamAuthInterceptorMissingToken(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	stream := &mockServerStream{ctx: context.Background()}
	interceptor := middleware.StreamAuthInterceptor()

	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream", IsServerStream: true}, func(srv interface{}, ss grpc.ServerStream) error {
		return nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestStreamAuthInterceptorScopesDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices"})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	stream := &mockServerStream{ctx: ctx}

	interceptor := middleware.StreamAuthInterceptor("read:reports")

	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream", IsServerStream: true}, func(srv interface{}, ss grpc.ServerStream) error {
		return nil
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if status.Convert(err).Message() != "Insufficient permissions" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func TestStreamAnyScopeInterceptorDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices"})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	stream := &mockServerStream{ctx: ctx}

	interceptor := middleware.StreamAnyScopeInterceptor("read:reports")

	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream", IsServerStream: true}, func(srv interface{}, ss grpc.ServerStream) error {
		return nil
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if status.Convert(err).Message() != "Insufficient permissions" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func TestStreamAnyScopeInterceptorSuccess(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:reports"})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token, "x-request-id", "stream-success"))
	stream := &mockServerStream{ctx: ctx}

	interceptor := middleware.StreamAnyScopeInterceptor("read:reports", "read:invoices")

	handled := false
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream", IsServerStream: true}, func(srv interface{}, ss grpc.ServerStream) error {
		handled = true
		claims, err := middleware.ClaimsFromContext(ss.Context())
		if err != nil {
			return err
		}
		if claims.UserID == "" {
			return errors.New("claims missing in stream")
		}
		if _, err := middleware.RequestIDFromContext(ss.Context()); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !handled {
		t.Fatalf("expected handler to run")
	}
	if got := stream.header.Get("x-request-id"); len(got) == 0 || got[0] == "" {
		t.Fatalf("expected x-request-id header to be set")
	}
}

func TestUnaryAuthInterceptorMissingSecret(t *testing.T) {
	authjwt.SetSecret("")
	authjwt.ResetSecretOnceForTests()

	interceptor := middleware.UnaryAuthInterceptor()
	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
	if status.Convert(err).Message() != "Auth configuration missing" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func TestUnaryAuthInterceptorJWKSFetchFailureMapsError(t *testing.T) {
	authjwt.SetSecret("")
	authjwt.ResetSecretOnceForTests()

	priv := rsaKeyPair(t)
	token := signedRSATokenMiddleware(t, priv, "kid-1", time.Now().Add(time.Hour))

	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs: []string{"RS256"},
		JWKSURL:     "http://127.0.0.1:0/unreachable",
	})
	t.Cleanup(authjwt.ResetValidationConfig)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	interceptor := middleware.UnaryAuthInterceptor()

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated due to JWKS fetch failure, got %v", err)
	}
	if status.Convert(err).Message() != "Unable to validate authentication token" {
		t.Fatalf("unexpected message %s", status.Convert(err).Message())
	}
}

func rsaKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	return key
}

func signedRSATokenMiddleware(t *testing.T, privateKey *rsa.PrivateKey, kid string, exp time.Time) string {
	t.Helper()

	claims := jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(exp)}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign rsa token: %v", err)
	}

	return signed
}

type mockServerStream struct {
	ctx     context.Context
	header  metadata.MD
	trailer metadata.MD
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerStream) SetTrailer(md metadata.MD) {
	m.trailer = metadata.Join(m.trailer, md)
}

func (m *mockServerStream) Context() context.Context { return m.ctx }

func (m *mockServerStream) SendMsg(interface{}) error { return nil }

func (m *mockServerStream) RecvMsg(interface{}) error { return nil }

type mockServerTransportStream struct {
	header  metadata.MD
	trailer metadata.MD
}

func (m *mockServerTransportStream) Method() string { return "/svc/Method" }

func (m *mockServerTransportStream) SetHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerTransportStream) SendHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

func (m *mockServerTransportStream) SetTrailer(md metadata.MD) error {
	m.trailer = metadata.Join(m.trailer, md)
	return nil
}

func useTestTracerProvider(t *testing.T) {
	t.Helper()

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
	})
}
