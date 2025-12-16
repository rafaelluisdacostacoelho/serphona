package health

import (
	"net/http"
)

// Checker returns an error when unhealthy; nil means healthy.
type Checker func() error

// Handler returns an HTTP handler that exposes liveness/readiness checks.
// - Liveness: always executed; failing returns 500.
// - Readiness: optional; any failure returns 503.
func Handler(liveness Checker, readiness ...Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if liveness != nil {
			if err := liveness(); err != nil {
				http.Error(w, "unhealthy", http.StatusInternalServerError)
				return
			}
		}

		for _, check := range readiness {
			if check == nil {
				continue
			}
			if err := check(); err != nil {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
