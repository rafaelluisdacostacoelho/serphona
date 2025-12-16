package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-core/config"
	"github.com/serphona/serphona/backend/go/libs/platform-core/health"
	"github.com/serphona/serphona/backend/go/libs/platform-core/logger"
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

	zlog, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer zlog.Sync()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health.Handler(
		func() error { return nil }, // liveness
		func() error { return nil }, // readiness stub, replace with real checks
	))
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("env=%s log=%s", cfg.Environment, cfg.LogLevel)))
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
