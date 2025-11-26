// Package config provides configuration management for the service.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the service.
type Config struct {
	// Service metadata
	ServiceName string `envconfig:"SERVICE_NAME" default:"tenant-management"`
	Version     string `envconfig:"VERSION" default:"1.0.0"`
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`

	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// Redis configuration
	Redis RedisConfig

	// Kafka configuration
	Kafka KafkaConfig

	// JWT configuration
	JWT JWTConfig

	// Metrics configuration
	Metrics MetricsConfig
}

// ServerConfig holds HTTP/gRPC server configuration.
type ServerConfig struct {
	Host            string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port            int           `envconfig:"SERVER_PORT" default:"8080"`
	GRPCPort        int           `envconfig:"GRPC_PORT" default:"9090"`
	ReadTimeout     time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout     time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	URL             string        `envconfig:"DATABASE_URL" required:"true"`
	Host            string        `envconfig:"DB_HOST" default:"localhost"`
	Port            int           `envconfig:"DB_PORT" default:"5432"`
	User            string        `envconfig:"DB_USER" default:"postgres"`
	Password        string        `envconfig:"DB_PASSWORD"`
	Database        string        `envconfig:"DB_NAME" default:"tenant_management"`
	SSLMode         string        `envconfig:"DB_SSL_MODE" default:"require"`
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"10"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `envconfig:"DB_CONN_MAX_IDLE_TIME" default:"1m"`
	MigrationsPath  string        `envconfig:"DB_MIGRATIONS_PATH" default:"migrations"`
	AutoMigrate     bool          `envconfig:"DB_AUTO_MIGRATE" default:"false"`
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	URL         string        `envconfig:"REDIS_URL" default:"redis://localhost:6379/0"`
	Host        string        `envconfig:"REDIS_HOST" default:"localhost"`
	Port        int           `envconfig:"REDIS_PORT" default:"6379"`
	Password    string        `envconfig:"REDIS_PASSWORD"`
	DB          int           `envconfig:"REDIS_DB" default:"0"`
	MaxRetries  int           `envconfig:"REDIS_MAX_RETRIES" default:"3"`
	PoolSize    int           `envconfig:"REDIS_POOL_SIZE" default:"10"`
	DialTimeout time.Duration `envconfig:"REDIS_DIAL_TIMEOUT" default:"5s"`
	CacheTTL    time.Duration `envconfig:"REDIS_CACHE_TTL" default:"15m"`
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	Brokers     []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	TopicPrefix string   `envconfig:"KAFKA_TOPIC_PREFIX" default:"voicecustomer"`
	GroupID     string   `envconfig:"KAFKA_GROUP_ID" default:"tenant-management"`

	// Producer settings
	ProducerAcks      string `envconfig:"KAFKA_PRODUCER_ACKS" default:"all"`
	ProducerRetries   int    `envconfig:"KAFKA_PRODUCER_RETRIES" default:"3"`
	ProducerBatchSize int    `envconfig:"KAFKA_PRODUCER_BATCH_SIZE" default:"16384"`
	ProducerLingerMs  int    `envconfig:"KAFKA_PRODUCER_LINGER_MS" default:"10"`

	// Security settings
	SecurityProtocol string `envconfig:"KAFKA_SECURITY_PROTOCOL" default:"PLAINTEXT"`
	SASLMechanism    string `envconfig:"KAFKA_SASL_MECHANISM"`
	SASLUsername     string `envconfig:"KAFKA_SASL_USERNAME"`
	SASLPassword     string `envconfig:"KAFKA_SASL_PASSWORD"`

	// TLS settings
	TLSEnabled    bool   `envconfig:"KAFKA_TLS_ENABLED" default:"false"`
	TLSCACertPath string `envconfig:"KAFKA_TLS_CA_CERT_PATH"`
	TLSCertPath   string `envconfig:"KAFKA_TLS_CERT_PATH"`
	TLSKeyPath    string `envconfig:"KAFKA_TLS_KEY_PATH"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret          string        `envconfig:"JWT_SECRET" required:"true"`
	Issuer          string        `envconfig:"JWT_ISSUER" default:"voicecustomer"`
	Audience        string        `envconfig:"JWT_AUDIENCE" default:"tenant-management"`
	AccessTokenTTL  time.Duration `envconfig:"JWT_ACCESS_TOKEN_TTL" default:"15m"`
	RefreshTokenTTL time.Duration `envconfig:"JWT_REFRESH_TOKEN_TTL" default:"7d"`
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled bool   `envconfig:"METRICS_ENABLED" default:"true"`
	Port    int    `envconfig:"METRICS_PORT" default:"9091"`
	Path    string `envconfig:"METRICS_PATH" default:"/metrics"`
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}

	// Build DATABASE_URL if not provided
	if cfg.Database.URL == "" {
		cfg.Database.URL = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Database,
			cfg.Database.SSLMode,
		)
	}

	return &cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}
