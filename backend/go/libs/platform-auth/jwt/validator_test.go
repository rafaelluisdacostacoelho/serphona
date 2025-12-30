package jwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

const testSecret = "test-secret-key-32-characters-minimum!"

func resetSecretForTests() {
	authjwt.SetSecret("")
	// reset the once so EnsureSecretLoaded can run again
	authjwt.ResetSecretOnceForTests()
	authjwt.ResetJWKSCacheForTests()
	authjwt.ResetValidationConfig()
}

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

func signedServiceToken(t *testing.T, exp time.Time, service string, scopes []string, tenant string) string {
	t.Helper()

	claims := types.Claims{
		Service:  service,
		TenantID: tenant,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign service token: %v", err)
	}

	return token
}

func signedRSAToken(t *testing.T, privateKey *rsa.PrivateKey, kid string, exp time.Time) string {
	t.Helper()

	claims := types.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "22222222-2222-2222-2222-222222222222",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign rsa token: %v", err)
	}

	return signed
}

func rsaKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	return key
}

func jwksFromPublicKey(t *testing.T, pub *rsa.PublicKey, kid string) string {
	t.Helper()

	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	body := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"kid": kid,
				"alg": "RS256",
				"use": "sig",
				"n":   n,
				"e":   e,
			},
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal jwks: %v", err)
	}

	return string(encoded)
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

func TestValidateTokenInvalidIssuer(t *testing.T) {
	authjwt.SetSecret(testSecret)
	authjwt.SetValidationConfig(authjwt.ValidationConfig{Issuer: "expected-issuer"})
	defer authjwt.ResetValidationConfig()

	claims := types.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "22222222-2222-2222-2222-222222222222",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "other-issuer",
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidIssuer {
		t.Fatalf("expected invalid issuer, got %v", err)
	}
}

func TestValidateTokenInvalidAudience(t *testing.T) {
	authjwt.SetSecret(testSecret)
	authjwt.SetValidationConfig(authjwt.ValidationConfig{Audience: "expected-aud"})
	defer authjwt.ResetValidationConfig()

	claims := types.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "22222222-2222-2222-2222-222222222222",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Audience:  jwt.ClaimStrings{"other-aud"},
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidAudience {
		t.Fatalf("expected invalid audience, got %v", err)
	}
}

func TestValidateTokenInvalidAlgorithm(t *testing.T) {
	authjwt.SetSecret(testSecret)
	authjwt.SetValidationConfig(authjwt.ValidationConfig{AllowedAlgs: []string{"HS512"}})
	defer authjwt.ResetValidationConfig()

	token := signedToken(t, time.Now().Add(time.Hour), "user")
	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidAlgorithm {
		t.Fatalf("expected invalid algorithm error, got %v", err)
	}
}

func TestValidateTokenMaxSize(t *testing.T) {
	authjwt.SetSecret(testSecret)
	authjwt.SetValidationConfig(authjwt.ValidationConfig{MaxTokenBytes: 10})
	defer authjwt.ResetValidationConfig()

	if _, err := authjwt.ValidateToken(strings.Repeat("a", 11)); err != autherrors.ErrTokenTooLarge {
		t.Fatalf("expected token too large error, got %v", err)
	}
}

func TestValidateTokenRequiredScopes(t *testing.T) {
	authjwt.SetSecret(testSecret)
	authjwt.SetValidationConfig(authjwt.ValidationConfig{RequiredScopes: []string{"read:invoices", "write:invoices"}})
	defer authjwt.ResetValidationConfig()

	// Token without scopes should fail
	tokenNoScopes := signedToken(t, time.Now().Add(time.Hour), "user")
	if _, err := authjwt.ValidateToken(tokenNoScopes); err != autherrors.ErrInsufficientPermissions {
		t.Fatalf("expected insufficient permissions error, got %v", err)
	}

	// Token with partial scopes should fail
	claims := types.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "22222222-2222-2222-2222-222222222222",
		Scopes:   []string{"read:invoices"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tokenPartial, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if _, err := authjwt.ValidateToken(tokenPartial); err != autherrors.ErrInsufficientPermissions {
		t.Fatalf("expected insufficient permissions error, got %v", err)
	}

	// Token with all required scopes should succeed
	claims.Scopes = []string{"read:invoices", "write:invoices", "admin:extra"}
	tokenValid, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if _, err := authjwt.ValidateToken(tokenValid); err != nil {
		t.Fatalf("expected token with all scopes to be valid, got error: %v", err)
	}
}

func TestValidateTokenNotYetValid(t *testing.T) {
	authjwt.SetSecret(testSecret)

	claims := types.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "22222222-2222-2222-2222-222222222222",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidToken {
		t.Fatalf("expected not-before violation to map to invalid token, got %v", err)
	}
}

func TestValidateTokenMissingTenantFails(t *testing.T) {
	authjwt.SetSecret(testSecret)

	claims := types.Claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "user@example.com",
		Name:   "Test User",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidToken {
		t.Fatalf("expected token without tenant to fail, got %v", err)
	}
}

func TestValidateServiceTokenSuccess(t *testing.T) {
	authjwt.SetSecret(testSecret)

	token := signedServiceToken(t, time.Now().Add(time.Hour), "tenant-manager", []string{"tenant:read"}, "platform")

	claims, err := authjwt.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected service token to be valid, got %v", err)
	}

	if claims.Service != "tenant-manager" {
		t.Fatalf("unexpected service claim: %+v", claims.Service)
	}
	if claims.TenantID != "platform" {
		t.Fatalf("expected platform tenant, got %s", claims.TenantID)
	}
}

