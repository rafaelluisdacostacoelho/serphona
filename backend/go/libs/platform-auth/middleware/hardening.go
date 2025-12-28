package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// HTTPRecovery guards net/http handlers against panics and returns a structured 500 error.
func HTTPRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeJSONError(w, internalErrorMapped)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// GinRecovery guards Gin handlers against panics and returns a structured 500 error.
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.AbortWithStatusJSON(internalErrorMapped.status, errorPayload(internalErrorMapped))
			}
		}()
		c.Next()
	}
}

// LimitBodySizeHTTP rejects requests exceeding maxBytes (Content-Length check) and caps reader.
func LimitBodySizeHTTP(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes > 0 && r.ContentLength > maxBytes {
				writeJSONError(w, requestTooLargeMapped)
				return
			}
			if maxBytes > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LimitBodySizeGin rejects requests exceeding maxBytes (Content-Length check) and caps reader.
func LimitBodySizeGin(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(requestTooLargeMapped.status, errorPayload(requestTooLargeMapped))
			return
		}
		if maxBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// CORSPolicy defines simple CORS settings.
type CORSPolicy struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// CORSMiddlewareHTTP applies a basic CORS policy for net/http handlers.
func CORSMiddlewareHTTP(policy CORSPolicy) func(http.Handler) http.Handler {
	allowedOrigins := normalizeList(policy.AllowedOrigins)
	allowedMethods := normalizeListWithDefault(policy.AllowedMethods, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	allowedHeaders := normalizeListWithDefault(policy.AllowedHeaders, []string{"Authorization", "Content-Type", requestIDHeader})
	maxAge := int(policy.MaxAge.Seconds())

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (contains(allowedOrigins, "*") || contains(allowedOrigins, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if policy.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			w.Header().Set("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
				if maxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GinCORS applies a basic CORS policy for Gin.
func GinCORS(policy CORSPolicy) gin.HandlerFunc {
	allowedOrigins := normalizeList(policy.AllowedOrigins)
	allowedMethods := normalizeListWithDefault(policy.AllowedMethods, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	allowedHeaders := normalizeListWithDefault(policy.AllowedHeaders, []string{"Authorization", "Content-Type", requestIDHeader})
	maxAge := int(policy.MaxAge.Seconds())

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (contains(allowedOrigins, "*") || contains(allowedOrigins, origin)) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			if policy.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		c.Writer.Header().Set("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
			if maxAge > 0 {
				c.Writer.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeList(vals []string) []string {
	out := []string{}
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func normalizeListWithDefault(vals []string, def []string) []string {
	res := normalizeList(vals)
	if len(res) == 0 {
		return def
	}
	return res
}

func contains(list []string, needle string) bool {
	for _, v := range list {
		if v == needle {
			return true
		}
	}
	return false
}
