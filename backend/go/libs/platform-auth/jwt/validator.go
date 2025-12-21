package jwt

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

const (
	// EnvJWTSecret is the default environment variable name used to load the JWT secret.
	EnvJWTSecret = "JWT_SECRET"
)

var jwtSecret string

// SetSecret configures the JWT secret used for validation.
func SetSecret(secret string) {
	jwtSecret = secret
}

// SetSecretFromEnv loads the JWT secret from EnvJWTSecret and returns an error if missing.
func SetSecretFromEnv() error {
	secret := os.Getenv(EnvJWTSecret)
	if secret == "" {
		return fmt.Errorf("environment variable %s not set", EnvJWTSecret)
	}
	SetSecret(secret)
	return nil
}

// MustSetSecretFromEnv loads the JWT secret from EnvJWTSecret and panics if missing.
func MustSetSecretFromEnv() {
	if err := SetSecretFromEnv(); err != nil {
		panic(err)
	}
}

// GetSecret returns the configured JWT secret.
func GetSecret() string {
	return jwtSecret
}

// ValidateToken validates a JWT token and returns its claims using the configured secret.
func ValidateToken(tokenString string) (*types.Claims, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	return ValidateTokenWithSecret(tokenString, jwtSecret)
}

// ValidateTokenWithSecret validates a JWT token with a provided secret.
func ValidateTokenWithSecret(tokenString, secret string) (*types.Claims, error) {
	if tokenString == "" {
		return nil, autherrors.ErrMissingToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, autherrors.ErrTokenExpired
		}
		return nil, autherrors.ErrInvalidToken
	}

	if claims, ok := token.Claims.(*types.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, autherrors.ErrInvalidToken
}

// ExtractTokenFromHeader extracts the token from the Authorization header (expects "Bearer <token>").
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", autherrors.ErrMissingToken
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", autherrors.ErrInvalidToken
	}

	token := parts[1]
	if token == "" {
		return "", autherrors.ErrMissingToken
	}

	return token, nil
}

// ValidateTokenFromHeader validates a token extracted from the Authorization header.
func ValidateTokenFromHeader(authHeader string) (*types.Claims, error) {
	token, err := ExtractTokenFromHeader(authHeader)
	if err != nil {
		return nil, err
	}

	return ValidateToken(token)
}
