package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all runtime configuration for tools-manager.
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Auth          AuthConfig
	Observability ObservabilityConfig
	Secrets       SecretsConfig
	Integrations  IntegrationsConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port int    `envconfig:"SERVER_PORT" default:"8087"`
	Env  string `envconfig:"ENV" default:"development"`
}

// DatabaseConfig holds Postgres connection settings.
type DatabaseConfig struct {
	URL      string `envconfig:"DATABASE_URL"`
	MaxConns int32  `envconfig:"DB_MAX_CONNS" default:"10"`
}

// AuthConfig holds platform-auth validation settings.
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

// ObservabilityConfig exposes health/metrics/pprof knobs.
type ObservabilityConfig struct {
	MetricsPath string  `envconfig:"METRICS_PATH" default:"/metrics"`
	HealthPath  string  `envconfig:"HEALTH_PATH" default:"/health"`
	ReadyPath   string  `envconfig:"READY_PATH" default:"/ready"`
	AuditSample float64 `envconfig:"AUDIT_SAMPLE" default:"1"`
}

// SecretsConfig controls secret storage integration.
type SecretsConfig struct {
	EncryptionKey string        `envconfig:"SECRETS_ENCRYPTION_KEY"`
	CacheTTL      time.Duration `envconfig:"SECRETS_CACHE_TTL" default:"5m"`
	UseVault      bool          `envconfig:"SECRETS_USE_VAULT" default:"false"`
	VaultAddress  string        `envconfig:"SECRETS_VAULT_ADDR"`
	VaultToken    string        `envconfig:"SECRETS_VAULT_TOKEN"`
	VaultMount    string        `envconfig:"SECRETS_VAULT_MOUNT" default:"secret"`
	VaultPrefix   string        `envconfig:"SECRETS_VAULT_PREFIX" default:"tools-manager"`
}

// IntegrationsConfig holds outbound sync targets.
type IntegrationsConfig struct {
	EventsWebhookURL string `envconfig:"EVENTS_WEBHOOK_URL"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
