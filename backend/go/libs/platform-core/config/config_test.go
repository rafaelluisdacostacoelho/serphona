package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("GRPC_ADDR", "")
	t.Setenv("JWT_EXPIRATION", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default HTTP_ADDR :8080, got %s", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != ":9090" {
		t.Fatalf("expected default GRPC_ADDR :9090, got %s", cfg.GRPCAddr)
	}
	if cfg.JWTExpiration != 3600 {
		t.Fatalf("expected default JWT_EXPIRATION 3600, got %d", cfg.JWTExpiration)
	}
	if cfg.ClickHousePort != 8123 {
		t.Fatalf("expected default CLICKHOUSE_PORT 8123, got %d", cfg.ClickHousePort)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("GRPC_ADDR", ":19000")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("KAFKA_BROKERS", "b1:9092,b2:9092")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_EXPIRATION", "7200")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.HTTPAddr != ":9000" || cfg.GRPCAddr != ":19000" {
		t.Fatalf("env overrides not applied: %s / %s", cfg.HTTPAddr, cfg.GRPCAddr)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "b1:9092" || cfg.KafkaBrokers[1] != "b2:9092" {
		t.Fatalf("unexpected kafka brokers: %+v", cfg.KafkaBrokers)
	}
	if cfg.JWTExpiration != 7200 {
		t.Fatalf("expected JWT_EXPIRATION 7200, got %d", cfg.JWTExpiration)
	}
	if cfg.JWTSecret != "secret" {
		t.Fatalf("expected JWT_SECRET to load from env")
	}
}

func TestValidateRequiredMissing(t *testing.T) {
	cfg := &Config{
		HTTPAddr: ":8080",
	}

	err := ValidateRequired(cfg, "DATABASE_URL", "JWT_SECRET")
	if err == nil {
		t.Fatalf("expected missing required config error")
	}
}

func TestValidateRequiredOK(t *testing.T) {
	cfg := &Config{
		HTTPAddr:    ":8080",
		DatabaseURL: "postgres://user:pass@localhost:5432/db",
		JWTSecret:   "secret",
	}

	if err := ValidateRequired(cfg, "DATABASE_URL", "JWT_SECRET"); err != nil {
		t.Fatalf("expected validation to pass, got: %v", err)
	}
}
