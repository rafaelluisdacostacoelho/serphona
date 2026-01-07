// Package http provides HTTP server configuration.
package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/asterisk"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/events"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tenant"
	callservice "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/application/call"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/config"
)

// NewRouter creates a new HTTP router with all routes configured.
func NewRouter(callService *callservice.Service, logger *zap.Logger, redisClient *redis.Client, kafkaClient events.Publisher, asteriskClient *asterisk.ARIClientHTTP, tenantClient *tenant.Client, allowedOrigins []string, astCfg config.AsteriskConfig) http.Handler {
	mux := http.NewServeMux()

	// Create handlers
	callHandler := handler.NewCallHandler(callService, logger)
	asteriskHandler := handler.NewAsteriskHandler(callService, tenantClient, redisClient, astCfg, logger)

	// Health check endpoints
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /health/live", livenessHandler)
	mux.HandleFunc("GET /health/ready", readinessHandler)

	// Call management API
	mux.Handle("GET /api/v1/calls/{call_id}", authmw.RequireAuthHTTP(http.HandlerFunc(callHandler.GetCall)))
	mux.Handle("DELETE /api/v1/calls/{call_id}", authmw.RequireAuthHTTP(http.HandlerFunc(callHandler.EndCall)))
	mux.Handle("POST /api/v1/calls/{call_id}/transfer", authmw.RequireAuthHTTP(http.HandlerFunc(callHandler.TransferCall)))
	mux.Handle("GET /api/v1/tenants/{tenant_id}/calls", authmw.RequireAuthHTTP(http.HandlerFunc(callHandler.ListCalls)))

	// Asterisk ARI webhooks
	mux.Handle("POST /asterisk/events", authmw.RequireAuthHTTP(http.HandlerFunc(asteriskHandler.HandleARIEvent)))

	// Passar os clientes para as funções de verificação
	checkRedisConnection = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return redisClient.Ping(ctx).Err()
	}
	checkKafkaConnection = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return kafkaClient.HealthCheck(ctx)
	}
	checkAsteriskConnection = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return asteriskClient.HealthCheck(ctx)
	}

	// Apply middleware
	return loggingMiddleware(logger)(corsMiddleware(allowedOrigins, logger)(mux))
}

// healthHandler handles general health checks.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"voice-gateway"}`))
}

// livenessHandler handles Kubernetes liveness probes.
func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive"}`))
}

// readinessHandler handles Kubernetes readiness probes.
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	// Check Redis connection
	if err := checkRedisConnection(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unready","reason":"redis_unavailable"}`))
		return
	}

	// Check Kafka connection
	if err := checkKafkaConnection(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unready","reason":"kafka_unavailable"}`))
		return
	}

	// Check Asterisk connection
	if err := checkAsteriskConnection(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unready","reason":"asterisk_unavailable"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// loggingMiddleware logs HTTP requests.
func loggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
			)
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware adds CORS headers.
func corsMiddleware(allowedOrigins []string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := ""
			if len(allowedOrigins) == 0 {
				allowed = "" // No CORS allowed unless explicitly configured
			} else if allowedOrigins[0] == "*" {
				allowed = "*"
			} else if origin != "" {
				for _, o := range allowedOrigins {
					if strings.EqualFold(o, origin) {
						allowed = origin
						break
					}
				}
			}

			if allowed == "" {
				if origin != "" {
					logger.Debug("cors rejected origin", zap.String("origin", origin))
				}
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Implementação da verificação de conexão com Redis
var checkRedisConnection = func() error {
	return nil
}

// Implementação da verificação de conexão com Kafka
var checkKafkaConnection = func() error {
	return nil
}

// Implementação da verificação de conexão com Asterisk
var checkAsteriskConnection = func() error {
	return nil
}
