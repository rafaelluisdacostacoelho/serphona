package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// RequestLogger logs sanitized request/response details with sensitive headers redacted.
func RequestLogger(l *log.Logger) gin.HandlerFunc {
	if l == nil {
		l = log.Default()
	}

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		safe := authmw.SafeRequestFieldsFromGin(c)
		payload := map[string]any{
			"event":       "http_request",
			"status":      c.Writer.Status(),
			"duration_ms": time.Since(start).Milliseconds(),
		}

		for k, v := range safe {
			payload[k] = v
		}

		if route := c.FullPath(); route != "" {
			payload["route"] = route
		}

		if data, err := json.Marshal(payload); err == nil {
			l.Println(string(data))
			return
		}

		l.Printf("http_request status=%d duration_ms=%d method=%s path=%s", c.Writer.Status(), time.Since(start).Milliseconds(), c.Request.Method, c.FullPath())
	}
}
