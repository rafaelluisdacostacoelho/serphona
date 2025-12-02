package jwt

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	autherrors "github.com/serphona/serphona/backend/go/libs/platform-auth/errors"
	"github.com/serphona/serphona/backend/go/libs/platform-auth/types"
)

var jwtSecret string

// SetSecret configura o secret JWT para validação
func SetSecret(secret string) {
	jwtSecret = secret
}

// GetSecret retorna o secret JWT configurado
func GetSecret() string {
	return jwtSecret
}

// ValidateToken valida um token JWT e retorna as claims
func ValidateToken(tokenString string) (*types.Claims, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	return ValidateTokenWithSecret(tokenString, jwtSecret)
}

// ValidateTokenWithSecret valida um token JWT com um secret específico
func ValidateTokenWithSecret(tokenString, secret string) (*types.Claims, error) {
	if tokenString == "" {
		return nil, autherrors.ErrMissingToken
	}

	// Parse e valida o token
	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verifica o método de assinatura
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		// Verifica se o token expirou
		if strings.Contains(err.Error(), "token is expired") {
			return nil, autherrors.ErrTokenExpired
		}
		return nil, autherrors.ErrInvalidToken
	}

	// Extrai as claims
	if claims, ok := token.Claims.(*types.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, autherrors.ErrInvalidToken
}

// ExtractTokenFromHeader extrai o token do header Authorization
// Espera formato: "Bearer <token>"
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", autherrors.ErrMissingToken
	}

	// Remove "Bearer " do início
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

// ValidateTokenFromHeader valida um token extraído do header Authorization
func ValidateTokenFromHeader(authHeader string) (*types.Claims, error) {
	token, err := ExtractTokenFromHeader(authHeader)
	if err != nil {
		return nil, err
	}

	return ValidateToken(token)
}
