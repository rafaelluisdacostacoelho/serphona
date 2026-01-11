package server

import (
	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("tools-manager/http")

// traceMiddleware starts a server span and tags it with tenant/user/tool metadata when available.
func traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}

		ctx, span := tracer.Start(c.Request.Context(), spanName, trace.WithSpanKind(trace.SpanKindServer))

		attrs := []attribute.KeyValue{
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.path", c.FullPath()),
		}

		if claims, err := authmw.GetClaimsFromContext(c); err == nil {
			attrs = append(attrs,
				attribute.String("tenant.id", claims.TenantID),
				attribute.String("user.id", claims.UserID),
				attribute.String("service", claims.Service),
			)
		}

		if toolID := c.Request.Header.Get("X-Tool-ID"); toolID != "" {
			attrs = append(attrs, attribute.String("tool.id", toolID))
		}
		if toolVersion := c.Request.Header.Get("X-Tool-Version"); toolVersion != "" {
			attrs = append(attrs, attribute.String("tool.version", toolVersion))
		}

		span.SetAttributes(attrs...)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
		span.End()
	}
}
