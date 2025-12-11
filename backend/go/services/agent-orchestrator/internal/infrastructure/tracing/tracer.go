package tracing

import (
	"context"
	"log"
	"time"
)

// Tracer interface defines methods for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
	Close() error
}

// Span represents a trace span
type Span interface {
	SetAttribute(key string, value interface{})
	SetStatus(code StatusCode, message string)
	RecordError(err error)
	End()
}

// StatusCode represents span status
type StatusCode int

const (
	StatusCodeOK StatusCode = iota
	StatusCodeError
)

// OpenTelemetryTracer implements Tracer using OpenTelemetry (structure ready for integration)
type OpenTelemetryTracer struct {
	serviceName string
	// When OpenTelemetry library is added:
	// tracer trace.Tracer
	// provider *tracesdk.TracerProvider
}

// NewOpenTelemetryTracer creates a new OpenTelemetry tracer
func NewOpenTelemetryTracer(serviceName, endpoint string) (Tracer, error) {
	tracer := &OpenTelemetryTracer{
		serviceName: serviceName,
	}

	// TODO: Initialize OpenTelemetry when library is added
	// Example:
	// exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	// if err != nil {
	//     return nil, err
	// }
	//
	// provider := tracesdk.NewTracerProvider(
	//     tracesdk.WithBatcher(exporter),
	//     tracesdk.WithResource(resource.NewWithAttributes(
	//         semconv.SchemaURL,
	//         semconv.ServiceNameKey.String(serviceName),
	//     )),
	// )
	// otel.SetTracerProvider(provider)
	// tracer.provider = provider
	// tracer.tracer = provider.Tracer(serviceName)

	log.Printf("✅ Tracer initialized (service: %s, endpoint: %s)", serviceName, endpoint)
	return tracer, nil
}

// StartSpan starts a new span
func (t *OpenTelemetryTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	// TODO: Create actual OpenTelemetry span
	// ctx, span := t.tracer.Start(ctx, name)
	// return ctx, &otelSpan{span: span}

	span := &noOpSpan{name: name, startTime: time.Now()}
	log.Printf("🔍 Trace started: %s", name)
	return ctx, span
}

// Close closes the tracer and flushes remaining spans
func (t *OpenTelemetryTracer) Close() error {
	// TODO: Shutdown OpenTelemetry provider
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// return t.provider.Shutdown(ctx)

	log.Println("👋 Tracer closed")
	return nil
}

// noOpSpan is a no-op span implementation
type noOpSpan struct {
	name      string
	startTime time.Time
}

func (s *noOpSpan) SetAttribute(key string, value interface{}) {
	log.Printf("🔍   %s: %s = %v", s.name, key, value)
}

func (s *noOpSpan) SetStatus(code StatusCode, message string) {
	status := "OK"
	if code == StatusCodeError {
		status = "ERROR"
	}
	log.Printf("🔍   %s: status = %s (%s)", s.name, status, message)
}

func (s *noOpSpan) RecordError(err error) {
	if err != nil {
		log.Printf("🔍   %s: error = %v", s.name, err)
	}
}

func (s *noOpSpan) End() {
	duration := time.Since(s.startTime)
	log.Printf("🔍 Trace ended: %s (%.3fs)", s.name, duration.Seconds())
}

// NoOpTracer is a no-op tracer implementation
type NoOpTracer struct{}

// NewNoOpTracer creates a no-op tracer
func NewNoOpTracer() Tracer {
	return &NoOpTracer{}
}

func (t *NoOpTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &noOpSpan{name: name, startTime: time.Now()}
}

func (t *NoOpTracer) Close() error {
	return nil
}
