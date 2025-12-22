package middleware

import (
    "net/http"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTP wraps a handler with OpenTelemetry instrumentation.
func HTTP(handler http.Handler, service string) http.Handler {
    return otelhttp.NewHandler(handler, service)
}
