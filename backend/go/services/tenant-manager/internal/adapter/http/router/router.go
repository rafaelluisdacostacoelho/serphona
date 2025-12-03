// Package router provides HTTP router configuration.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	httphandler "tenant-manager/internal/adapter/http/handler"
)

// Config holds router configuration.
type Config struct {
	healthHandler    *httphandler.HealthHandler
	tenantHandler    *httphandler.TenantHandler
	apiKeyHandler    *httphandler.APIKeyHandler
	middlewares      []func(http.Handler) http.Handler
	authMiddleware   func(http.Handler) http.Handler
	tenantMiddleware func(http.Handler) http.Handler
}

// Option is a router configuration option.
type Option func(*Config)

// WithHealthHandler sets the health handler.
func WithHealthHandler(h *httphandler.HealthHandler) Option {
	return func(c *Config) {
		c.healthHandler = h
	}
}

// WithTenantHandler sets the tenant handler.
func WithTenantHandler(h *httphandler.TenantHandler) Option {
	return func(c *Config) {
		c.tenantHandler = h
	}
}

// WithAPIKeyHandler sets the API key handler.
func WithAPIKeyHandler(h *httphandler.APIKeyHandler) Option {
	return func(c *Config) {
		c.apiKeyHandler = h
	}
}

// WithMiddleware adds global middleware.
func WithMiddleware(mw ...func(http.Handler) http.Handler) Option {
	return func(c *Config) {
		c.middlewares = append(c.middlewares, mw...)
	}
}

// WithAuthMiddleware sets the auth middleware.
func WithAuthMiddleware(mw func(http.Handler) http.Handler) Option {
	return func(c *Config) {
		c.authMiddleware = mw
	}
}

// WithTenantMiddleware sets the tenant middleware.
func WithTenantMiddleware(mw func(http.Handler) http.Handler) Option {
	return func(c *Config) {
		c.tenantMiddleware = mw
	}
}

// New creates a new HTTP router.
func New(opts ...Option) http.Handler {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	r := chi.NewRouter()

	// Apply global middlewares
	for _, mw := range cfg.middlewares {
		r.Use(mw)
	}

	// Health check endpoint (no auth required)
	if cfg.healthHandler != nil {
		r.Get("/health", cfg.healthHandler.Health)
		r.Get("/health/live", cfg.healthHandler.Live)
		r.Get("/health/ready", cfg.healthHandler.Ready)
	}

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Tenant routes
		if cfg.tenantHandler != nil {
			r.Route("/tenants", func(r chi.Router) {
				r.Get("/", cfg.tenantHandler.List)
				r.Post("/", cfg.tenantHandler.Create)
				r.Get("/{id}", cfg.tenantHandler.Get)
				r.Put("/{id}", cfg.tenantHandler.Update)
				r.Delete("/{id}", cfg.tenantHandler.Delete)
			})
		}

		// API Key routes (if handler exists)
		if cfg.apiKeyHandler != nil {
			r.Route("/api-keys", func(r chi.Router) {
				r.Get("/", cfg.apiKeyHandler.List)
				r.Post("/", cfg.apiKeyHandler.Create)
			})
		}
	})

	return r
}
