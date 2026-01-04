package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
)

// RequestLogger logs sanitized request/response metadata with redaction applied by platform-auth helpers.
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		safe := authmw.SafeRequestFieldsFromGin(c)
		fields := []zap.Field{
			zap.String("event", "http_request"),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("duration_ms", time.Since(start).Milliseconds()),
		}

		if route := c.FullPath(); route != "" {
			fields = append(fields, zap.String("route", route))
		}

		if reqID := c.GetString("request_id"); reqID != "" {
			fields = append(fields, zap.String("request_id", reqID))
		}

		for k, v := range safe {
			fields = append(fields, zap.Any(k, v))
		}

		logger.Info("http_request", fields...)
	}
}
