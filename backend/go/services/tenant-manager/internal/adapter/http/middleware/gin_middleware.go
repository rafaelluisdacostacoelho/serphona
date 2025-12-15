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
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestID adds a request ID to each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Also inject into request context so downstream handlers that rely on Context values can read it.
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
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

		tenantID, _ := claims["tenant_id"].(string)
		userID, _ := claims["sub"].(string)

		if tenantID == "" || userID == "" {
			unauthorized(c, "Missing tenant_id or sub claim")
			return
		}

		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)

		c.Next()
	}
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
	c.Abort()
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
