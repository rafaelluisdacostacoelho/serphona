# Observability Best Practices for Go Services

This document outlines observability best practices for Go microservices in the Voice of Customer platform.

## Table of Contents

1. [Structured Logging](#structured-logging)
2. [Metrics](#metrics)
3. [Distributed Tracing](#distributed-tracing)
4. [Health Checks](#health-checks)
5. [Multi-tenant Context](#multi-tenant-context)

---

## Structured Logging

### Logger Implementation

We use `zap` for high-performance structured logging:

```go
// pkg/logger/logger.go
package logger

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// New creates a new logger instance.
func New(level, environment string) (*zap.Logger, error) {
    var config zap.Config
    
    if environment == "production" {
        config = zap.NewProductionConfig()
        config.EncoderConfig.TimeKey = "timestamp"
        config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    } else {
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    }
    
    // Set log level
    switch level {
    case "debug":
        config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
    case "warn":
        config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
    case "error":
        config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
    default:
        config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
    }
    
    return config.Build()
}

// WithContext returns a logger with context fields.
func WithContext(logger *zap.Logger, requestID, tenantID string) *zap.Logger {
    return logger.With(
        zap.String("request_id", requestID),
        zap.String("tenant_id", tenantID),
    )
}
```

### Logging Best Practices

1. **Always include correlation IDs**:
```go
log.Info("processing request",
    zap.String("request_id", requestID),
    zap.String("tenant_id", tenantID),
    zap.String("user_id", userID),
)
```

2. **Log at appropriate levels**:
- `Debug`: Detailed debugging info (disabled in production)
- `Info`: General operational events
- `Warn`: Potentially problematic situations
- `Error`: Error events (always include error object)

3. **Include relevant context**:
```go
log.Error("failed to create tenant",
    zap.Error(err),
    zap.String("request_id", requestID),
    zap.String("email", req.Email),
    zap.Duration("latency", time.Since(start)),
)
```

4. **Never log sensitive data**:
```go
// BAD - logs password
log.Info("user login", zap.String("password", password))

// GOOD - mask sensitive fields
log.Info("user login", zap.String("email", email))
```

---

## Metrics

### Prometheus Metrics Implementation

```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP metrics
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status", "tenant_id"},
    )
    
    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status"},
    )
    
    // Business metrics
    TenantsCreated = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "tenants_created_total",
            Help: "Total number of tenants created",
        },
        []string{"plan"},
    )
    
    ActiveTenants = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_tenants",
            Help: "Number of active tenants",
        },
    )
    
    // Database metrics
    DBConnectionsOpen = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "db_connections_open",
            Help: "Number of open database connections",
        },
    )
    
    DBQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
        []string{"query", "table"},
    )
    
    // Kafka metrics
    KafkaMessagesPublished = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kafka_messages_published_total",
            Help: "Total Kafka messages published",
        },
        []string{"topic"},
    )
    
    KafkaPublishDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "kafka_publish_duration_seconds",
            Help:    "Kafka message publish duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"topic"},
    )
)
```

### Metrics Middleware

```go
// internal/adapter/http/middleware/metrics.go
package middleware

import (
    "net/http"
    "strconv"
    "time"
    
    "tenant-manager/pkg/metrics"
)

func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(wrapped, r)
        
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(wrapped.statusCode)
        tenantID := r.Header.Get("X-Tenant-ID")
        
        metrics.HTTPRequestsTotal.WithLabelValues(
            r.Method, r.URL.Path, status, tenantID,
        ).Inc()
        
        metrics.HTTPRequestDuration.WithLabelValues(
            r.Method, r.URL.Path, status,
        ).Observe(duration)
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

### Key Metrics to Track

| Metric Type | Examples |
|-------------|----------|
| **RED Metrics** | Request rate, Error rate, Duration |
| **USE Metrics** | Utilization, Saturation, Errors |
| **Business Metrics** | Tenants created, API keys issued, Calls made |
| **SLI Metrics** | Availability, Latency percentiles, Error budget |

---

## Distributed Tracing

### OpenTelemetry Integration

```go
// pkg/tracing/tracing.go
package tracing

import (
    "context"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
    "go.opentelemetry.io/otel/trace"
)

// InitTracer initializes OpenTelemetry tracer.
func InitTracer(ctx context.Context, serviceName, environment, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
    client := otlptracegrpc.NewClient(
        otlptracegrpc.WithEndpoint(otlpEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    
    exporter, err := otlptrace.New(ctx, client)
    if err != nil {
        return nil, err
    }
    
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(serviceName),
            semconv.DeploymentEnvironment(environment),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()), // Adjust for production
    )
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    
    return tp, nil
}

// StartSpan starts a new span with common attributes.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    tracer := otel.Tracer("tenant-manager")
    return tracer.Start(ctx, name, opts...)
}

// AddTenantID adds tenant ID to the current span.
func AddTenantID(ctx context.Context, tenantID string) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.String("tenant.id", tenantID))
}
```

### Tracing in Handlers

```go
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracing.StartSpan(r.Context(), "TenantHandler.Create")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("http.method", r.Method),
        attribute.String("http.url", r.URL.String()),
    )
    
    // ... handler logic
    
    // Record errors
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
}
```

### Tracing in Repository

```go
func (r *TenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
    ctx, span := tracing.StartSpan(ctx, "TenantRepository.Create",
        trace.WithAttributes(
            attribute.String("db.system", "postgresql"),
            attribute.String("db.operation", "INSERT"),
            attribute.String("db.table", "tenants"),
        ),
    )
    defer span.End()
    
    // ... database operation
}
```

---

## Health Checks

### Health Check Implementation

```go
// internal/adapter/http/handler/health.go
package handler

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
)

