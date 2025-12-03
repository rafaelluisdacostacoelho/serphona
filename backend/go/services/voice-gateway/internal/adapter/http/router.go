// Package http provides HTTP server configuration.
package http

import (
	"net/http"

	"go.uber.org/zap"

	"voice-gateway/internal/adapter/http/handler"
	callservice "voice-gateway/internal/application/call"
)

// NewRouter creates a new HTTP router with all routes configured.
func NewRouter(callService *callservice.Service, logger *zap.Logger) http.Handler {
	mux := http.NewServeMux()

	// Create handlers
	callHandler := handler.NewCallHandler(callService, logger)
	asteriskHandler := handler.NewAsteriskHandler(callService, logger)

	// Health check endpoints
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /health/live", livenessHandler)
	mux.HandleFunc("GET /health/ready", readinessHandler)

	// Call management API
	mux.HandleFunc("GET /api/v1/calls/{call_id}", callHandler.GetCall)
	mux.HandleFunc("DELETE /api/v1/calls/{call_id}", callHandler.EndCall)
	mux.HandleFunc("POST /api/v1/calls/{call_id}/transfer", callHandler.TransferCall)
	mux.HandleFunc("GET /api/v1/tenants/{tenant_id}/calls", callHandler.ListCalls)

	// Asterisk ARI webhooks
	mux.HandleFunc("POST /asterisk/events", asteriskHandler.HandleARIEvent)

	// Apply middleware
	return loggingMiddleware(logger)(corsMiddleware(mux))
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
	// TODO: Check dependencies (Redis, Kafka, Asterisk)
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
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
