package jwt_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	autherrors "github.com/serphona/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/serphona/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/serphona/serphona/backend/go/libs/platform-auth/types"
)

const testSecret = "test-secret-key-32-characters-minimum!"

func signedToken(t *testing.T, exp time.Time, role string) string {
	t.Helper()

	claims := types.Claims{
		UserID:    "11111111-1111-1111-1111-111111111111",
		Email:     "user@example.com",
		Name:      "Test User",
		Role:      role,
		TenantID:  "22222222-2222-2222-2222-222222222222",
		SessionID: "33333333-3333-3333-3333-333333333333",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return token
}

func TestValidateTokenSuccess(t *testing.T) {
	authjwt.SetSecret(testSecret)

	token := signedToken(t, time.Now().Add(time.Hour), "admin")

	claims, err := authjwt.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected token to be valid, got error: %v", err)
	}

	if claims.Role != "admin" || claims.TenantID == "" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestValidateTokenExpired(t *testing.T) {
	authjwt.SetSecret(testSecret)

	token := signedToken(t, time.Now().Add(-time.Hour), "user")

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrTokenExpired {
		t.Fatalf("expected expired token error, got: %v", err)
	}
}

func TestValidateTokenInvalidSignature(t *testing.T) {
	authjwt.SetSecret(testSecret)

	token := signedToken(t, time.Now().Add(time.Hour), "user")

	if _, err := authjwt.ValidateTokenWithSecret(token, "wrong-secret"); err != autherrors.ErrInvalidToken {
		t.Fatalf("expected invalid token error, got: %v", err)
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	token := "abc123"

	extracted, err := authjwt.ExtractTokenFromHeader("Bearer " + token)
	if err != nil {
		t.Fatalf("expected token extraction to succeed, got: %v", err)
	}
	if extracted != token {
		t.Fatalf("expected token %s, got %s", token, extracted)
	}

	if _, err := authjwt.ExtractTokenFromHeader(""); err != autherrors.ErrMissingToken {
		t.Fatalf("expected missing token error, got: %v", err)
	}

	if _, err := authjwt.ExtractTokenFromHeader("InvalidHeader"); err != autherrors.ErrInvalidToken {
		t.Fatalf("expected invalid token error, got: %v", err)
	}
}

func TestValidateTokenFromHeader(t *testing.T) {
	authjwt.SetSecret(testSecret)

	token := signedToken(t, time.Now().Add(time.Hour), "user")
	header := "Bearer " + token

	claims, err := authjwt.ValidateTokenFromHeader(header)
	if err != nil {
		t.Fatalf("expected validation to succeed, got: %v", err)
	}
	if claims.UserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected user id: %s", claims.UserID)
	}
}

func TestSetSecretFromEnvSuccess(t *testing.T) {
	t.Setenv(authjwt.EnvJWTSecret, "env-secret-value")

	if err := authjwt.SetSecretFromEnv(); err != nil {
		t.Fatalf("expected secret to load from env, got: %v", err)
	}

	if authjwt.GetSecret() != "env-secret-value" {
		t.Fatalf("unexpected secret loaded: %s", authjwt.GetSecret())
	}
}

func TestSetSecretFromEnvMissing(t *testing.T) {
	t.Setenv(authjwt.EnvJWTSecret, "")

	if err := authjwt.SetSecretFromEnv(); err == nil {
		t.Fatalf("expected error when env secret is missing")
	}
}