type HealthHandler struct {
    db    *pgxpool.Pool
    redis *redis.Client
}

type HealthResponse struct {
    Status    string            `json:"status"`
    Timestamp string            `json:"timestamp"`
    Version   string            `json:"version"`
    Checks    map[string]Check  `json:"checks"`
}

type Check struct {
    Status   string `json:"status"`
    Latency  string `json:"latency,omitempty"`
    Message  string `json:"message,omitempty"`
}

// Liveness check - is the service running?
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ok",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })
}

// Readiness check - is the service ready to accept traffic?
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    response := HealthResponse{
        Status:    "ok",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Version:   "1.0.0",
        Checks:    make(map[string]Check),
    }
    
    allHealthy := true
    
    // Check PostgreSQL
    dbCheck := h.checkDatabase(ctx)
    response.Checks["database"] = dbCheck
    if dbCheck.Status != "ok" {
        allHealthy = false
    }
    
    // Check Redis
    redisCheck := h.checkRedis(ctx)
    response.Checks["redis"] = redisCheck
    if redisCheck.Status != "ok" {
        allHealthy = false
    }
    
    w.Header().Set("Content-Type", "application/json")
    if allHealthy {
        response.Status = "ok"
        w.WriteHeader(http.StatusOK)
    } else {
        response.Status = "degraded"
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(response)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) Check {
    start := time.Now()
    err := h.db.Ping(ctx)
    latency := time.Since(start)
    
    if err != nil {
        return Check{
            Status:  "error",
            Latency: latency.String(),
            Message: err.Error(),
        }
    }
    
    return Check{
        Status:  "ok",
        Latency: latency.String(),
    }
}

func (h *HealthHandler) checkRedis(ctx context.Context) Check {
    start := time.Now()
    _, err := h.redis.Ping(ctx).Result()
    latency := time.Since(start)
    
    if err != nil {
        return Check{
            Status:  "error",
            Latency: latency.String(),
            Message: err.Error(),
        }
    }
    
    return Check{
        Status:  "ok",
        Latency: latency.String(),
    }
}
```

### Kubernetes Probes Configuration

```yaml
# In Helm values.yaml or Kubernetes deployment
livenessProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 10
  periodSeconds: 15
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: http
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

startupProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 30
```

---

## Multi-tenant Context

### Tenant Context Middleware

```go
// internal/adapter/http/middleware/tenant.go
package middleware

import (
    "context"
    "net/http"
    
    "github.com/google/uuid"
)

type contextKey string

