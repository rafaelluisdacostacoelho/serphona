// Package middleware provides HTTP middlewares for Gin.
package middleware

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimit applies a simple IP-based rate limiter (requests per minute). A non-positive rpm disables limiting.
func RateLimit(rpm int) gin.HandlerFunc {
	if rpm <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	limit := rate.Limit(float64(rpm) / 60.0)
	bucket := rpm // burst

	limiter := struct {
		mu sync.Mutex
		m  map[string]*rate.Limiter
	}{m: make(map[string]*rate.Limiter)}

	getLimiter := func(ip string) *rate.Limiter {
		limiter.mu.Lock()
		defer limiter.mu.Unlock()
		if l, ok := limiter.m[ip]; ok {
			return l
		}
		l := rate.NewLimiter(limit, bucket)
		limiter.m[ip] = l
		return l
	}

	return func(c *gin.Context) {
		ip := clientIP(c.Request)
		if ip == "" {
			ip = "unknown"
		}
		if !getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// TenantRateMetrics tracks allow/deny counts per tenant for observability.
type TenantRateMetrics struct {
	mu      sync.Mutex
	allow   map[string]int64
	blocked map[string]int64
}

// NewTenantRateMetrics creates a new metrics accumulator.
func NewTenantRateMetrics() *TenantRateMetrics {
	return &TenantRateMetrics{allow: make(map[string]int64), blocked: make(map[string]int64)}
}

// RecordAllowed increments the allowed counter for a tenant.
func (m *TenantRateMetrics) RecordAllowed(tenantID string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allow[tenantID]++
}

// RecordBlocked increments the blocked counter for a tenant.
func (m *TenantRateMetrics) RecordBlocked(tenantID string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocked[tenantID]++
}

// TenantRateSnapshot provides a read-only view of metrics.
type TenantRateSnapshot struct {
	Allowed int64
	Blocked int64
}

// Snapshot returns copies of counters per tenant.
func (m *TenantRateMetrics) Snapshot() map[string]TenantRateSnapshot {
	result := make(map[string]TenantRateSnapshot)
	if m == nil {
		return result
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for tenantID, allowed := range m.allow {
		result[tenantID] = TenantRateSnapshot{Allowed: allowed, Blocked: m.blocked[tenantID]}
	}
	for tenantID, blocked := range m.blocked {
		if _, ok := result[tenantID]; !ok {
			result[tenantID] = TenantRateSnapshot{Allowed: m.allow[tenantID], Blocked: blocked}
		}
	}
	return result
}

// TenantRateLimiter enforces per-tenant limits and records metrics.
type TenantRateLimiter struct {
	limit    rate.Limit
	burst    int
	metrics  *TenantRateMetrics
	mu       sync.Mutex
	byTenant map[string]*rate.Limiter
}

// NewTenantRateLimiter builds a limiter; nil when disabled.
func NewTenantRateLimiter(rpm int, metrics *TenantRateMetrics) *TenantRateLimiter {
	if rpm <= 0 {
		return nil
	}
	return &TenantRateLimiter{
		limit:    rate.Limit(float64(rpm) / 60.0),
		burst:    rpm,
		metrics:  metrics,
		byTenant: make(map[string]*rate.Limiter),
	}
}

// Allow checks quota for a tenant and updates metrics.
func (l *TenantRateLimiter) Allow(tenantID string) bool {
	if l == nil || tenantID == "" {
		return true
	}

	l.mu.Lock()
	limiter, ok := l.byTenant[tenantID]
	if !ok {
		limiter = rate.NewLimiter(l.limit, l.burst)
		l.byTenant[tenantID] = limiter
	}
	l.mu.Unlock()

	if limiter.Allow() {
		l.metrics.RecordAllowed(tenantID)
		return true
	}

	l.metrics.RecordBlocked(tenantID)
	return false
}

// TenantRateLimit applies a tenant-scoped rate limiter using JWT claims. Missing tenant IDs are skipped.
func TenantRateLimit(rpm int, metrics *TenantRateMetrics) gin.HandlerFunc {
	return TenantRateLimitWithLimiter(NewTenantRateLimiter(rpm, metrics))
}

// TenantRateLimitWithLimiter reuses a shared limiter across transports.
func TenantRateLimitWithLimiter(limiter *TenantRateLimiter) gin.HandlerFunc {
	if limiter == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		tenantID, err := authmw.TenantIDFromContext(c.Request.Context())
		if err != nil || tenantID == "" {
			c.Next()
			return
		}

		if limiter.Allow(tenantID) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "tenant rate limit exceeded"})
	}
}
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return ip
}

// BodyLimit caps request body size. Non-positive disables.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// CORSWithConfig applies a restrictive CORS policy.
func CORSWithConfig(cfg CORSConfig) gin.HandlerFunc {
	allowedMethods := cfg.AllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	allowedHeaders := cfg.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"Authorization", "Content-Type", "X-Request-ID"}
	}
	allowAll := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && !allowAll && !originAllowed(origin, cfg.AllowedOrigins) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}
		c.Header("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", int(cfg.MaxAge/time.Second)))
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}

