package config

import "testing"

func TestLoadBuildsConfigFromEnv(t *testing.T) {
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_MAX_BODY_BYTES", "4096")
	t.Setenv("SERVER_READ_TIMEOUT_MS", "2000")
	t.Setenv("SERVER_WRITE_TIMEOUT_MS", "3000")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://example.com,https://another.com")
	t.Setenv("CORS_ALLOWED_METHODS", "GET,POST")
	t.Setenv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")

	t.Setenv("DATABASE_URL", "postgresql://dbuser:dbpass@db.example:5555/billingdb?sslmode=require")

	t.Setenv("REDIS_HOST", "redis.example")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "redispw")
	t.Setenv("REDIS_DB", "2")

	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("KAFKA_GROUP_ID", "billing-group")
	t.Setenv("KAFKA_TOPICS", "topic1,topic2")
	t.Setenv("KAFKA_DLQ_TOPIC", "dlq-topic")
	t.Setenv("KAFKA_MAX_RETRIES", "5")
	t.Setenv("KAFKA_RETRY_BACKOFF_MS", "1500")

	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")

	t.Setenv("JWT_SECRET", "123456789012345678901234567890123456")
	t.Setenv("JWT_ISSUER", "issuer")
	t.Setenv("JWT_AUDIENCE", "aud")

	t.Setenv("WALLET_DEFAULT_CURRENCY", "USD")
	t.Setenv("WALLET_INITIAL_CREDITS", "500")
	t.Setenv("WALLET_MIN_TOPUP_AMOUNT", "200")
	t.Setenv("WALLET_MAX_TOPUP_AMOUNT", "1000")

	t.Setenv("PRICING_DEFAULT_PLAN", "pro")
	t.Setenv("PRICING_PLANS_JSON", `{"pro":{"call_cents":2,"minute_cents":3,"message_cents":1,"api_request_cents":1,"storage_gb_cents":4}}`)

	t.Setenv("ENABLE_CREDIT_WALLET", "true")
	t.Setenv("ENABLE_SUBSCRIPTIONS", "false")
	t.Setenv("ENABLE_INVOICING", "true")
	t.Setenv("ENABLE_PAYMENT_METHODS", "false")

	t.Setenv("ENABLE_METRICS", "true")
	t.Setenv("METRICS_PORT", "9100")
	t.Setenv("ENABLE_TRACING", "true")
	t.Setenv("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "console")

	t.Setenv("SERVICE_NAME", "billing-svc")
	t.Setenv("SERVICE_INSTANCE", "billing-1")
	t.Setenv("SERVICE_AUTH_TOKEN", "token")
	t.Setenv("SERVICE_AUDIENCE", "audience")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}

	if cfg.Database.URL != "postgresql://dbuser:dbpass@db.example:5555/billingdb?sslmode=require" {
		t.Fatalf("unexpected database url %s", cfg.Database.URL)
	}
	expectedRedisURL := "redis://:redispw@redis.example:6380/2"
	if cfg.Redis.URL != expectedRedisURL {
		t.Fatalf("unexpected redis url %s", cfg.Redis.URL)
	}

	if len(cfg.Kafka.Brokers) != 2 || cfg.Kafka.Brokers[0] != "broker1:9092" || cfg.Kafka.Topics[0] != "topic1" {
		t.Fatalf("unexpected kafka config %+v", cfg.Kafka)
	}

	if cfg.Pricing.DefaultPlan != "pro" {
		t.Fatalf("expected pricing default plan override, got %s", cfg.Pricing.DefaultPlan)
	}
	plan := cfg.Pricing.Plans["pro"]
	if plan.CallCents != 2 || plan.MinuteCents != 3 || plan.MessageCents != 1 || plan.APIRequestCents != 1 || plan.StorageGBCents != 4 {
		t.Fatalf("unexpected pricing overrides %+v", plan)
	}

	if cfg.Wallet.DefaultCurrency != "USD" || cfg.Wallet.InitialCredits != 500 || cfg.Wallet.MaxTopupAmount != 1000 {
		t.Fatalf("unexpected wallet config %+v", cfg.Wallet)
	}

	if cfg.Features.EnableSubscriptions || !cfg.Features.EnableInvoicing || cfg.Features.EnablePaymentMethods {
		t.Fatalf("unexpected features config %+v", cfg.Features)
	}

	if cfg.Service.Name != "billing-svc" || cfg.Service.Instance != "billing-1" || cfg.Service.AuthToken != "token" || cfg.Service.Audience != "audience" {
		t.Fatalf("unexpected service config %+v", cfg.Service)
	}

	if len(cfg.Server.AllowedOrigins) != 2 || cfg.Server.AllowedOrigins[0] != "http://example.com" {
		t.Fatalf("unexpected server cors config %+v", cfg.Server.AllowedOrigins)
	}
}

func TestLoadPricingConfigFallbacksToStarter(t *testing.T) {
	t.Setenv("PRICING_DEFAULT_PLAN", "unknown")
	t.Setenv("PRICING_PLANS_JSON", "")

	cfg := loadPricingConfig()
	if cfg.DefaultPlan != "starter" {
		t.Fatalf("expected default plan to fall back to starter, got %s", cfg.DefaultPlan)
	}
	starter, ok := cfg.Plans["starter"]
	if !ok || starter.CallCents == 0 {
		t.Fatalf("expected starter plan to be populated, got %+v", starter)
	}
}