func TestValidateServiceTokenRequiresScopes(t *testing.T) {
	authjwt.SetSecret(testSecret)

	token := signedServiceToken(t, time.Now().Add(time.Hour), "billing-service", nil, "22222222-2222-2222-2222-222222222222")

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInsufficientPermissions {
		t.Fatalf("expected service token without scopes to fail with insufficient permissions, got %v", err)
	}
}

func TestValidateTokenInvalidRole(t *testing.T) {
	authjwt.SetSecret(testSecret)

	claims := types.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "reader",
		TenantID: "22222222-2222-2222-2222-222222222222",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidRole {
		t.Fatalf("expected invalid role error, got %v", err)
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
	resetSecretForTests()
	t.Setenv(authjwt.EnvJWTSecret, "env-secret-value")

	if err := authjwt.SetSecretFromEnv(); err != nil {
		t.Fatalf("expected secret to load from env, got: %v", err)
	}

	if authjwt.GetSecret() != "env-secret-value" {
		t.Fatalf("unexpected secret loaded: %s", authjwt.GetSecret())
	}
}

func TestSetSecretFromEnvMissing(t *testing.T) {
	resetSecretForTests()
	t.Setenv(authjwt.EnvJWTSecret, "")

	if err := authjwt.SetSecretFromEnv(); err == nil {
		t.Fatalf("expected error when env secret is missing")
	}
}

func TestEnsureSecretLoadedMissing(t *testing.T) {
	resetSecretForTests()
	t.Setenv(authjwt.EnvJWTSecret, "")

	if err := authjwt.EnsureSecretLoaded(); err != autherrors.ErrSecretNotConfigured {
		t.Fatalf("expected ErrSecretNotConfigured, got %v", err)
	}
}

func TestValidateTokenDisallowedKID(t *testing.T) {
	authjwt.ResetJWKSCacheForTests()

	priv := rsaKeyPair(t)

	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(jwksFromPublicKey(t, &priv.PublicKey, "kid-allowed")))
	}))
	defer ts.Close()

	authjwt.SetSecret("")
	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs: []string{"RS256"},
		JWKSURL:     ts.URL,
		AllowedKIDs: []string{"kid-allowed"},
	})
	t.Cleanup(func() {
		authjwt.ResetValidationConfig()
		authjwt.ResetJWKSCacheForTests()
	})

	token := signedRSAToken(t, priv, "kid-blocked", time.Now().Add(time.Hour))

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidKeyID {
		t.Fatalf("expected invalid key id error, got %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected AllowedKIDs check to short-circuit JWKS fetch, got %d calls", calls.Load())
	}
}

func TestValidateTokenJWKSRotation(t *testing.T) {
	authjwt.ResetJWKSCacheForTests()

	priv1 := rsaKeyPair(t)
	priv2 := rsaKeyPair(t)

	var jwksBody atomic.Value
	jwksBody.Store(jwksFromPublicKey(t, &priv1.PublicKey, "kid-1"))

	var fetches atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		_, _ = w.Write([]byte(jwksBody.Load().(string)))
	}))
	defer ts.Close()

	authjwt.SetSecret("")
	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs:  []string{"RS256"},
		JWKSURL:      ts.URL,
		JWKSCacheTTL: 25 * time.Millisecond,
		AllowedKIDs:  []string{"kid-1", "kid-2"},
	})
	t.Cleanup(func() {
		authjwt.ResetValidationConfig()
		authjwt.ResetJWKSCacheForTests()
	})

	firstToken := signedRSAToken(t, priv1, "kid-1", time.Now().Add(time.Hour))
	if _, err := authjwt.ValidateToken(firstToken); err != nil {
		t.Fatalf("expected first token to validate, got %v", err)
	}
	if got := fetches.Load(); got != 1 {
		t.Fatalf("expected single JWKS fetch, got %d", got)
	}

	jwksBody.Store(jwksFromPublicKey(t, &priv2.PublicKey, "kid-2"))
	secondToken := signedRSAToken(t, priv2, "kid-2", time.Now().Add(time.Hour))
	time.Sleep(30 * time.Millisecond)

	if _, err := authjwt.ValidateToken(secondToken); err != nil {
		t.Fatalf("expected rotated key to validate, got %v", err)
	}
	if got := fetches.Load(); got < 2 {
		t.Fatalf("expected JWKS re-fetch after cache expiry, got %d", got)
	}
}

