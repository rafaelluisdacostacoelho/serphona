package config

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.EnableAutoCommit {
		t.Fatalf("expected auto-commit disabled by default")
	}
	if cfg.CommitInterval != time.Second || cfg.SessionTimeout != 10*time.Second {
		t.Fatalf("unexpected commit/session defaults: %+v", cfg)
	}
	if cfg.PublisherBatchSize != 100 || cfg.PublisherBatchTimeout != 100*time.Millisecond {
		t.Fatalf("unexpected publisher defaults: %+v", cfg)
	}
	if cfg.ConsumerConcurrency != 5 || cfg.ConsumerMaxRetries != 3 {
		t.Fatalf("unexpected consumer defaults: %+v", cfg)
	}
	if cfg.ServiceName != "unknown" || cfg.Environment != "development" || cfg.Debug {
		t.Fatalf("unexpected service/env defaults: %+v", cfg)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "b1:9092,b2:9092")
	t.Setenv("KAFKA_GROUP_ID", "group")
	t.Setenv("KAFKA_CLIENT_ID", "client")
	t.Setenv("SERVICE_NAME", "svc")
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("DEBUG", "true")
	t.Setenv("KAFKA_AUTO_COMMIT", "1")
	t.Setenv("KAFKA_COMMIT_INTERVAL", "2s")
	t.Setenv("KAFKA_SESSION_TIMEOUT", "3s")
	t.Setenv("KAFKA_PUBLISHER_BATCH_SIZE", "10")
	t.Setenv("KAFKA_PUBLISHER_BATCH_TIMEOUT", "50ms")
	t.Setenv("KAFKA_PUBLISHER_MAX_RETRIES", "7")
	t.Setenv("KAFKA_PUBLISHER_RETRY_INTERVAL", "150ms")
	t.Setenv("KAFKA_CONSUMER_MAX_RETRIES", "4")
	t.Setenv("KAFKA_CONSUMER_RETRY_INTERVAL", "90ms")
	t.Setenv("KAFKA_CONSUMER_CONCURRENCY", "2")
	t.Setenv("KAFKA_TLS", "true")
	t.Setenv("KAFKA_TLS_INSECURE_SKIP_VERIFY", "true")
	t.Setenv("KAFKA_SASL_MECHANISM", "scram-sha256")
	t.Setenv("KAFKA_SASL_USERNAME", "user")
	t.Setenv("KAFKA_SASL_PASSWORD", "pass")

	cfg := LoadFromEnv()

	if got := strings.Join(cfg.Brokers, ","); got != "b1:9092,b2:9092" {
		t.Fatalf("brokers not loaded: %s", got)
	}
	if cfg.GroupID != "group" || cfg.ClientID != "client" || cfg.ServiceName != "svc" || cfg.Environment != "prod" {
		t.Fatalf("ids/service/env not loaded: %+v", cfg)
	}
	if !cfg.Debug || !cfg.EnableAutoCommit || cfg.CommitInterval != 2*time.Second || cfg.SessionTimeout != 3*time.Second {
		t.Fatalf("debug/commit/session not loaded: %+v", cfg)
	}
	if cfg.PublisherBatchSize != 10 || cfg.PublisherBatchTimeout != 50*time.Millisecond {
		t.Fatalf("publisher batch not loaded: %+v", cfg)
	}
	if cfg.PublisherMaxRetries != 7 || cfg.PublisherRetryInterval != 150*time.Millisecond {
		t.Fatalf("publisher retry not loaded: %+v", cfg)
	}
	if cfg.ConsumerMaxRetries != 4 || cfg.ConsumerRetryInterval != 90*time.Millisecond || cfg.ConsumerConcurrency != 2 {
		t.Fatalf("consumer settings not loaded: %+v", cfg)
	}
	if !cfg.UseTLS || !cfg.TLSInsecureSkipVerify {
		t.Fatalf("tls flags not loaded: %+v", cfg)
	}
	if cfg.SASLMechanism != "scram-sha256" || cfg.SASLUsername != "user" || cfg.SASLPassword != "pass" {
		t.Fatalf("sasl settings not loaded: %+v", cfg)
	}
}

func TestLoadFromEnvServiceNameOverridesDefaultClient(t *testing.T) {
	t.Setenv("SERVICE_NAME", "svc-override")

	cfg := LoadFromEnv()
	if cfg.ClientID != "svc-override" {
		t.Fatalf("service name should override default client id, got %s", cfg.ClientID)
	}
}

func TestParseBool(t *testing.T) {
	truths := []string{"true", "True", "1", "yes", "YES"}
	for _, val := range truths {
		if !parseBool(val) {
			t.Fatalf("expected true for %s", val)
		}
	}
	if parseBool("false") {
		t.Fatalf("expected false for 'false'")
	}
}

func TestValidateAndErrors(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Config)
		want   error
	}{
		{"no brokers", func(c *Config) { c.Brokers = nil }, ErrNoBrokers},
		{"no group", func(c *Config) { c.GroupID = "" }, ErrNoGroupID},
		{"no client", func(c *Config) { c.ClientID = "" }, ErrNoClientID},
		{"no service", func(c *Config) { c.ServiceName = "" }, ErrNoServiceName},
		{"unsupported sasl", func(c *Config) { c.SASLMechanism = "kerberos" }, ErrUnsupportedSASL},
		{"missing sasl creds", func(c *Config) { c.SASLMechanism = "plain"; c.SASLUsername = ""; c.SASLPassword = "" }, ErrMissingSASLCreds},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpy := *cfg
			tc.mutate(&cpy)
			err := cpy.Validate()
			if err == nil || err != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
			if msg := err.Error(); !strings.HasPrefix(msg, "config error: ") {
				t.Fatalf("unexpected error prefix: %s", msg)
			}
		})
	}
}
