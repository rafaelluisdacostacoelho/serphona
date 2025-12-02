package config

import (
	"os"
	"strconv"
	"time"
)

// Config representa a configuração de observabilidade
type Config struct {
	// Service Info
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Tracing
	TracingEnabled  bool
	TracingEndpoint string
	TracingSampler  float64 // 0.0 to 1.0

	// Metrics
	MetricsEnabled bool
	MetricsPort    int
	MetricsPath    string

	// Logging
	LoggingEnabled bool
	LogLevel       string
	LokiEndpoint   string

	// ClickHouse
	ClickHouseEnabled  bool
	ClickHouseEndpoint string
	ClickHouseDatabase string
	ClickHouseUser     string
	ClickHousePassword string
	BatchSize          int
	FlushInterval      time.Duration

	// Features
	ConversationTracking bool
	ComplianceChecking   bool
	SentimentAnalysis    bool
}

// LoadFromEnv carrega configuração das variáveis de ambiente
func LoadFromEnv() *Config {
	return &Config{
		// Service Info
		ServiceName:    getEnv("SERVICE_NAME", "unknown"),
		ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("ENVIRONMENT", "development"),

		// Tracing
		TracingEnabled:  getEnvBool("TRACING_ENABLED", true),
		TracingEndpoint: getEnv("TRACING_ENDPOINT", "http://tempo:4317"),
		TracingSampler:  getEnvFloat("TRACING_SAMPLER", 1.0),

		// Metrics
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
		MetricsPort:    getEnvInt("METRICS_PORT", 9090),
		MetricsPath:    getEnv("METRICS_PATH", "/metrics"),

		// Logging
		LoggingEnabled: getEnvBool("LOGGING_ENABLED", true),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LokiEndpoint:   getEnv("LOKI_ENDPOINT", "http://loki:3100"),

		// ClickHouse
		ClickHouseEnabled:  getEnvBool("CLICKHOUSE_ENABLED", true),
		ClickHouseEndpoint: getEnv("CLICKHOUSE_ENDPOINT", "http://clickhouse:8123"),
		ClickHouseDatabase: getEnv("CLICKHOUSE_DATABASE", "analytics"),
		ClickHouseUser:     getEnv("CLICKHOUSE_USER", "default"),
		ClickHousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),
		BatchSize:          getEnvInt("BATCH_SIZE", 1000),
		FlushInterval:      getEnvDuration("FLUSH_INTERVAL", 10*time.Second),

		// Features
		ConversationTracking: getEnvBool("CONVERSATION_TRACKING", true),
		ComplianceChecking:   getEnvBool("COMPLIANCE_CHECKING", true),
		SentimentAnalysis:    getEnvBool("SENTIMENT_ANALYSIS", true),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
