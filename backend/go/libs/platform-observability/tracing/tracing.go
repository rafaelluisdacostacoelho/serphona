package tracing

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "errors"
    "os"
    "strings"
    "sync/atomic"

    "google.golang.org/grpc/credentials"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/trace"

    "github.com/serphona/backend/go/libs/platform-observability/config"
)

var adaptiveSamplerInstance *adaptiveSampler

// Setup initializes an OTLP tracer provider with configurable sampling and TLS/auth options.
func Setup(ctx context.Context, cfg *config.Config) (*sdktrace.TracerProvider, func(context.Context) error, error) {
    if !cfg.TracingEnabled {
        noop := sdktrace.NewTracerProvider()
        otel.SetTracerProvider(noop)
        return noop, func(context.Context) error { return nil }, nil
    }

    clientOpts, err := buildClientOptions(cfg)
    if err != nil {
        return nil, nil, err
    }

    exporter, err := otlptracegrpc.New(ctx, clientOpts...)
    if err != nil {
        return nil, nil, err
    }

    sampler := samplerFromConfig(cfg)

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithSampler(sampler),
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            "",
            attribute.String("service.name", cfg.ServiceName),
            attribute.String("service.version", cfg.ServiceVersion),
            attribute.String("deployment.environment", cfg.Environment),
        )),
    )

    otel.SetTracerProvider(tp)
    return tp, tp.Shutdown, nil
}

// Tracer returns a tracer with service scope.
func Tracer() trace.Tracer {
    return otel.Tracer("platform-observability")
}

// UpdateAdaptiveSamplingRatio updates the adaptive sampler ratio at runtime.
func UpdateAdaptiveSamplingRatio(ratio float64) {
    if adaptiveSamplerInstance == nil {
        return
    }
    adaptiveSamplerInstance.setRatio(ratio)
}

func samplerFromConfig(cfg *config.Config) sdktrace.Sampler {
    strategy := strings.ToLower(cfg.TracingSamplerStrategy)
    switch strategy {
    case "always_on":
        return sdktrace.AlwaysSample()
    case "always_off":
        return sdktrace.NeverSample()
    case "ratio":
        return sdktrace.TraceIDRatioBased(cfg.TracingSampler)
    case "adaptive":
        adaptiveSamplerInstance = newAdaptiveSampler(cfg.TracingSampler)
        return sdktrace.ParentBased(adaptiveSamplerInstance)
    case "parent_ratio", "parentbased", "parent":
        return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TracingSampler))
    default:
        return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TracingSampler))
    }
}

func buildClientOptions(cfg *config.Config) ([]otlptracegrpc.Option, error) {
    endpoint := sanitizeEndpoint(cfg.TracingEndpoint)
    if endpoint == "" {
        return nil, errors.New("tracing endpoint is required")
    }

    opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}

    if cfg.TracingInsecure {
        opts = append(opts, otlptracegrpc.WithInsecure())
    } else {
        tlsConfig, err := buildTLSConfig(cfg)
        if err != nil {
            return nil, err
        }
        opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
    }

    headers := map[string]string{}
    if cfg.TracingBearerToken != "" {
        headers["Authorization"] = "Bearer " + cfg.TracingBearerToken
    }
    if len(headers) > 0 {
        opts = append(opts, otlptracegrpc.WithHeaders(headers))
    }

    return opts, nil
}

func buildTLSConfig(cfg *config.Config) (*tls.Config, error) {
    tlsConfig := &tls.Config{InsecureSkipVerify: cfg.TracingTLSInsecure}

    if cfg.TracingTLSCACertPath != "" {
        caData, err := os.ReadFile(cfg.TracingTLSCACertPath)
        if err != nil {
            return nil, err
        }
        pool := x509.NewCertPool()
        pool.AppendCertsFromPEM(caData)
        tlsConfig.RootCAs = pool
    }

    if cfg.TracingTLSClientCert != "" && cfg.TracingTLSClientKey != "" {
        cert, err := tls.LoadX509KeyPair(cfg.TracingTLSClientCert, cfg.TracingTLSClientKey)
        if err != nil {
            return nil, err
        }
        tlsConfig.Certificates = []tls.Certificate{cert}
    }

    return tlsConfig, nil
}

func sanitizeEndpoint(endpoint string) string {
    trimmed := strings.TrimSpace(endpoint)
    trimmed = strings.TrimPrefix(trimmed, "http://")
    trimmed = strings.TrimPrefix(trimmed, "https://")
    return trimmed
}

type adaptiveSampler struct {
    ratio atomic.Value
}

func newAdaptiveSampler(initial float64) *adaptiveSampler {
    if initial <= 0 {
        initial = 0.01
    }
    if initial > 1 {
        initial = 1
    }
    as := &adaptiveSampler{}
    as.ratio.Store(initial)
    return as
}

func (a *adaptiveSampler) setRatio(r float64) {
    if r < 0 {
        r = 0
    }
    if r > 1 {
        r = 1
    }
    a.ratio.Store(r)
}

func (a *adaptiveSampler) ShouldSample(params sdktrace.SamplingParameters) sdktrace.SamplingResult {
    ratio := a.ratio.Load().(float64)
    return sdktrace.TraceIDRatioBased(ratio).ShouldSample(params)
}

func (a *adaptiveSampler) Description() string {
    return "adaptive-ratio"
}
