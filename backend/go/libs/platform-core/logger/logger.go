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
