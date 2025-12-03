// Package config provides configuration for the tenant-manager service.
package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the application configuration.
type Config struct {
	Version     string `envconfig:"VERSION" default:"1.0.0"`
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`

	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
	Metrics  MetricsConfig
}

// ServerConfig represents server configuration.
type ServerConfig struct {
	Host            string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port            int           `envconfig:"SERVER_PORT" default:"8080"`
	GRPCPort        int           `envconfig:"SERVER_GRPC_PORT" default:"9090"`
	ReadTimeout     time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
}

// DatabaseConfig represents database configuration.
type DatabaseConfig struct {
	URL            string        `envconfig:"DATABASE_URL" required:"true"`
	MaxOpenConns   int           `envconfig:"DATABASE_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns   int           `envconfig:"DATABASE_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLife    time.Duration `envconfig:"DATABASE_CONN_MAX_LIFE" default:"5m"`
	AutoMigrate    bool          `envconfig:"DATABASE_AUTO_MIGRATE" default:"true"`
	MigrationsPath string        `envconfig:"DATABASE_MIGRATIONS_PATH" default:"migrations"`
}

// RedisConfig represents Redis configuration.
type RedisConfig struct {
	URL      string        `envconfig:"REDIS_URL" default:"redis://localhost:6379"`
	Password string        `envconfig:"REDIS_PASSWORD"`
	DB       int           `envconfig:"REDIS_DB" default:"0"`
	CacheTTL time.Duration `envconfig:"REDIS_CACHE_TTL" default:"5m"`
}

// KafkaConfig represents Kafka configuration.
type KafkaConfig struct {
	Brokers     []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	TopicPrefix string   `envconfig:"KAFKA_TOPIC_PREFIX" default:"serphona"`
	GroupID     string   `envconfig:"KAFKA_GROUP_ID" default:"tenant-manager"`
}

// JWTConfig represents JWT configuration.
type JWTConfig struct {
	Secret string `envconfig:"JWT_SECRET" required:"true"`
	Issuer string `envconfig:"JWT_ISSUER" default:"serphona"`
}

// MetricsConfig represents metrics configuration.
type MetricsConfig struct {
	Port int `envconfig:"METRICS_PORT" default:"9091"`
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
