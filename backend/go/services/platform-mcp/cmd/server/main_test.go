package main

import (
	"context"
	"strings"
	"testing"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTenantUnaryGuardMismatch(t *testing.T) {
	interceptor := tenantUnaryGuard()
	ctx := ctxWithClaimsAndTenant("tenant-1", "tenant-2")

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(context.Context, interface{}) (interface{}, error) {
		t.Fatalf("handler should not be called on mismatch")
		return nil, nil
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", status.Code(err))
	}
}

func TestTenantUnaryGuardSuccess(t *testing.T) {
	interceptor := tenantUnaryGuard()
	ctx := ctxWithClaimsAndTenant("tenant-1", "")

	called := false
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, _ interface{}) (interface{}, error) {
		called = true
		tenant, terr := authmw.TenantIDFromContext(ctx)
		if terr != nil {
			t.Fatalf("expected tenant in context, got err: %v", terr)
		}
		if tenant != "tenant-1" {
			t.Fatalf("unexpected tenant propagated: %s", tenant)
		}
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !called {
		t.Fatalf("handler was not called")
	}
}

func TestTenantStreamGuardMismatch(t *testing.T) {
	interceptor := tenantStreamGuard()
	ctx := ctxWithClaimsAndTenant("tenant-1", "tenant-2")

	err := interceptor(nil, &fakeServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/svc/Stream"}, func(_ interface{}, _ grpc.ServerStream) error {
		t.Fatalf("handler should not be called on mismatch")
		return nil
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", status.Code(err))
	}
}

func TestTenantStreamGuardSuccess(t *testing.T) {
	interceptor := tenantStreamGuard()
	ctx := ctxWithClaimsAndTenant("tenant-1", "")

	called := false
	err := interceptor(nil, &fakeServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/svc/Stream"}, func(_ interface{}, ss grpc.ServerStream) error {
		called = true
		tenant, terr := authmw.TenantIDFromContext(ss.Context())
		if terr != nil {
			t.Fatalf("expected tenant in context, got err: %v", terr)
		}
		if tenant != "tenant-1" {
			t.Fatalf("unexpected tenant propagated: %s", tenant)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !called {
		t.Fatalf("handler was not called")
	}
}

func ctxWithClaimsAndTenant(claimTenant, headerTenant string) context.Context {
	md := metadata.MD{}
	if headerTenant != "" {
		md.Set(strings.ToLower(authmw.TenantIDHeader), headerTenant)
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)
	return authmw.WithClaims(ctx, &types.Claims{TenantID: claimTenant, UserID: "user-1", Service: "svc"})
}

// fakeServerStream is a minimal grpc.ServerStream for testing.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeServerStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeServerStream) SetTrailer(metadata.MD)       {}
func (f *fakeServerStream) Context() context.Context     { return f.ctx }
func (f *fakeServerStream) SendMsg(interface{}) error    { return nil }
func (f *fakeServerStream) RecvMsg(interface{}) error    { return nil }
