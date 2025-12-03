// Package middleware provides HTTP middleware implementations.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"tenant-manager/internal/application/tenant"
)

// TenantMiddleware validates tenant context.
type TenantMiddleware struct {
	service *tenant.Service
}

// NewTenantMiddleware creates a new tenant middleware.
func NewTenantMiddleware(service *tenant.Service) *TenantMiddleware {
	return &TenantMiddleware{service: service}
}

// Handle is the middleware handler function.
func (m *TenantMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For now, just pass through - implement tenant validation here
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests.
type LoggingMiddleware struct {
	logger *zap.Logger
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

// Handle is the middleware handler function.
func (m *LoggingMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.logger.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics.
type RecoveryMiddleware struct {
	logger *zap.Logger
}

// NewRecoveryMiddleware creates a new recovery middleware.
func NewRecoveryMiddleware(logger *zap.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{logger: logger}
}

// Handle is the middleware handler function.
func (m *RecoveryMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("panic recovered", zap.Any("error", err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CorrelationMiddleware adds correlation ID to requests.
type CorrelationMiddleware struct{}

// NewCorrelationMiddleware creates a new correlation middleware.
func NewCorrelationMiddleware() *CorrelationMiddleware {
	return &CorrelationMiddleware{}
}

// Handle is the middleware handler function.
func (m *CorrelationMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GRPCLoggingInterceptor logs gRPC requests.
func GRPCLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Info("gRPC request", zap.String("method", info.FullMethod))
		return handler(ctx, req)
	}
}

// GRPCRecoveryInterceptor recovers from panics in gRPC handlers.
func GRPCRecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered", zap.Any("error", r))
				err = grpc.Errorf(2, "internal error") // Using error code 2 (Unknown)
			}
		}()
		return handler(ctx, req)
	}
}

// GRPCCorrelationInterceptor adds correlation ID to gRPC requests.
func GRPCCorrelationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := uuid.New().String()
		ctx = context.WithValue(ctx, "request_id", requestID)
		return handler(ctx, req)
	}
}
