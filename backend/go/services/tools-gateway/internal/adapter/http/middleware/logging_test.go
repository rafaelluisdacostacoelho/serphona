package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerRedactsSensitiveHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	router := gin.New()
	router.Use(RequestLogger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Cookie", "sessionid=super-secret")
	router.ServeHTTP(httptest.NewRecorder(), req)

	output := buf.String()

	if strings.Contains(output, "secret-token") || strings.Contains(output, "super-secret") {
		t.Fatalf("log output leaked sensitive values: %s", output)
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("expected sensitive headers to be redacted, got: %s", output)
	}

	if !strings.Contains(output, "\"status\":200") {
		t.Fatalf("expected status in log output, got: %s", output)
	}

	if !strings.Contains(output, "\"path\":\"/test\"") {
		t.Fatalf("expected path in log output, got: %s", output)
	}
}
