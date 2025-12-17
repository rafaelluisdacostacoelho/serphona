package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contém as configurações para o sistema de eventos
type Config struct {
	// Kafka configuration
	Brokers          []string
	GroupID          string
	ClientID         string
	EnableAutoCommit bool
	SessionTimeout   time.Duration
	CommitInterval   time.Duration

	// Publisher configuration
	PublisherBatchSize     int
	PublisherBatchTimeout  time.Duration
	PublisherMaxRetries    int
	PublisherRetryInterval time.Duration

	// Consumer configuration
	ConsumerMaxRetries    int
	ConsumerRetryInterval time.Duration
	ConsumerConcurrency   int

	// Security (optional)
	UseTLS                bool
	TLSInsecureSkipVerify bool
	SASLMechanism         string
	SASLUsername          string
	SASLPassword          string

	// General configuration
	ServiceName string
	Environment string
	Debug       bool
}

// DefaultConfig retorna uma configuração padrão
func DefaultConfig() *Config {
	return &Config{
		Brokers:                []string{"localhost:9092"},
		GroupID:                "serphona-default",
		ClientID:               "serphona-client",
		EnableAutoCommit:       false,
		SessionTimeout:         10 * time.Second,
		CommitInterval:         1 * time.Second,
		PublisherBatchSize:     100,
		PublisherBatchTimeout:  100 * time.Millisecond,
		PublisherMaxRetries:    3,
		PublisherRetryInterval: 1 * time.Second,
		ConsumerMaxRetries:     3,
		ConsumerRetryInterval:  1 * time.Second,
		ConsumerConcurrency:    5,
		ServiceName:            "unknown",
		Environment:            "development",
		Debug:                  false,
	}
}

// LoadFromEnv carrega configurações das variáveis de ambiente
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	// Kafka brokers
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		cfg.Brokers = strings.Split(brokers, ",")
	}

	// Group ID
	if groupID := os.Getenv("KAFKA_GROUP_ID"); groupID != "" {
		cfg.GroupID = groupID
	}

	// Client ID
	if clientID := os.Getenv("KAFKA_CLIENT_ID"); clientID != "" {
		cfg.ClientID = clientID
	}

	// Service name
	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		cfg.ServiceName = serviceName
		if cfg.ClientID == "serphona-client" {
			cfg.ClientID = serviceName
		}
	}

	// Environment
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}

	// Debug mode
	if debug := os.Getenv("DEBUG"); debug != "" {
		cfg.Debug = parseBool(debug)
	}

	// Auto commit
	if autoCommit := os.Getenv("KAFKA_AUTO_COMMIT"); autoCommit != "" {
		cfg.EnableAutoCommit = parseBool(autoCommit)
	}

	// Commit interval
	if interval := os.Getenv("KAFKA_COMMIT_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			cfg.CommitInterval = d
		}
	}

	// Session timeout
	if timeout := os.Getenv("KAFKA_SESSION_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.SessionTimeout = d
		}
	}

	// Publisher batch size
	if batchSize := os.Getenv("KAFKA_PUBLISHER_BATCH_SIZE"); batchSize != "" {
		if size, err := strconv.Atoi(batchSize); err == nil {
			cfg.PublisherBatchSize = size
		}
	}

	// Publisher batch timeout
	if batchTimeout := os.Getenv("KAFKA_PUBLISHER_BATCH_TIMEOUT"); batchTimeout != "" {
		if d, err := time.ParseDuration(batchTimeout); err == nil {
			cfg.PublisherBatchTimeout = d
		}
	}

	// Publisher max retries
	if maxRetries := os.Getenv("KAFKA_PUBLISHER_MAX_RETRIES"); maxRetries != "" {
		if r, err := strconv.Atoi(maxRetries); err == nil {
			cfg.PublisherMaxRetries = r
		}
	}

	// Publisher retry interval
	if retryInterval := os.Getenv("KAFKA_PUBLISHER_RETRY_INTERVAL"); retryInterval != "" {
		if d, err := time.ParseDuration(retryInterval); err == nil {
			cfg.PublisherRetryInterval = d
		}
	}

	// Consumer max retries
	if maxRetries := os.Getenv("KAFKA_CONSUMER_MAX_RETRIES"); maxRetries != "" {
		if r, err := strconv.Atoi(maxRetries); err == nil {
			cfg.ConsumerMaxRetries = r
		}
	}

	// Consumer retry interval
	if retryInterval := os.Getenv("KAFKA_CONSUMER_RETRY_INTERVAL"); retryInterval != "" {
		if d, err := time.ParseDuration(retryInterval); err == nil {
			cfg.ConsumerRetryInterval = d
		}
	}

	// Consumer concurrency
	if concurrency := os.Getenv("KAFKA_CONSUMER_CONCURRENCY"); concurrency != "" {
		if c, err := strconv.Atoi(concurrency); err == nil {
			cfg.ConsumerConcurrency = c
		}
	}

	// TLS
	if tlsEnabled := os.Getenv("KAFKA_TLS"); tlsEnabled != "" {
		cfg.UseTLS = parseBool(tlsEnabled)
	}
	if insecure := os.Getenv("KAFKA_TLS_INSECURE_SKIP_VERIFY"); insecure != "" {
		cfg.TLSInsecureSkipVerify = parseBool(insecure)
	}

	// SASL
	if mechanism := os.Getenv("KAFKA_SASL_MECHANISM"); mechanism != "" {
		cfg.SASLMechanism = strings.ToLower(mechanism)
	}
	if user := os.Getenv("KAFKA_SASL_USERNAME"); user != "" {
		cfg.SASLUsername = user
	}
	if pass := os.Getenv("KAFKA_SASL_PASSWORD"); pass != "" {
		cfg.SASLPassword = pass
	}

	return cfg
}

// parseBool converte string para bool
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes"
}

// Validate valida a configuração
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrNoBrokers
	}

	if c.GroupID == "" {
		return ErrNoGroupID
	}

	if c.ClientID == "" {
		return ErrNoClientID
	}

	if c.ServiceName == "" {
		return ErrNoServiceName
	}

	if c.SASLMechanism != "" {
		if c.SASLMechanism != "plain" && c.SASLMechanism != "scram-sha256" && c.SASLMechanism != "scram-sha512" {
			return ErrUnsupportedSASL
		}
		if c.SASLUsername == "" || c.SASLPassword == "" {
			return ErrMissingSASLCreds
		}
	}

	return nil
}

// Errors
var (
	ErrNoBrokers        = &ConfigError{Message: "no Kafka brokers configured"}
	ErrNoGroupID        = &ConfigError{Message: "no group ID configured"}
	ErrNoClientID       = &ConfigError{Message: "no client ID configured"}
	ErrNoServiceName    = &ConfigError{Message: "no service name configured"}
	ErrUnsupportedSASL  = &ConfigError{Message: "unsupported SASL mechanism (supported: plain, scram-sha256, scram-sha512)"}
	ErrMissingSASLCreds = &ConfigError{Message: "SASL mechanism configured but username or password missing"}
)

// ConfigError representa um erro de configuração
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Message
}
