package middleware

import (
    "google.golang.org/grpc"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// GRPCUnary returns a unary server interceptor with tracing enabled.
func GRPCUnary() grpc.UnaryServerInterceptor {
    return otelgrpc.UnaryServerInterceptor()
}

// GRPCStream returns a stream server interceptor with tracing enabled.
func GRPCStream() grpc.StreamServerInterceptor {
    return otelgrpc.StreamServerInterceptor()
}
