package client

import "testing"

func TestNewFromEnvSuccess(t *testing.T) {
	t.Setenv(EnvAuthGatewayURL, "http://auth-gateway:8080")

	cl, err := NewFromEnv()
	if err != nil {
		t.Fatalf("expected client to be created, got: %v", err)
	}

	if cl == nil {
		t.Fatalf("expected client instance, got nil")
	}
}

func TestNewFromEnvMissing(t *testing.T) {
	t.Setenv(EnvAuthGatewayURL, "")

	if _, err := NewFromEnv(); err == nil {
		t.Fatalf("expected error when %s is missing", EnvAuthGatewayURL)
	}
}
