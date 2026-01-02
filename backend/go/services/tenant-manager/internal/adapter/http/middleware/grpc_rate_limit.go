package middleware

import (
	"context"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TenantUnaryRateLimit applies per-tenant limits to unary RPCs.
func TenantUnaryRateLimit(limiter *TenantRateLimiter) grpc.UnaryServerInterceptor {
	if limiter == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		tenantID, err := authmw.TenantIDFromContext(ctx)
		if err == nil && tenantID != "" {
			if !limiter.Allow(tenantID) {
				return nil, status.Error(codes.ResourceExhausted, "tenant rate limit exceeded")
			}
		}
		return handler(ctx, req)
	}
}

// TenantStreamRateLimit applies per-tenant limits to stream RPCs (checked at stream open).
func TenantStreamRateLimit(limiter *TenantRateLimiter) grpc.StreamServerInterceptor {
	if limiter == nil {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		tenantID, err := authmw.TenantIDFromContext(ss.Context())
		if err == nil && tenantID != "" {
			if !limiter.Allow(tenantID) {
				return status.Error(codes.ResourceExhausted, "tenant rate limit exceeded")
			}
		}
		return handler(srv, ss)
	}
}
