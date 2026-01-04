package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/metrics"
)

// Metrics records HTTP latency/counters with service/tenant labels.
func Metrics(m metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		tenant := tenantID(c)
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		reqSize := c.Request.ContentLength
		if reqSize < 0 {
			reqSize = 0
		}

		m.RecordHTTPRequest(
			c.Request.Method,
			path,
			tenant,
			c.Writer.Status(),
			time.Since(start),
			reqSize,
			int64(c.Writer.Size()),
		)
	}
}

func tenantID(c *gin.Context) string {
	if t, err := authmw.GetTenantIDFromContext(c); err == nil && t != "" {
		return t
	}
	if h := c.GetHeader("X-Tenant-Id"); h != "" {
		return h
	}
	return "unknown"
}
