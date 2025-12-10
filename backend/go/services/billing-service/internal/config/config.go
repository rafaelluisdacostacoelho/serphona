package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Stripe        StripeConfig
	Wallet        WalletConfig
	JWT           JWTConfig
	Features      FeaturesConfig
	Observability ObservabilityConfig
}

type ServerConfig struct {
	Host string
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  string
	URL          string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
	Topics  []string
}

type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	APIVersion     string
}

type WalletConfig struct {
	DefaultCurrency string
	InitialCredits  int64
	MinTopupAmount  int64
	MaxTopupAmount  int64
}

type JWTConfig struct {
	Secret   string
	Issuer   string
	Audience string
}

type FeaturesConfig struct {
	EnableCreditWallet   bool
	EnableSubscriptions  bool
	EnableInvoicing      bool
	EnablePaymentMethods bool
}

type ObservabilityConfig struct {
	EnableMetrics  bool
	MetricsPort    string
	EnableTracing  bool
	JaegerEndpoint string
	LogLevel       string
	LogFormat      string
}

func Load() (*Config, error) {
	// Try to load .env file (ignore error if not found)
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8081"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnv("DB_PORT", "5432"),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", "postgres"),
			Name:         getEnv("DB_NAME", "serphona_billing"),
			SSLMode:      getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  getEnv("DB_CONN_MAX_LIFETIME", "5m"),
			URL:          getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 1),
			URL:      getEnv("REDIS_URL", ""),
		},
		Kafka: KafkaConfig{
			Brokers: getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID: getEnv("KAFKA_GROUP_ID", "billing-service"),
			Topics:  getEnvAsSlice("KAFKA_TOPICS", []string{"billing.events", "payments.webhooks", "subscriptions.events"}),
		},
		Stripe: StripeConfig{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			APIVersion:     getEnv("STRIPE_API_VERSION", "2023-10-16"),
		},
		Wallet: WalletConfig{
			DefaultCurrency: getEnv("WALLET_DEFAULT_CURRENCY", "BRL"),
			InitialCredits:  getEnvAsInt64("WALLET_INITIAL_CREDITS", 100),
			MinTopupAmount:  getEnvAsInt64("WALLET_MIN_TOPUP_AMOUNT", 1000),
			MaxTopupAmount:  getEnvAsInt64("WALLET_MAX_TOPUP_AMOUNT", 1000000),
		},
		JWT: JWTConfig{
			Secret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production-min-32-chars"),
			Issuer:   getEnv("JWT_ISSUER", "serphona-auth"),
			Audience: getEnv("JWT_AUDIENCE", "serphona-api"),
		},
		Features: FeaturesConfig{
			EnableCreditWallet:   getEnvAsBool("ENABLE_CREDIT_WALLET", true),
			EnableSubscriptions:  getEnvAsBool("ENABLE_SUBSCRIPTIONS", true),
			EnableInvoicing:      getEnvAsBool("ENABLE_INVOICING", false),
			EnablePaymentMethods: getEnvAsBool("ENABLE_PAYMENT_METHODS", true),
		},
		Observability: ObservabilityConfig{
			EnableMetrics:  getEnvAsBool("ENABLE_METRICS", true),
			MetricsPort:    getEnv("METRICS_PORT", "9091"),
			EnableTracing:  getEnvAsBool("ENABLE_TRACING", false),
			JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
			LogLevel:       getEnv("LOG_LEVEL", "info"),
			LogFormat:      getEnv("LOG_FORMAT", "json"),
		},
	}

	// Build database URL if not provided
	if config.Database.URL == "" {
		config.Database.URL = fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
			config.Database.User,
			config.Database.Password,
			config.Database.Host,
			config.Database.Port,
			config.Database.Name,
			config.Database.SSLMode,
		)
	}

	// Build Redis URL if not provided
	if config.Redis.URL == "" {
		if config.Redis.Password != "" {
			config.Redis.URL = fmt.Sprintf("redis://:%s@%s:%s/%d",
				config.Redis.Password,
				config.Redis.Host,
				config.Redis.Port,
				config.Redis.DB,
			)
		} else {
			config.Redis.URL = fmt.Sprintf("redis://%s:%s/%d",
				config.Redis.Host,
				config.Redis.Port,
				config.Redis.DB,
			)
		}
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.Stripe.SecretKey == "" {
		log.Println("Warning: STRIPE_SECRET_KEY is not set")
	}

	if c.Stripe.WebhookSecret == "" {
		log.Println("Warning: STRIPE_WEBHOOK_SECRET is not set")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
