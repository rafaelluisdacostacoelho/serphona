// Package config provides configuration management for all services.
package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service.
type Config struct {
	// Server
	HTTPAddr string `mapstructure:"HTTP_ADDR"`
	GRPCAddr string `mapstructure:"GRPC_ADDR"`

	// Database
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	// Redis
	RedisURL string `mapstructure:"REDIS_URL"`

	// Kafka
	KafkaBrokers []string `mapstructure:"KAFKA_BROKERS"`

	// ClickHouse
	ClickHouseHost string `mapstructure:"CLICKHOUSE_HOST"`
	ClickHousePort int    `mapstructure:"CLICKHOUSE_PORT"`

	// MinIO
	MinIOEndpoint  string `mapstructure:"MINIO_ENDPOINT"`
	MinIOAccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	MinIOSecretKey string `mapstructure:"MINIO_SECRET_KEY"`

	// JWT
	JWTSecret     string `mapstructure:"JWT_SECRET"`
	JWTExpiration int    `mapstructure:"JWT_EXPIRATION"`

	// Observability
	OTLPEndpoint string `mapstructure:"OTLP_ENDPOINT"`
	LogLevel     string `mapstructure:"LOG_LEVEL"`

	// Environment
	Environment string `mapstructure:"ENVIRONMENT"`
}

// Load loads configuration from environment variables and config files.
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/serphona/")

	// Environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	viper.SetDefault("HTTP_ADDR", ":8080")
	viper.SetDefault("GRPC_ADDR", ":9090")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("JWT_EXPIRATION", 3600)
	viper.SetDefault("CLICKHOUSE_PORT", 8123)

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
