// Package config provides configuration for the voice-gateway service.
package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the application configuration.
type Config struct {
	ServiceName string `envconfig:"SERVICE_NAME" default:"voice-gateway"`
	Version     string `envconfig:"VERSION" default:"1.0.0"`
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`

	Server            ServerConfig
	Asterisk          AsteriskConfig
	Redis             RedisConfig
	Kafka             KafkaConfig
	TenantManager     TenantManagerConfig
	AgentOrchestrator AgentOrchestratorConfig
	Audio             AudioConfig
	Call              CallConfig
	Metrics           MetricsConfig
	HealthCheck       HealthCheckConfig
	FeatureFlags      FeatureFlagsConfig
}

// ServerConfig represents server configuration.
type ServerConfig struct {
	Host            string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port            int           `envconfig:"SERVER_PORT" default:"8080"`
	GRPCPort        int           `envconfig:"SERVER_GRPC_PORT" default:"9090"`
	ReadTimeout     time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout     time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
}

// AsteriskConfig represents Asterisk connection configuration.
type AsteriskConfig struct {
	// ARI Configuration
	ARIURL      string `envconfig:"ASTERISK_ARI_URL" required:"true"`
	ARIUsername string `envconfig:"ASTERISK_ARI_USERNAME" required:"true"`
	ARIPassword string `envconfig:"ASTERISK_ARI_PASSWORD" required:"true"`
	ARIAppName  string `envconfig:"ASTERISK_ARI_APP_NAME" default:"serphona"`

	// AMI Configuration (optional, for fallback)
	AMIHost     string `envconfig:"ASTERISK_AMI_HOST"`
	AMIPort     int    `envconfig:"ASTERISK_AMI_PORT" default:"5038"`
	AMIUsername string `envconfig:"ASTERISK_AMI_USERNAME"`
	AMIPassword string `envconfig:"ASTERISK_AMI_PASSWORD"`
}

// RedisConfig represents Redis configuration.
type RedisConfig struct {
	URL          string        `envconfig:"REDIS_URL" default:"redis://localhost:6379"`
	Password     string        `envconfig:"REDIS_PASSWORD"`
	DB           int           `envconfig:"REDIS_DB" default:"0"`
	CallStateTTL time.Duration `envconfig:"REDIS_CALL_STATE_TTL" default:"1h"`
}

// KafkaConfig represents Kafka configuration.
type KafkaConfig struct {
	Brokers           []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	TopicPrefix       string   `envconfig:"KAFKA_TOPIC_PREFIX" default:"serphona"`
	GroupID           string   `envconfig:"KAFKA_GROUP_ID" default:"voice-gateway"`
	EnableIdempotence bool     `envconfig:"KAFKA_ENABLE_IDEMPOTENCE" default:"true"`
}

// TenantManagerConfig represents tenant-manager client configuration.
type TenantManagerConfig struct {
	URL     string        `envconfig:"TENANT_MANAGER_URL" required:"true"`
	Timeout time.Duration `envconfig:"TENANT_MANAGER_TIMEOUT" default:"10s"`
}

// AgentOrchestratorConfig represents agent-orchestrator client configuration.
type AgentOrchestratorConfig struct {
	URL     string        `envconfig:"AGENT_ORCHESTRATOR_URL" required:"true"`
	Timeout time.Duration `envconfig:"AGENT_ORCHESTRATOR_TIMEOUT" default:"30s"`
}

// AudioConfig represents audio processing configuration.
type AudioConfig struct {
	SampleRate int    `envconfig:"AUDIO_SAMPLE_RATE" default:"16000"`
	Channels   int    `envconfig:"AUDIO_CHANNELS" default:"1"`
	Format     string `envconfig:"AUDIO_FORMAT" default:"pcm"`
	BufferSize int    `envconfig:"AUDIO_BUFFER_SIZE" default:"8192"`
}

// CallConfig represents call handling configuration.
type CallConfig struct {
	MaxConcurrentCalls   int           `envconfig:"MAX_CONCURRENT_CALLS" default:"1000"`
	CallTimeout          time.Duration `envconfig:"CALL_TIMEOUT" default:"30m"`
	SilenceTimeout       time.Duration `envconfig:"SILENCE_TIMEOUT" default:"5s"`
	MaxConversationTurns int           `envconfig:"MAX_CONVERSATION_TURNS" default:"100"`
}

// MetricsConfig represents metrics configuration.
type MetricsConfig struct {
	Port int    `envconfig:"METRICS_PORT" default:"9091"`
	Path string `envconfig:"METRICS_PATH" default:"/metrics"`
}

// HealthCheckConfig represents health check configuration.
type HealthCheckConfig struct {
	Interval time.Duration `envconfig:"HEALTH_CHECK_INTERVAL" default:"30s"`
}

// FeatureFlagsConfig represents feature flags.
type FeatureFlagsConfig struct {
	EnableCallRecording        bool `envconfig:"ENABLE_CALL_RECORDING" default:"true"`
	EnableTranscriptionStorage bool `envconfig:"ENABLE_TRANSCRIPTION_STORAGE" default:"true"`
	EnableAudioStreaming       bool `envconfig:"ENABLE_AUDIO_STREAMING" default:"true"`
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
