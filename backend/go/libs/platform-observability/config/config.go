package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config representa a configuração de observabilidade
type Config struct {
	// Service Info
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Tracing
	TracingEnabled         bool
	TracingEndpoint        string
	TracingSampler         float64 // 0.0 to 1.0
	TracingSamplerStrategy string   // ratio|parent_ratio|always_on|always_off|adaptive
	TracingInsecure        bool
	TracingTLSInsecure     bool
	TracingTLSCACertPath   string
	TracingTLSClientCert   string
	TracingTLSClientKey    string
	TracingBearerToken     string

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

	// Kafka exporter
	KafkaEnabled      bool
	KafkaBrokers      []string
	KafkaTopic        string
	KafkaClientID     string
	KafkaSASLUser     string
	KafkaSASLPass     string
	KafkaSASLMechanism string
	KafkaTLSEnabled   bool
	KafkaTLSInsecure  bool

	// Logging/Loki
	LokiEnabled bool
	LokiTenant  string

	// Features
	ConversationTracking bool
	ComplianceChecking   bool
	SentimentAnalysis    bool

	// Anomaly detection
	AnomalyDetectionEnabled bool
	AnomalyWindowMinutes    int
	AnomalyBucketMinutes    int
	AnomalyZScoreThreshold  float64
	AnomalyMinCount         int
	AnomalyCooldownSeconds  int

	// ML alerts
	MLAlertsEnabled bool
}

// LoadFromEnv carrega configuração das variáveis de ambiente
func LoadFromEnv() *Config {
	return &Config{
		// Service Info
		ServiceName:    getEnv("SERVICE_NAME", "unknown"),
		ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("ENVIRONMENT", "development"),

		// Tracing
			TracingEnabled:         getEnvBool("TRACING_ENABLED", true),
			TracingEndpoint:        getEnv("TRACING_ENDPOINT", "tempo:4317"),
			TracingSampler:         getEnvFloat("TRACING_SAMPLER", 1.0),
			TracingSamplerStrategy: getEnv("TRACING_SAMPLER_STRATEGY", "parent_ratio"),
			TracingInsecure:        getEnvBool("TRACING_INSECURE", true),
			TracingTLSInsecure:     getEnvBool("TRACING_TLS_INSECURE", false),
			TracingTLSCACertPath:   getEnv("TRACING_TLS_CA_CERT", ""),
			TracingTLSClientCert:   getEnv("TRACING_TLS_CLIENT_CERT", ""),
			TracingTLSClientKey:    getEnv("TRACING_TLS_CLIENT_KEY", ""),
			TracingBearerToken:     getEnv("TRACING_BEARER_TOKEN", ""),

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

		// Kafka exporter
		KafkaEnabled:       getEnvBool("KAFKA_ENABLED", false),
		KafkaBrokers:       splitCSV(getEnv("KAFKA_BROKERS", "")),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "observability.events"),
		KafkaClientID:      getEnv("KAFKA_CLIENT_ID", "platform-observability"),
		KafkaSASLUser:      getEnv("KAFKA_SASL_USERNAME", ""),
		KafkaSASLPass:      getEnv("KAFKA_SASL_PASSWORD", ""),
		KafkaSASLMechanism: getEnv("KAFKA_SASL_MECHANISM", ""),
		KafkaTLSEnabled:    getEnvBool("KAFKA_TLS_ENABLED", false),
		KafkaTLSInsecure:   getEnvBool("KAFKA_TLS_INSECURE", false),

		// Logging/Loki
		LokiEnabled: getEnvBool("LOKI_ENABLED", false),
		LokiTenant:  getEnv("LOKI_TENANT", ""),

		// Features
		ConversationTracking: getEnvBool("CONVERSATION_TRACKING", true),
		ComplianceChecking:   getEnvBool("COMPLIANCE_CHECKING", true),
		SentimentAnalysis:    getEnvBool("SENTIMENT_ANALYSIS", true),

		// Anomaly detection
		AnomalyDetectionEnabled: getEnvBool("ANOMALY_DETECTION_ENABLED", false),
		AnomalyWindowMinutes:    getEnvInt("ANOMALY_WINDOW_MINUTES", 5),
		AnomalyBucketMinutes:    getEnvInt("ANOMALY_BUCKET_MINUTES", 1),
		AnomalyZScoreThreshold:  getEnvFloat("ANOMALY_ZSCORE_THRESHOLD", 3.0),
		AnomalyMinCount:         getEnvInt("ANOMALY_MIN_COUNT", 5),
		AnomalyCooldownSeconds:  getEnvInt("ANOMALY_COOLDOWN_SECONDS", 300),

		// ML alerts
		MLAlertsEnabled: getEnvBool("ML_ALERTS_ENABLED", false),
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

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var out []string
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
