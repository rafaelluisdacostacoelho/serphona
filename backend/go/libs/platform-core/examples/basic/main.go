package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core/config"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core/health"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core/logger"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core/secrets"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := config.ValidateRequired(cfg, "HTTP_ADDR"); err != nil {
		log.Fatalf("config validation: %v", err)
	}

	zlog, err := logger.NewWithMeta(cfg.LogLevel, "platform-core-example", cfg.Environment, "1.0.0")
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer zlog.Sync()

	// Example of secret retrieval (env-backed)
	dbURL, err := secrets.Get("DATABASE_URL")
	if err != nil {
		zlog.Warn("database url missing, using config fallback", zap.Error(err))
		dbURL = cfg.DatabaseURL
	}

	// Simulated readiness check for dependencies (replace with real ping)
	readiness := func() error {
		if dbURL == "" {
			return fmt.Errorf("database unavailable")
		}
		if len(cfg.KafkaBrokers) == 0 {
			return fmt.Errorf("kafka brokers not configured")
		}
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health.Handler(
		func() error { return nil }, // liveness
		readiness,
	))
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("env=%s log=%s db=%t kafka=%d", cfg.Environment, cfg.LogLevel, dbURL != "", len(cfg.KafkaBrokers))))
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	zlog.Info("starting example", zap.String("addr", cfg.HTTPAddr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zlog.Fatal("server failed", zap.Error(err))
	}
}
