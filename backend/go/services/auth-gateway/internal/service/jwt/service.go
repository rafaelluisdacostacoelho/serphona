package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	platformauth "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims is an alias to the platform-auth claims to keep compatibility with downstream middleware.
type Claims = platformauth.Claims

// Service handles JWT token generation and validation
type Service struct {
	secretKey            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewService creates a new JWT service
func NewService(secretKey string, accessTokenDuration, refreshTokenDuration time.Duration) *Service {
	return &Service{
		secretKey:            []byte(secretKey),
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// defaultScopes returns coarse-grained scopes based on role; keeps ingestion/tool flows unblocked in dev.
func defaultScopes(role string) []string {
	scopes := []string{"tools:read", "tools:write", "tools:execute"}
	if role == "admin" || role == "superadmin" {
		return append(scopes, "admin:read", "admin:write")
	}
	return scopes
}

// GenerateAccessToken generates a new access token using platform-auth compatible claims.
func (s *Service) GenerateAccessToken(userID, tenantID uuid.UUID, email, name, role string) (string, error) {
	claims := Claims{
		UserID:   userID.String(),
		Email:    email,
		Name:     name,
		Role:     role,
		TenantID: tenantID.String(),
		Scopes:   defaultScopes(role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "serphona",
			Audience:  []string{"serphona-services"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateRefreshToken generates a new refresh token
func (s *Service) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTokenDuration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "serphona-auth",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateAccessToken validates an access token and returns the claims
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (s *Service) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrExpiredToken
		}
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}

// JWTService defines the interface for JWT operations
// This allows for easier mocking and testing.
type JWTService interface {
	ValidateAccessToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(tokenString string) (uuid.UUID, error)
	GenerateAccessToken(userID, tenantID uuid.UUID, email, name, role string) (string, error)
	GenerateRefreshToken(userID uuid.UUID) (string, error)
}

// Ensure Service implements JWTService
var _ JWTService = (*Service)(nil)
