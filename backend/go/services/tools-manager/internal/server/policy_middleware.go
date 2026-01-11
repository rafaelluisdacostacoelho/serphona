package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/audit"
	"tools-manager/internal/policy"
	"tools-manager/internal/repository"
)

// policyGuard loads policy rules for the tenant and evaluates allow/deny.
func policyGuard(repo *repository.Repository, log *zap.Logger, counter *prometheus.CounterVec) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authmw.GetClaimsFromContext(c)
		if err != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
			c.Abort()
			return
		}

		rules, err := repo.ListPolicyRules(c.Request.Context(), claims.TenantID)
		if err != nil {
			log.Error("failed to load policy rules", zap.Error(err))
			response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to evaluate policy", nil)
			c.Abort()
			return
		}

		evalRules := make([]policy.Rule, 0, len(rules))
		for _, r := range rules {
			evalRules = append(evalRules, policy.Rule{
				Effect:   policy.Decision(r.Effect),
				TenantID: stringOrEmpty(r.Tenant),
				AgentID:  ptrOrEmpty(r.AgentID),
				ToolID:   uuidPtrToString(r.ToolID),
				Env:      ptrOrEmpty(r.Env),
				Scopes:   r.Scopes,
				Roles:    r.Roles,
				Weight:   r.Weight,
			})
		}

		evaluator := policy.NewEvaluator(evalRules)
		decision := evaluator.Evaluate(policy.Request{
			TenantID: claims.TenantID,
			AgentID:  claims.Service,
			Scopes:   claims.Scopes,
			Roles:    []string{claims.Role},
			Env:      c.Request.Header.Get("X-Env"),
		})

		counter.WithLabelValues(decision.String(), c.FullPath(), claims.TenantID).Inc()
		log.Info("policy decision", zap.String("decision", decision.String()), zap.String("tenant_id", claims.TenantID), zap.String("path", c.FullPath()), zap.String("user_id", claims.UserID), zap.String("service", claims.Service))
		audit.Emit(log, audit.Event{
			Category: "policy_decision",
			Action:   c.Request.Method,
			Outcome:  decision.String(),
			TenantID: claims.TenantID,
			UserID:   claims.UserID,
			Service:  claims.Service,
			Path:     c.FullPath(),
			ToolID:   uuidPtrToString(nil),
		})

		if decision == policy.DecisionDeny {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "forbidden", "policy deny", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

func stringOrEmpty(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func uuidPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
