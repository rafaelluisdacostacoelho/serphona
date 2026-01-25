package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for tools-gateway.
type Config struct {
	HTTPAddr     string `envconfig:"HTTP_ADDR" default:":8085"`
	MaxBodyBytes int    `envconfig:"HTTP_MAX_BODY_BYTES" default:"1048576"`

	Auth      AuthConfig
	Execution ExecutionConfig
	Billing   BillingConfig
	RAGIngest RAGIngestConfig
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
	AllowedHosts      []string `envconfig:"EXEC_ALLOWED_HOSTS"`
	MaxPayloadBytes   int      `envconfig:"EXEC_MAX_PAYLOAD_BYTES" default:"1048576"`
	MaxTimeoutSeconds int      `envconfig:"EXEC_MAX_TIMEOUT_SECONDS" default:"30"`
	AllowedMethods    []string `envconfig:"EXEC_ALLOWED_METHODS" default:"GET,POST,PUT,PATCH,DELETE"`
	BlockedMethods    []string `envconfig:"EXEC_BLOCKED_METHODS"`
	AllowedHeaders    []string `envconfig:"EXEC_ALLOWED_HEADERS"`
	BlockedHeaders    []string `envconfig:"EXEC_BLOCKED_HEADERS"`
	MaxQueryParams    int      `envconfig:"EXEC_MAX_QUERY_PARAMS" default:"25"`
}

// BillingConfig controls usage event publishing to billing/analytics sinks.
type BillingConfig struct {
	UsageEndpoint   string        `envconfig:"USAGE_PUBLISH_ENDPOINT"`
	Enabled         bool          `envconfig:"USAGE_PUBLISH_ENABLED" default:"true"`
	AuthToken       string        `envconfig:"USAGE_PUBLISH_TOKEN"`
	Timeout         time.Duration `envconfig:"USAGE_PUBLISH_TIMEOUT" default:"3s"`
	RetryMax        uint          `envconfig:"USAGE_PUBLISH_RETRY_MAX" default:"3"`
	RetryBackoff    time.Duration `envconfig:"USAGE_PUBLISH_RETRY_BACKOFF" default:"250ms"`
	BreakerEnabled  bool          `envconfig:"USAGE_PUBLISH_BREAKER_ENABLED" default:"true"`
	BreakerFailures uint          `envconfig:"USAGE_PUBLISH_BREAKER_FAILURES" default:"3"`
	BreakerReset    time.Duration `envconfig:"USAGE_PUBLISH_BREAKER_RESET" default:"30s"`
	KafkaEnabled    bool          `envconfig:"USAGE_PUBLISH_KAFKA_ENABLED" default:"false"`
	KafkaBrokers    []string      `envconfig:"USAGE_PUBLISH_KAFKA_BROKERS" default:"localhost:9092"`
	KafkaTopic      string        `envconfig:"USAGE_PUBLISH_KAFKA_TOPIC" default:"tools.usage"`
	KafkaClientID   string        `envconfig:"USAGE_PUBLISH_KAFKA_CLIENT_ID" default:"tools-gateway"`
}

// RAGIngestConfig controls publishing rag.ingestion.requested events.
type RAGIngestConfig struct {
	Enabled            bool          `envconfig:"RAG_INGEST_ENABLED" default:"true"`
	KafkaEnabled       bool          `envconfig:"RAG_INGEST_KAFKA_ENABLED" default:"true"`
	KafkaBrokers       []string      `envconfig:"RAG_INGEST_KAFKA_BROKERS" default:"localhost:9092"`
	KafkaTopic         string        `envconfig:"RAG_INGEST_KAFKA_TOPIC" default:"rag.ingestion.requested"`
	KafkaClient        string        `envconfig:"RAG_INGEST_KAFKA_CLIENT_ID" default:"tools-gateway"`
	KafkaSASLMechanism string        `envconfig:"RAG_INGEST_KAFKA_SASL_MECHANISM"`
	KafkaSASLUsername  string        `envconfig:"RAG_INGEST_KAFKA_SASL_USERNAME"`
	KafkaSASLPassword  string        `envconfig:"RAG_INGEST_KAFKA_SASL_PASSWORD"`
	RetryMax           uint          `envconfig:"RAG_INGEST_RETRY_MAX" default:"3"`
	RetryBackoff       time.Duration `envconfig:"RAG_INGEST_RETRY_BACKOFF" default:"250ms"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
