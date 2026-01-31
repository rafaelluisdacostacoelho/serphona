package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds service configuration loaded from environment variables.
type Config struct {
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	HTTP        HTTPConfig
	GRPC        GRPCConfig
	Metrics     MetricsConfig
	Tracing     TracingConfig
	Auth        AuthConfig
	Pprof       PprofConfig
}

// HTTPConfig controls the HTTP server.
type HTTPConfig struct {
	Host              string        `envconfig:"HTTP_HOST" default:"0.0.0.0"`
	Port              int           `envconfig:"HTTP_PORT" default:"8088"`
	ReadTimeout       time.Duration `envconfig:"HTTP_READ_TIMEOUT" default:"10s"`
	ReadHeaderTimeout time.Duration `envconfig:"HTTP_READ_HEADER_TIMEOUT" default:"5s"`
	WriteTimeout      time.Duration `envconfig:"HTTP_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout       time.Duration `envconfig:"HTTP_IDLE_TIMEOUT" default:"120s"`
	MaxHeaderBytes    int           `envconfig:"HTTP_MAX_HEADER_BYTES" default:"1048576"`
	ShutdownTimeout   time.Duration `envconfig:"HTTP_SHUTDOWN_TIMEOUT" default:"30s"`
}

// GRPCConfig controls the gRPC server.
type GRPCConfig struct {
	Host                  string        `envconfig:"GRPC_HOST" default:"0.0.0.0"`
	Port                  int           `envconfig:"GRPC_PORT" default:"9098"`
	MaxRecvMsgSizeMB      int           `envconfig:"GRPC_MAX_RECV_MSG_SIZE_MB" default:"10"`
	MaxSendMsgSizeMB      int           `envconfig:"GRPC_MAX_SEND_MSG_SIZE_MB" default:"10"`
	ConnectionTimeout     time.Duration `envconfig:"GRPC_CONNECTION_TIMEOUT" default:"5s"`
	DefaultRequestTimeout time.Duration `envconfig:"GRPC_DEFAULT_REQUEST_TIMEOUT" default:"10s"`
	ReflectionEnabled     bool          `envconfig:"GRPC_REFLECTION_ENABLED" default:"false"`
}

// MetricsConfig exposes Prometheus metrics.
type MetricsConfig struct {
	Enabled bool   `envconfig:"METRICS_ENABLED" default:"true"`
	Path    string `envconfig:"METRICS_PATH" default:"/metrics"`
}

// TracingConfig controls OpenTelemetry exporter settings.
type TracingConfig struct {
	Enabled    bool    `envconfig:"TRACING_ENABLED" default:"true"`
	Endpoint   string  `envconfig:"TRACING_ENDPOINT" default:"localhost:4317"`
	Insecure   bool    `envconfig:"TRACING_INSECURE" default:"true"`
	SampleRate float64 `envconfig:"TRACING_SAMPLER" default:"1.0"`
}

// PprofConfig toggles pprof handlers.
type PprofConfig struct {
	Enabled bool `envconfig:"PPROF_ENABLED" default:"false"`
}

// AuthConfig maps platform-auth validation settings.
type AuthConfig struct {
	JWTSecret       string        `envconfig:"JWT_SECRET"`
	AllowedAlgs     []string      `envconfig:"JWT_ALLOWED_ALGS" default:"HS256"`
	Issuer          string        `envconfig:"JWT_ISSUER"`
	Audience        string        `envconfig:"JWT_AUDIENCE"`
	ServiceAudience string        `envconfig:"JWT_SERVICE_AUDIENCE"`
	ClockSkew       time.Duration `envconfig:"JWT_CLOCK_SKEW" default:"30s"`
	MaxTokenBytes   int           `envconfig:"JWT_MAX_TOKEN_BYTES" default:"4096"`
	JWKSURL         string        `envconfig:"JWKS_URL"`
	JWKSCacheTTL    time.Duration `envconfig:"JWKS_CACHE_TTL" default:"5m"`
	AllowedKIDs     []string      `envconfig:"JWKS_ALLOWED_KIDS"`
	RequiredScopes  []string      `envconfig:"JWT_REQUIRED_SCOPES"`
	TenantClaim     string        `envconfig:"TENANT_CLAIM" default:"tenant_id"`
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
