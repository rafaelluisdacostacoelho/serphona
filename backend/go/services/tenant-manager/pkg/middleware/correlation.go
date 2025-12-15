// Package middleware provides shared HTTP middleware helpers.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// CorrelationID adds a correlation/request ID to the context and response header.
// Useful for chi/net/http handlers outside Gin.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), "request_id", id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
