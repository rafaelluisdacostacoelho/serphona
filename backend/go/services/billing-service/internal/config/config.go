package config

import (
	"encoding/json"
	"fmt"
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
	Pricing       PricingConfig
	JWT           JWTConfig
	Features      FeaturesConfig
	Observability ObservabilityConfig
	Service       ServiceConfig
}

type ServerConfig struct {
	Host           string
	Port           string
	Env            string
	ReadTimeoutMs  int
	WriteTimeoutMs int
	MaxBodyBytes   int64
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string
}

type KafkaConfig struct {
	Brokers        []string
	GroupID        string
	Topics         []string
	DLQTopic       string
	MaxRetries     int
	RetryBackoffMs int
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

type PricingConfig struct {
	DefaultPlan string
	Plans       map[string]PlanPricing
}

type PlanPricing struct {
	CallCents       int64 `json:"call_cents"`
	MinuteCents     int64 `json:"minute_cents"`
	MessageCents    int64 `json:"message_cents"`
	APIRequestCents int64 `json:"api_request_cents"`
	StorageGBCents  int64 `json:"storage_gb_cents"`
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

type ServiceConfig struct {
	Name      string
	Instance  string
	AuthToken string
	Audience  string
}

func Load() (*Config, error) {
	// Try to load .env file (ignore error if not found)
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Host:           getEnv("SERVER_HOST", "0.0.0.0"),
			Port:           getEnv("SERVER_PORT", "8081"),
			Env:            getEnv("ENV", "development"),
			ReadTimeoutMs:  getEnvAsInt("SERVER_READ_TIMEOUT_MS", 30000),
			WriteTimeoutMs: getEnvAsInt("SERVER_WRITE_TIMEOUT_MS", 30000),
			MaxBodyBytes:   getEnvAsInt64("SERVER_MAX_BODY_BYTES", 2*1024*1024),
			AllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{}),
			AllowedMethods: getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}),
			AllowedHeaders: getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization"}),
		},
		Database: DatabaseConfig{
			URL:          getEnv("DATABASE_URL", ""),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  getEnv("DB_CONN_MAX_LIFETIME", "5m"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 1),
			URL:      getEnv("REDIS_URL", ""),
		},
		Kafka: KafkaConfig{
			Brokers:        getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID:        getEnv("KAFKA_GROUP_ID", "billing-service"),
			Topics:         getEnvAsSlice("KAFKA_TOPICS", []string{"billing.events", "payments.webhooks", "subscriptions.events"}),
			DLQTopic:       getEnv("KAFKA_DLQ_TOPIC", "usage.reported.dlq"),
			MaxRetries:     getEnvAsInt("KAFKA_MAX_RETRIES", 3),
			RetryBackoffMs: getEnvAsInt("KAFKA_RETRY_BACKOFF_MS", 2000),
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
		Pricing: loadPricingConfig(),
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
		Service: ServiceConfig{
			Name:      getEnv("SERVICE_NAME", "billing-service"),
			Instance:  getEnv("SERVICE_INSTANCE", "billing-service-1"),
			AuthToken: getEnv("SERVICE_AUTH_TOKEN", ""),
			Audience:  getEnv("SERVICE_AUDIENCE", ""),
		},
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
	var missing []string

	if c.Stripe.SecretKey == "" {
		missing = append(missing, "STRIPE_SECRET_KEY")
	}
	if c.Stripe.WebhookSecret == "" {
		missing = append(missing, "STRIPE_WEBHOOK_SECRET")
	}
	if len(c.JWT.Secret) < 32 {
		missing = append(missing, "JWT_SECRET (>=32 chars)")
	}
	if c.Database.URL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.Redis.URL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if len(c.Kafka.Brokers) == 0 || strings.TrimSpace(c.Kafka.Brokers[0]) == "" {
		missing = append(missing, "KAFKA_BROKERS")
	}
	if len(c.Server.AllowedOrigins) == 0 {
		missing = append(missing, "CORS_ALLOWED_ORIGINS")
	}
	if c.Server.MaxBodyBytes <= 0 {
		missing = append(missing, "SERVER_MAX_BODY_BYTES (>0)")
	}
	if c.Server.ReadTimeoutMs <= 0 {
		missing = append(missing, "SERVER_READ_TIMEOUT_MS (>0)")
	}
	if c.Server.WriteTimeoutMs <= 0 {
		missing = append(missing, "SERVER_WRITE_TIMEOUT_MS (>0)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}

func loadPricingConfig() PricingConfig {
	defaultPlans := map[string]PlanPricing{
		"starter": {
			CallCents:       5,
			MinuteCents:     10,
			MessageCents:    2,
			APIRequestCents: 1,
			StorageGBCents:  15,
		},
		"professional": {
			CallCents:       4,
			MinuteCents:     8,
			MessageCents:    2,
			APIRequestCents: 1,
			StorageGBCents:  12,
		},
		"enterprise": {
			CallCents:       3,
			MinuteCents:     6,
			MessageCents:    2,
			APIRequestCents: 1,
			StorageGBCents:  10,
		},
	}

	cfg := PricingConfig{
		DefaultPlan: getEnv("PRICING_DEFAULT_PLAN", "starter"),
		Plans:       defaultPlans,
	}

	overrides := getEnv("PRICING_PLANS_JSON", "")
	if overrides != "" {
		var m map[string]PlanPricing
		if err := json.Unmarshal([]byte(overrides), &m); err == nil {
			for k, v := range m {
				cfg.Plans[strings.ToLower(k)] = v
			}
		}
	}

	if _, ok := cfg.Plans[cfg.DefaultPlan]; !ok {
		cfg.DefaultPlan = "starter"
	}

	return cfg
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
