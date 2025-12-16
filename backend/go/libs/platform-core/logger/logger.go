package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a zap logger with the provided level (e.g., "info", "debug", "warn", "error").
// Defaults to info if level is empty.
func New(level string) (*zap.Logger, error) {
	lvl := zapcore.InfoLevel
	if level != "" {
		if err := lvl.Set(level); err != nil {
			return nil, fmt.Errorf("invalid log level %q: %w", level, err)
		}
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.MessageKey = "msg"

	return cfg.Build()
}

// NewWithMeta creates a logger and attaches common service metadata fields.
func NewWithMeta(level, service, environment, version string) (*zap.Logger, error) {
	base, err := New(level)
	if err != nil {
		return nil, err
	}
	fields := []zap.Field{}
	if service != "" {
		fields = append(fields, zap.String("service", service))
	}
	if environment != "" {
		fields = append(fields, zap.String("env", environment))
	}
	if version != "" {
		fields = append(fields, zap.String("version", version))
	}
	if len(fields) == 0 {
		return base, nil
	}
	return base.With(fields...), nil
}
