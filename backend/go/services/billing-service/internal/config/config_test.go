package config

import "testing"

func TestValidate_Succeeds(t *testing.T) {
	cfg := &Config{
		Server:   ServerConfig{AllowedOrigins: []string{"http://localhost"}, MaxBodyBytes: 1024, ReadTimeoutMs: 1000, WriteTimeoutMs: 1000},
		Database: DatabaseConfig{URL: "postgres://user:pass@host/db"},
		Redis:    RedisConfig{URL: "redis://localhost:6379/1"},
		Kafka:    KafkaConfig{Brokers: []string{"localhost:9092"}},
		Stripe:   StripeConfig{SecretKey: "sk_test", WebhookSecret: "whsec_test"},
		JWT:      JWTConfig{Secret: "12345678901234567890123456789012"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validate success, got %v", err)
	}
}

func TestValidate_FailsOnMissing(t *testing.T) {
	cfg := &Config{
		Server:   ServerConfig{AllowedOrigins: []string{}, MaxBodyBytes: 0, ReadTimeoutMs: 0, WriteTimeoutMs: 0},
		Database: DatabaseConfig{URL: ""},
		Redis:    RedisConfig{URL: ""},
		Kafka:    KafkaConfig{Brokers: []string{""}},
		Stripe:   StripeConfig{SecretKey: "", WebhookSecret: ""},
		JWT:      JWTConfig{Secret: "short"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}