const (
    TenantIDKey   contextKey = "tenant_id"
    RequestIDKey  contextKey = "request_id"
    UserIDKey     contextKey = "user_id"
)

// TenantContextMiddleware extracts and validates tenant context.
func TenantContextMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant ID from header or JWT claims
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            // Try to get from JWT claims
            if claims := GetJWTClaims(r.Context()); claims != nil {
                tenantID = claims.TenantID
            }
        }
        
        // Validate tenant ID format
        if tenantID != "" {
            if _, err := uuid.Parse(tenantID); err != nil {
                http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
                return
            }
        }
        
        // Add to context
        ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// GetTenantID retrieves tenant ID from context.
func GetTenantID(ctx context.Context) string {
    if id, ok := ctx.Value(TenantIDKey).(string); ok {
        return id
    }
    return ""
}

// GetRequestID retrieves request ID from context.
func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(RequestIDKey).(string); ok {
        return id
    }
    return ""
}
```

### Correlation ID Middleware

```go
// internal/adapter/http/middleware/correlation.go
package middleware

import (
    "context"
    "net/http"
    
    "github.com/google/uuid"
)

const (
    RequestIDHeader = "X-Request-ID"
    TraceIDHeader   = "X-Trace-ID"
)

// CorrelationMiddleware adds request/trace IDs to the context.
func CorrelationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get or generate request ID
        requestID := r.Header.Get(RequestIDHeader)
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        // Get or generate trace ID
        traceID := r.Header.Get(TraceIDHeader)
        if traceID == "" {
            traceID = uuid.New().String()
        }
        
        // Add to response headers
        w.Header().Set(RequestIDHeader, requestID)
        w.Header().Set(TraceIDHeader, traceID)
        
        // Add to context
        ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
        ctx = context.WithValue(ctx, "trace_id", traceID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## Dashboard Examples

### Grafana Dashboard Queries

```promql
# Request Rate by Tenant
sum(rate(http_requests_total{service="tenant-manager"}[5m])) by (tenant_id)

# Error Rate
sum(rate(http_requests_total{service="tenant-manager",status=~"5.."}[5m])) 
/ sum(rate(http_requests_total{service="tenant-manager"}[5m])) * 100

# P99 Latency
histogram_quantile(0.99, 
  sum(rate(http_request_duration_seconds_bucket{service="tenant-manager"}[5m])) by (le)
)

# Database Connection Pool
db_connections_open{service="tenant-manager"}

# Tenant Growth
increase(tenants_created_total[24h])
```

### Alerting Rules

```yaml
groups:
  - name: tenant-manager-alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{service="tenant-manager",status=~"5.."}[5m])) 
          / sum(rate(http_requests_total{service="tenant-manager"}[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: High error rate in tenant-manager service
          
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99, 
            sum(rate(http_request_duration_seconds_bucket{service="tenant-manager"}[5m])) by (le)
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High latency in tenant-manager service
          
      - alert: DatabaseConnectionExhausted
        expr: db_connections_open{service="tenant-manager"} > 20
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: Database connection pool near exhaustion
```

---

## Summary Checklist

### Logging
- [ ] Use structured logging (zap)
- [ ] Include correlation IDs in all logs
- [ ] Log at appropriate levels
- [ ] Never log sensitive data
- [ ] Include relevant context (tenant_id, request_id)

### Metrics
- [ ] Expose Prometheus endpoint
- [ ] Track RED metrics (Rate, Errors, Duration)
- [ ] Track business metrics
- [ ] Include tenant_id label where appropriate
- [ ] Set up alerting rules

### Tracing
- [ ] Implement OpenTelemetry
- [ ] Propagate trace context
- [ ] Add spans for key operations
- [ ] Include relevant attributes
- [ ] Record errors in spans

### Health Checks
- [ ] Implement liveness probe
- [ ] Implement readiness probe
- [ ] Check all dependencies
- [ ] Configure Kubernetes probes

### Multi-tenancy
- [ ] Extract tenant context from requests
- [ ] Pass tenant_id through all layers
- [ ] Include tenant_id in logs/metrics/traces
- [ ] Validate tenant access
