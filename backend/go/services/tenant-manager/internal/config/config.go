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
	GRPC     GRPCConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
	Metrics  MetricsConfig
	Tracing  TracingConfig
}

// ServerConfig represents server configuration.
type ServerConfig struct {
	Host           string `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port           int    `envconfig:"SERVER_PORT" default:"8080"`
	GRPCPort       int    `envconfig:"SERVER_GRPC_PORT" default:"9090"`
	BodyLimitBytes int64  `envconfig:"SERVER_BODY_LIMIT_BYTES" default:"1048576"`

	ReadTimeout       time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
	ReadHeaderTimeout time.Duration `envconfig:"SERVER_READ_HEADER_TIMEOUT" default:"5s"`
	WriteTimeout      time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout       time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"120s"`

	MaxHeaderBytes     int           `envconfig:"SERVER_MAX_HEADER_BYTES" default:"1048576"` // 1MB
	ShutdownTimeout    time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
	RateLimitRPM       int           `envconfig:"SERVER_RATE_LIMIT_RPM" default:"600"`
	CORSAllowedOrigins []string      `envconfig:"CORS_ALLOWED_ORIGINS" default:"*"`
}

type GRPCConfig struct {
	Host string `envconfig:"GRPC_HOST" default:"0.0.0.0"`
	Port int    `envconfig:"GRPC_PORT" default:"9090"` // você pode manter alinhado com SERVER_GRPC_PORT por enquanto

	// Handshake: tempo para estabelecer conexão (não é timeout de RPC)
	ConnectionTimeout time.Duration `envconfig:"GRPC_CONNECTION_TIMEOUT" default:"5s"`

	// Limites de payload: evita OOM e abuse
	MaxRecvMsgSizeMB int `envconfig:"GRPC_MAX_RECV_MSG_SIZE_MB" default:"10"`
	MaxSendMsgSizeMB int `envconfig:"GRPC_MAX_SEND_MSG_SIZE_MB" default:"10"`

	// Produção: reflection deve ser false por padrão
	ReflectionEnabled bool `envconfig:"GRPC_REFLECTION_ENABLED" default:"false"`

	// Deadline policy (escala): evita chamadas sem deadline ficarem presas
	DefaultRequestTimeout time.Duration `envconfig:"GRPC_DEFAULT_REQUEST_TIMEOUT" default:"10s"`
	RequireClientDeadline bool          `envconfig:"GRPC_REQUIRE_CLIENT_DEADLINE" default:"false"`

	// Keepalive: estabilidade sob LB/proxies e conexões zumbis
	Keepalive GRPCKeepaliveConfig
}

type GRPCKeepaliveConfig struct {
	// ServerParameters
	MaxConnectionIdle     time.Duration `envconfig:"GRPC_KA_MAX_CONNECTION_IDLE" default:"5m"`
	MaxConnectionAge      time.Duration `envconfig:"GRPC_KA_MAX_CONNECTION_AGE" default:"2h"`
	MaxConnectionAgeGrace time.Duration `envconfig:"GRPC_KA_MAX_CONNECTION_AGE_GRACE" default:"5m"`
	Time                  time.Duration `envconfig:"GRPC_KA_TIME" default:"2h"`
	Timeout               time.Duration `envconfig:"GRPC_KA_TIMEOUT" default:"20s"`

	// EnforcementPolicy
	MinTime             time.Duration `envconfig:"GRPC_KA_MIN_TIME" default:"1m"`
	PermitWithoutStream bool          `envconfig:"GRPC_KA_PERMIT_WITHOUT_STREAM" default:"true"`
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
	Brokers      []string      `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	TopicPrefix  string        `envconfig:"KAFKA_TOPIC_PREFIX" default:"serphona"`
	GroupID      string        `envconfig:"KAFKA_GROUP_ID" default:"tenant-manager"`
	RetryMax     int           `envconfig:"KAFKA_RETRY_MAX" default:"3"`
	RetryBackoff time.Duration `envconfig:"KAFKA_RETRY_BACKOFF" default:"500ms"`
	DLQTopic     string        `envconfig:"KAFKA_DLQ_TOPIC"`
}

// JWTConfig represents JWT configuration.
type JWTConfig struct {
	Secret    string   `envconfig:"JWT_SECRET" required:"true"`
	PublicKey string   `envconfig:"JWT_PUBLIC_KEY"`                // PEM encoded RSA public key for RS256 (optional)
	Issuer    string   `envconfig:"JWT_ISSUER" default:"serphona"` // Expected issuer (optional)
	Audience  []string `envconfig:"JWT_AUDIENCE"`                  // Expected audience list (optional)
}

// MetricsConfig represents metrics configuration.
type MetricsConfig struct {
	Enabled bool `envconfig:"METRICS_ENABLED" default:"true"`
	Port    int  `envconfig:"METRICS_PORT" default:"9091"`
}

// TracingConfig represents tracing configuration.
type TracingConfig struct {
	Enabled        bool   `envconfig:"TRACING_ENABLED" default:"false"`
	JaegerEndpoint string `envconfig:"JAEGER_ENDPOINT"`
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
