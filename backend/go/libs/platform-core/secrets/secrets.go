package secrets

import (
	"fmt"
	"os"
)

// Provider defines how secrets are retrieved.
type Provider interface {
	Get(key string) (string, error)
}

var defaultProvider Provider = envProvider{}

// SetProvider overrides the default provider (useful to plug a secret manager).
func SetProvider(p Provider) {
	if p == nil {
		return
	}
	defaultProvider = p
}

// Get returns the secret using the configured provider (env by default).
func Get(key string) (string, error) {
	if defaultProvider == nil {
		return "", fmt.Errorf("secret provider not configured")
	}
	return defaultProvider.Get(key)
}

type envProvider struct{}

func (envProvider) Get(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return val, nil
}