// RequestID adds a request ID to each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Inject into both legacy and platform-auth contexts so envelopes can emit IDs.
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
		ctx = authmw.WithRequestID(ctx, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// ZapLogger logs HTTP requests using zap.
func ZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		requestID, _ := c.Get("request_id")

		logger.Info("HTTP request",
			zap.String("request_id", requestID.(string)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		)
	}
}

// CORS adds CORS headers.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// JWTAuth validates JWT tokens.
func JWTAuth(secret, publicKey, issuer string, audience []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorized(c, "Authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c, "Invalid authorization header format")
			return
		}
		tokenStr := parts[1]

		claims, err := ParseJWT(tokenStr, secret, publicKey, issuer, audience)
		if err != nil {
			unauthorized(c, err.Error())
			return
		}

		claimsObj := buildClaims(claims)
		if claimsObj.TenantID == "" || claimsObj.UserID == "" {
			unauthorized(c, "Missing tenant_id or sub claim")
			return
		}

		c.Set("tenant_id", claimsObj.TenantID)
		c.Set("user_id", claimsObj.UserID)
		c.Set("claims", claimsObj)

		ctx := authmw.WithClaims(c.Request.Context(), claimsObj)
		ctx = authmw.WithTenantID(ctx, claimsObj.TenantID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func buildClaims(raw map[string]interface{}) *types.Claims {
	claims := &types.Claims{}

	if tenantID, _ := raw["tenant_id"].(string); tenantID != "" {
		claims.TenantID = tenantID
	}
	if sub, _ := raw["sub"].(string); sub != "" {
		claims.UserID = sub
	}
	if role, _ := raw["role"].(string); role != "" {
		claims.Role = role
	}

	// Accept either "scopes" claim as array or "scope" as space-delimited string.
	if scopesRaw, ok := raw["scopes"]; ok {
		if arr, ok := scopesRaw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					claims.Scopes = append(claims.Scopes, s)
				}
			}
		}
	}
	if scopeStr, ok := raw["scope"].(string); ok {
		for _, s := range strings.Split(scopeStr, " ") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				claims.Scopes = append(claims.Scopes, trimmed)
			}
		}
	}

	return claims
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
	c.Abort()
}

// RequireScopes enforces that the request context contains all required scopes.
func RequireScopes(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authmw.ClaimsFromContext(c.Request.Context())
		if err != nil {
			unauthorized(c, "missing claims")
			return
		}

		if !claims.HasAllScopes(scopes...) {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient scopes"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// parseAndValidateJWT validates HS256 or RS256 JWT without external deps.
// ParseJWT is exported to allow gRPC interceptor reuse.
func ParseJWT(token, secret, publicKey, issuer string, audience []string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("invalid token header")
	}
	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, errors.New("invalid token header")
	}
	alg, ok := header["alg"].(string)
	if !ok {
		return nil, errors.New("unsupported alg")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, errors.New("invalid token claims")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("invalid token signature")
	}

	switch alg {
	case "HS256":
		if secret == "" {
			return nil, errors.New("hs256 secret not configured")
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(parts[0] + "." + parts[1]))
		expectedSig := mac.Sum(nil)
		if !hmac.Equal(sig, expectedSig) {
			return nil, errors.New("invalid token signature")
		}
	case "RS256":
		if publicKey == "" {
			return nil, errors.New("rs256 public key not configured")
		}
		if err := verifyRS256([]byte(parts[0]+"."+parts[1]), sig, publicKey); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported alg")
	}

	if expVal, ok := claims["exp"]; ok {
		switch v := expVal.(type) {
		case float64:
			if time.Now().Unix() > int64(v) {
				return nil, errors.New("token expired")
			}
		case json.Number:
			n, _ := v.Int64()
			if time.Now().Unix() > n {
				return nil, errors.New("token expired")
			}
		}
	}

	if issuer != "" {
		if iss, _ := claims["iss"].(string); iss != issuer {
			return nil, errors.New("invalid issuer")
		}
	}

	if len(audience) > 0 {
		if !validateAudience(claims["aud"], audience) {
			return nil, errors.New("invalid audience")
		}
	}

	return claims, nil
}

func verifyRS256(signedData, sig []byte, publicKey string) error {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil || block.Type != "PUBLIC KEY" && !strings.HasSuffix(block.Type, "RSA PUBLIC KEY") {
		return errors.New("invalid public key")
	}
	var pub *rsa.PublicKey
	var err error
	if block.Type == "PUBLIC KEY" {
		var p any
		p, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err == nil {
			pub, _ = p.(*rsa.PublicKey)
		}
	} else {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
	}
	if err != nil || pub == nil {
		return errors.New("invalid public key")
	}
	hash := sha256.Sum256(signedData)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sig); err != nil {
		return errors.New("invalid token signature")
	}
	return nil
}

func validateAudience(audClaim interface{}, expected []string) bool {
	expectedSet := make(map[string]struct{}, len(expected))
	for _, a := range expected {
		expectedSet[a] = struct{}{}
	}

	switch v := audClaim.(type) {
	case string:
		_, ok := expectedSet[v]
		return ok
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				if _, found := expectedSet[s]; found {
					return true
				}
			}
		}
	}
	return false
}
