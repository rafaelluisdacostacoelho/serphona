// Package main is the entry point for the voice-gateway service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/agent"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/asterisk"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/events"
	httpadapter "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/http"
	redisadapter "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/redis"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/stt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tenant"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tts"
	callservice "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/application/call"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := initLogger(cfg.LogLevel, cfg.Environment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("starting voice-gateway service",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
	)

	// Initialize infrastructure
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tenantClient := tenant.NewClient(cfg.TenantManager.URL, cfg.TenantManager.Token, cfg.ServiceName, cfg.ServiceInstance, cfg.ServiceAudience, log)
	agentClient := agent.NewClient(cfg.AgentOrchestrator.URL, cfg.AgentOrchestrator.Token, cfg.ServiceName, cfg.ServiceInstance, cfg.ServiceAudience, log)

	asteriskClient, err := asterisk.NewARIClientHTTP(asterisk.ARIConfig{
		URL:      cfg.Asterisk.ARIURL,
		Username: cfg.Asterisk.ARIUsername,
		Password: cfg.Asterisk.ARIPassword,
		AppName:  cfg.Asterisk.ARIAppName,
	}, log)
	if err != nil {
		log.Fatal("failed to initialize Asterisk ARI client", zap.Error(err))
	}
	defer asteriskClient.Close()

	redisClient, err := redisadapter.NewClient(ctx, cfg.Redis.URL, cfg.Redis.Password, cfg.Redis.DB, log)
	if err != nil {
		log.Fatal("failed to initialize redis client", zap.Error(err))
	}
	defer redisClient.Close()

	callStateRepo := redisadapter.NewCallStateRepository(redisClient, cfg.Redis.CallStateTTL)

	eventPublisher, err := events.NewPublisher(cfg.Kafka.Brokers, cfg.Kafka.TopicPrefix, log)
	if err != nil {
		log.Fatal("failed to initialize event publisher", zap.Error(err))
	}
	defer eventPublisher.Close()

	sttProviders := make(map[string]stt.Provider)
	ttsProviders := make(map[string]tts.Provider)

	providerClosers := make([]func(), 0)

	if cfg.STT.GoogleProjectID != "" {
		googleSTT, err := stt.NewGoogleProviderV2(cfg.STT.GoogleProjectID, cfg.STT.GoogleCredentials, log)
		if err != nil {
			log.Warn("failed to initialize google stt provider", zap.Error(err))
		} else {
			sttProviders[string(stt.ProviderGoogle)] = googleSTT
			providerClosers = append(providerClosers, func() {
				if err := googleSTT.Close(); err != nil {
					log.Warn("failed to close google stt provider", zap.Error(err))
				}
			})
			log.Info("google stt provider registered")
		}
	}

	if cfg.TTS.GoogleProjectID != "" {
		googleTTS, err := tts.NewGoogleProviderV2(cfg.TTS.GoogleProjectID, cfg.TTS.GoogleCredentials, log)
		if err != nil {
			log.Warn("failed to initialize google tts provider", zap.Error(err))
		} else {
			ttsProviders[string(tts.ProviderGoogle)] = googleTTS
			providerClosers = append(providerClosers, func() {
				if err := googleTTS.Close(); err != nil {
					log.Warn("failed to close google tts provider", zap.Error(err))
				}
			})
			log.Info("google tts provider registered")
		}
	}

	if cfg.TTS.ElevenLabsAPIKey != "" {
		elevenLabsTTS, err := tts.NewElevenLabsProviderV2(cfg.TTS.ElevenLabsAPIKey, log)
		if err != nil {
			log.Warn("failed to initialize elevenlabs tts provider", zap.Error(err))
		} else {
			ttsProviders[string(tts.ProviderElevenLabs)] = elevenLabsTTS
			providerClosers = append(providerClosers, func() {
				if err := elevenLabsTTS.Close(); err != nil {
					log.Warn("failed to close elevenlabs tts provider", zap.Error(err))
				}
			})
			log.Info("elevenlabs tts provider registered")
		}
	}

	callSvc := callservice.NewService(
		asteriskClient,
		callStateRepo,
		eventPublisher,
		tenantClient,
		agentClient,
		sttProviders,
		ttsProviders,
		cfg.Call.MaxConcurrentCalls,
		log,
	)

	// Metrics server (separate port for Prometheus scraping)
	metricsServer := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Metrics.Port),
		Handler: func() http.Handler {
			mux := http.NewServeMux()
			mux.Handle(cfg.Metrics.Path, promhttp.Handler())
			return mux
		}(),
	}

	// HTTP server for management API
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      httpadapter.NewRouter(callSvc, log, redisClient, eventPublisher, asteriskClient, tenantClient, cfg.Server.CORSAllowedOrigins, cfg.Asterisk),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start servers
	errChan := make(chan error, 2)

	// Start HTTP server
	go func() {
		log.Info("starting HTTP server", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("http server error: %w", err)
		}
	}()

	// Start metrics server
	go func() {
		log.Info("starting metrics server", zap.Int("port", cfg.Metrics.Port))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Error("server error", zap.Error(err))
	case sig := <-quit:
		log.Info("received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	log.Info("shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown error", zap.Error(err))
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}

	log.Info("servers stopped")

	for _, closer := range providerClosers {
		closer()
	}
}

// initLogger initializes the logger with the specified level and environment.
func initLogger(logLevel, environment string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Parse log level
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	config.Level = level

	return config.Build()
}

// buildHTTPRouter builds the HTTP router for the management API.
func buildHTTPRouter(logger *zap.Logger) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	return mux
}
