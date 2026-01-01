package middleware

import (
	"context"
	"crypto/rand"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/otel/trace"
)

// Correlation ensures request and trace identifiers are attached to the context and echoed back.
func Correlation() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)

		ctx := authmw.WithRequestID(c.Request.Context(), reqID)
		ctx = withTraceContext(ctx)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func withTraceContext(ctx context.Context) context.Context {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return ctx
	}

	var tid trace.TraceID
	_, _ = rand.Read(tid[:])
	var sid trace.SpanID
	_, _ = rand.Read(sid[:])

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(ctx, sc)
}
