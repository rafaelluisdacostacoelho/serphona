package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/config"
	"tools-manager/internal/events"
	"tools-manager/internal/handler"
	"tools-manager/internal/metrics"
	"tools-manager/internal/repository"
	"tools-manager/internal/secret"
)

// Router bundles HTTP setup.
type Router struct {
	Engine *gin.Engine
}

// NewRouter constructs the HTTP engine with health, metrics, and stub APIs.
func NewRouter(cfg *config.Config, log *zap.Logger, repo *repository.Repository, secretStore *secret.Store, notifier events.Notifier) *Router {
	gin.SetMode(modeFromEnv(cfg.Server.Env))
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLogger(log))

	// Prometheus instrumentation (basic request counter by path/status).
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tools_manager_http_requests_total",
			Help: "HTTP requests processed",
		},
		[]string{"path", "method", "status"},
	)
	policyDecisionCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tools_manager_policy_decisions_total",
			Help: "Policy decisions by outcome",
		},
		[]string{"decision", "path", "tenant"},
	)
	quotaDecisionCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tools_manager_quota_decisions_total",
			Help: "Quota decisions by outcome",
		},
		[]string{"decision", "path", "tenant"},
	)
	prometheus.MustRegister(requestCounter)
	prometheus.MustRegister(policyDecisionCounter)
	prometheus.MustRegister(quotaDecisionCounter)
	metrics.Register(prometheus.DefaultRegisterer)

	engine.Use(func(c *gin.Context) {
		c.Next()
		status := fmt.Sprintf("%d", c.Writer.Status())
		requestCounter.WithLabelValues(c.FullPath(), c.Request.Method, status).Inc()
	})

	// Health and readiness.
	engine.GET(cfg.Observability.HealthPath, func(c *gin.Context) {
		response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET(cfg.Observability.ReadyPath, func(c *gin.Context) {
		response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"status": "ready"})
	})

	// Metrics.
	engine.GET(cfg.Observability.MetricsPath, gin.WrapH(promhttp.Handler()))

	// pprof endpoints under /debug/pprof.
	pprof.Register(engine)

	toolsHandler := handler.NewToolsHandler(log, repo, notifier)
	policyHandler := handler.NewPolicyHandler(log, repo)
	quotaHandler := handler.NewQuotaHandler(log, repo)
	secretHandler := handler.NewSecretHandler(log, secretStore)
	catalogHandler := handler.NewCatalogHandler(log, repo, notifier)
	qGuard := newQuotaGuard(repo, log, quotaDecisionCounter)

	api := engine.Group("/api/v1")
	api.Use(authmw.RequireAuth(), tenantGuard(), qGuard.Handle(), policyGuard(repo, log, policyDecisionCounter))
	api.GET("/tools", toolsHandler.List)
	api.POST("/tools", toolsHandler.Create)
	api.GET("/catalog/resolved", catalogHandler.Resolved)
	api.GET("/catalog/mcp", catalogHandler.MCP)
	api.GET("/catalog/changes", catalogHandler.Changes)
	api.GET("/policy/rules", policyHandler.List)
	api.POST("/policy/rules", policyHandler.Create)
	api.GET("/quota/rules", quotaHandler.List)
	api.POST("/quota/rules", quotaHandler.Create)
	api.GET("/secrets/:id", secretHandler.Get)
	api.POST("/secrets", secretHandler.Put)

	return &Router{Engine: engine}
}

func modeFromEnv(env string) string {
	if env == "production" {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}

func requestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		latency := time.Since(start)
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

// tenantGuard enforces tenant consistency between claims and headers.
func tenantGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authmw.GetClaimsFromContext(c)
		if err != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing auth claims", nil)
			c.Abort()
			return
		}

		tenantID := claims.TenantID
		if tenantID == "" {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "tenant_id missing in token", nil)
			c.Abort()
			return
		}

		headerTenant := c.GetHeader(authmw.TenantIDHeader)
		if headerTenant == "" {
			c.Request.Header.Set(authmw.TenantIDHeader, tenantID)
		} else if tenantID != "platform" && headerTenant != tenantID {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "tenant_mismatch", "tenant header does not match token", nil)
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(authmw.WithTenantID(c.Request.Context(), tenantID))
		c.Next()
	}
}
