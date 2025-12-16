package logger

import "testing"

func TestNewLoggerValidLevel(t *testing.T) {
	log, err := New("debug")
	if err != nil {
		t.Fatalf("expected logger, got error: %v", err)
	}
	log.Sync()
}

func TestNewLoggerInvalidLevel(t *testing.T) {
	if _, err := New("invalid-level"); err == nil {
		t.Fatalf("expected error for invalid level")
	}
}
