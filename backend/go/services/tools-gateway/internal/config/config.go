package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for tools-gateway.
type Config struct {
	HTTPAddr string `envconfig:"HTTP_ADDR" default:":8085"`

	Auth      AuthConfig
	Execution ExecutionConfig
}

// AuthConfig maps platform-auth settings.
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

// ExecutionConfig governs outbound execution guardrails.
type ExecutionConfig struct {
	AllowedHosts    []string `envconfig:"EXEC_ALLOWED_HOSTS"`
	MaxPayloadBytes int      `envconfig:"EXEC_MAX_PAYLOAD_BYTES" default:"1048576"`
	AllowedMethods  []string `envconfig:"EXEC_ALLOWED_METHODS" default:"GET,POST,PUT,PATCH,DELETE"`
	BlockedMethods  []string `envconfig:"EXEC_BLOCKED_METHODS"`
	AllowedHeaders  []string `envconfig:"EXEC_ALLOWED_HEADERS"`
	BlockedHeaders  []string `envconfig:"EXEC_BLOCKED_HEADERS"`
	MaxQueryParams  int      `envconfig:"EXEC_MAX_QUERY_PARAMS" default:"25"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
