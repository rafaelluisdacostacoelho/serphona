package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/audit"
	"tools-manager/internal/quota"
	"tools-manager/internal/repository"
)

// quotaGuard enforces quota rules using an in-memory limiter.
type quotaGuard struct {
	repo    *repository.Repository
	log     *zap.Logger
	counter *prometheus.CounterVec
	limiter *quota.Limiter
	mu      sync.Mutex
}

func newQuotaGuard(repo *repository.Repository, log *zap.Logger, counter *prometheus.CounterVec) *quotaGuard {
	return &quotaGuard{repo: repo, log: log, counter: counter, limiter: quota.NewLimiter()}
}

func (q *quotaGuard) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authmw.GetClaimsFromContext(c)
		if err != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
			c.Abort()
			return
		}

		// Load rules (cached per request for now)
		rules, err := q.repo.ListQuotaRules(c.Request.Context(), claims.TenantID)
		if err != nil {
			q.log.Error("failed to load quota rules", zap.Error(err))
			response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to evaluate quota", nil)
			c.Abort()
			return
		}

		evalRules := make([]quota.Rule, 0, len(rules))
		for _, r := range rules {
			evalRules = append(evalRules, quota.Rule{
				TenantID:       uuidPtrToString(r.Tenant),
				ToolID:         uuidPtrToString(r.ToolID),
				AgentID:        ptrOrEmpty(r.AgentID),
				Env:            ptrOrEmpty(r.Env),
				LimitPerMinute: r.LimitPerMinute,
				LimitPerDay:    r.LimitPerDay,
				Weight:         r.Weight,
			})
		}

		toolID := c.Request.Header.Get("X-Tool-ID")
		decision := q.limiter.Evaluate(time.Now(), quota.Request{
			TenantID: claims.TenantID,
			ToolID:   toolID,
			AgentID:  claims.Service,
			Env:      c.Request.Header.Get("X-Env"),
		}, evalRules)

		q.counter.WithLabelValues(outcome(decision), c.FullPath(), claims.TenantID).Inc()
		audit.Emit(q.log, audit.Event{
			Category: "quota_decision",
			Action:   c.Request.Method,
			Outcome:  outcome(decision),
			TenantID: claims.TenantID,
			UserID:   claims.UserID,
			Service:  claims.Service,
			ToolID:   toolID,
			Path:     c.FullPath(),
			Reason:   decision.Reason,
		})

		if !decision.Allowed {
			q.log.Info("quota denied", zap.String("tenant_id", claims.TenantID), zap.String("path", c.FullPath()), zap.String("reason", decision.Reason))
			response.WriteError(c.Request.Context(), c.Writer, http.StatusTooManyRequests, "quota_exceeded", decision.Reason, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

func outcome(decision quota.Decision) string {
	if decision.Allowed {
		return "allow"
	}
	return decision.Reason
}