func TestEnsureSecretLoadedFromEnv(t *testing.T) {
	resetSecretForTests()
	t.Setenv(authjwt.EnvJWTSecret, "env-secret-value")

	if err := authjwt.EnsureSecretLoaded(); err != nil {
		t.Fatalf("expected secret to load, got %v", err)
	}
	if authjwt.GetSecret() != "env-secret-value" {
		t.Fatalf("expected secret to be set, got %s", authjwt.GetSecret())
	}
}

func TestValidateTokenWithJWKS(t *testing.T) {
	resetSecretForTests()

	priv := rsaKeyPair(t)
	pub := &priv.PublicKey
	kid := "kid-1"
	jwksBody := jwksFromPublicKey(t, pub, kid)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, jwksBody)
	}))
	defer jwksServer.Close()

	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs:  []string{"RS256"},
		JWKSURL:      jwksServer.URL,
		JWKSCacheTTL: time.Minute,
		AllowedKIDs:  []string{kid},
	})

	token := signedRSAToken(t, priv, kid, time.Now().Add(time.Hour))

	claims, err := authjwt.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected JWKS validation to succeed, got %v", err)
	}

	if claims.Role != "user" || claims.TenantID == "" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestValidateTokenJWKSRejectsUnallowedKid(t *testing.T) {
	resetSecretForTests()

	priv := rsaKeyPair(t)
	pub := &priv.PublicKey
	kid := "kid-unallowed"
	jwksBody := jwksFromPublicKey(t, pub, kid)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, jwksBody)
	}))
	defer jwksServer.Close()

	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs:  []string{"RS256"},
		JWKSURL:      jwksServer.URL,
		JWKSCacheTTL: time.Minute,
		AllowedKIDs:  []string{"kid-allowed"},
	})

	token := signedRSAToken(t, priv, kid, time.Now().Add(time.Hour))

	if _, err := authjwt.ValidateToken(token); err != autherrors.ErrInvalidKeyID {
		t.Fatalf("expected invalid key id error, got %v", err)
	}
}

func TestValidateTokenJWKSRefreshOnRotation(t *testing.T) {
	resetSecretForTests()

	key1 := rsaKeyPair(t)
	key2 := rsaKeyPair(t)

	kid1 := "kid-rotate-1"
	kid2 := "kid-rotate-2"

	jwksBody1 := jwksFromPublicKey(t, &key1.PublicKey, kid1)
	jwksBody2 := jwksFromPublicKey(t, &key2.PublicKey, kid2)

	var currentJWKS atomic.Value
	currentJWKS.Store(jwksBody1)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, currentJWKS.Load().(string))
	}))
	defer jwksServer.Close()

	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs:  []string{"RS256"},
		JWKSURL:      jwksServer.URL,
		JWKSCacheTTL: 50 * time.Millisecond,
		AllowedKIDs:  []string{kid1, kid2},
	})

	token1 := signedRSAToken(t, key1, kid1, time.Now().Add(time.Hour))
	if _, err := authjwt.ValidateToken(token1); err != nil {
		t.Fatalf("expected token1 to validate before rotation, got %v", err)
	}

	currentJWKS.Store(jwksBody2)
	time.Sleep(60 * time.Millisecond)

	if _, err := authjwt.ValidateToken(token1); err != autherrors.ErrInvalidKeyID {
		t.Fatalf("expected token1 to fail after rotation with invalid key id, got %v", err)
	}

	token2 := signedRSAToken(t, key2, kid2, time.Now().Add(time.Hour))
	if _, err := authjwt.ValidateToken(token2); err != nil {
		t.Fatalf("expected token2 to validate after rotation, got %v", err)
	}
}

func TestValidateTokenPrefersHMACWhenAlgIsHS256(t *testing.T) {
	resetSecretForTests()
	authjwt.SetSecret(testSecret)

	// Configure JWKS to ensure HMAC path does not depend on network.
	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs: []string{"HS256", "RS256"},
		JWKSURL:     "http://127.0.0.1:0/unused",
	})

	token := signedToken(t, time.Now().Add(time.Hour), "user")

	if _, err := authjwt.ValidateToken(token); err != nil {
		t.Fatalf("expected HMAC token to validate without JWKS, got %v", err)
	}
}
