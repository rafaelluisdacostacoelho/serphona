package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
)

// RequireAuthHTTP validates a JWT from the Authorization header and injects claims into the request context.
func RequireAuthHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := extractOrGenerateRequestID(r.Header.Get(requestIDHeader))
		w.Header().Set(requestIDHeader, reqID)
		ctx := WithRequestID(r.Context(), reqID)
		safeFields := SafeRequestFields(r)
		ctx = WithSafeRequestFields(ctx, safeFields)
		ctx, span := startAuthSpan(ctx, "platform-auth.require-auth-http", reqID)
		defer span.End()
		cfg := authjwt.GetValidationConfig()
		if needsSecret(cfg) {
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				mapped := mapAuthError(err)
				tenant := tenantFromHeaders(r.Header)
				recordAuthError("http", mapped, tenant, start)
				recordSpanError(span, mapped, err)
				writeJSONError(w, mapped)
				return
			}
		} else {
			_ = authjwt.EnsureSecretLoaded()
		}

		claims, err := authjwt.ValidateTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			mapped := mapAuthError(err)
			tenant := tenantFromHeaders(r.Header)
			recordAuthError("http", mapped, tenant, start)
			recordSpanError(span, mapped, err)
			writeJSONError(w, mapped)
			return
		}

		annotateSpanWithClaims(span, claims)
		ctx = WithClaims(ctx, claims)
		recordAuthSuccess("http", claims.TenantID, start)
		recordSpanSuccess(span)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScopesHTTP enforces that the request context contains all specified scopes.
func RequireScopesHTTP(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				writeJSONError(w, mapAuthError(autherrors.ErrUnauthorized))
				return
			}

			if !claims.HasAllScopes(scopes...) {
				writeJSONError(w, mapAuthError(autherrors.ErrInsufficientPermissions))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyScopeHTTP enforces that the request context contains at least one of the specified scopes.
func RequireAnyScopeHTTP(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				writeJSONError(w, mapAuthError(autherrors.ErrUnauthorized))
				return
			}

			if !claims.HasAnyScope(scopes...) {
				writeJSONError(w, mapAuthError(autherrors.ErrInsufficientPermissions))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ChiRequireAuth aliases RequireAuthHTTP for chi routers.
func ChiRequireAuth(next http.Handler) http.Handler {
	return RequireAuthHTTP(next)
}

// ChiRequireScopes aliases RequireScopesHTTP for chi routers.
func ChiRequireScopes(scopes ...string) func(http.Handler) http.Handler {
	return RequireScopesHTTP(scopes...)
}

// ChiRequireAnyScope aliases RequireAnyScopeHTTP for chi routers.
func ChiRequireAnyScope(scopes ...string) func(http.Handler) http.Handler {
	return RequireAnyScopeHTTP(scopes...)
}

func writeJSONError(w http.ResponseWriter, mapped authMappedError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(mapped.status)
	_ = json.NewEncoder(w).Encode(errorPayload(mapped))
}
