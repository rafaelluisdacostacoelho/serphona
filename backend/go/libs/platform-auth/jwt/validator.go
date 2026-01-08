package jwt

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

const (
	// EnvJWTSecret is the default environment variable name used to load the JWT secret.
	EnvJWTSecret = "JWT_SECRET"
)

var jwtSecret string
var ensureSecretOnce sync.Once
var ensureSecretErr error
var validationCfgMu sync.RWMutex
var validationConfig = defaultValidationConfig()

// ValidationConfig controls validation rules beyond the secret.
type ValidationConfig struct {
	AllowedAlgs     []string
	Issuer          string
	Audience        string
	ServiceAudience string
	ClockSkew       time.Duration
	MaxTokenBytes   int
	JWKSURL         string
	JWKSCacheTTL    time.Duration
	AllowedKIDs     []string
	RequiredScopes  []string // Optional: enforce presence of specific scopes at validation time
}

func defaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		AllowedAlgs:   []string{"HS256"},
		ClockSkew:     30 * time.Second,
		MaxTokenBytes: 4096,
		JWKSCacheTTL:  5 * time.Minute,
	}
}

// SetSecret configures the JWT secret used for validation.
func SetSecret(secret string) {
	jwtSecret = secret
}

// SetValidationConfig overrides validation rules. Zero/empty values fall back to defaults.
func SetValidationConfig(cfg ValidationConfig) {
	validationCfgMu.Lock()
	defer validationCfgMu.Unlock()

	if len(cfg.AllowedAlgs) == 0 {
		cfg.AllowedAlgs = defaultValidationConfig().AllowedAlgs
	}
	if cfg.ClockSkew < 0 {
		cfg.ClockSkew = 0
	}
	if cfg.MaxTokenBytes < 0 {
		cfg.MaxTokenBytes = 0
	}
	if cfg.JWKSCacheTTL <= 0 {
		cfg.JWKSCacheTTL = defaultValidationConfig().JWKSCacheTTL
	}
	validationConfig = cfg
}

// GetValidationConfig returns a copy of the current validation config.
func GetValidationConfig() ValidationConfig {
	validationCfgMu.RLock()
	defer validationCfgMu.RUnlock()
	return validationConfig
}

// ResetValidationConfigForTests resets validation rules to defaults.
func ResetValidationConfigForTests() {
	SetValidationConfig(defaultValidationConfig())
}

// SetSecretFromEnv loads the JWT secret from EnvJWTSecret and returns an error if missing.
func SetSecretFromEnv() error {
	secret := os.Getenv(EnvJWTSecret)
	if secret == "" {
		return autherrors.ErrSecretNotConfigured
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

// EnsureSecretLoaded loads the secret once, preferring the already set value and falling back to EnvJWTSecret.
func EnsureSecretLoaded() error {
	ensureSecretOnce.Do(func() {
		if jwtSecret != "" {
			return
		}
		ensureSecretErr = SetSecretFromEnv()
	})
	return ensureSecretErr
}

// MustEnsureSecretLoaded panics when the secret cannot be loaded.
func MustEnsureSecretLoaded() {
	if err := EnsureSecretLoaded(); err != nil {
		panic(err)
	}
}

// GetSecret returns the configured JWT secret.
func GetSecret() string {
	return jwtSecret
}

// ValidateToken validates a JWT token and returns its claims using the configured secret.
func ValidateToken(tokenString string) (*types.Claims, error) {
	cfg := GetValidationConfig()
	if requiresSecret(cfg) {
		if err := EnsureSecretLoaded(); err != nil {
			return nil, err
		}
	} else {
		// Attempt to load the secret for optional HMAC fallback, but do not fail if absent.
		_ = EnsureSecretLoaded()
	}

	return ValidateTokenWithSecret(tokenString, jwtSecret)
}

func requiresSecret(cfg ValidationConfig) bool {
	if cfg.JWKSURL == "" {
		return true
	}

	for _, alg := range cfg.AllowedAlgs {
		if strings.HasPrefix(strings.ToUpper(alg), "HS") {
			return true
		}
	}

	return false
}

// ValidateTokenWithSecret validates a JWT token with a provided secret.
func ValidateTokenWithSecret(tokenString, secret string) (*types.Claims, error) {
	if tokenString == "" {
		return nil, autherrors.ErrMissingToken
	}
	cfg := GetValidationConfig()
	if cfg.MaxTokenBytes > 0 && len(tokenString) > cfg.MaxTokenBytes {
		return nil, autherrors.ErrTokenTooLarge
	}

	if secret == "" && cfg.JWKSURL == "" {
		return nil, autherrors.ErrSecretNotConfigured
	}

	parserOpts := []jwt.ParserOption{jwt.WithLeeway(cfg.ClockSkew)}
	if cfg.Issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(cfg.Issuer))
	}
	expectedAudiences := []string{}
	if cfg.Audience != "" {
		expectedAudiences = append(expectedAudiences, cfg.Audience)
	}
	if cfg.ServiceAudience != "" {
		expectedAudiences = append(expectedAudiences, cfg.ServiceAudience)
	}
	if len(expectedAudiences) == 1 {
		parserOpts = append(parserOpts, jwt.WithAudience(expectedAudiences[0]))
	}
	if len(cfg.AllowedAlgs) > 0 {
		parserOpts = append(parserOpts, jwt.WithValidMethods(cfg.AllowedAlgs))
	}

	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil {
			return nil, autherrors.ErrInvalidToken
		}

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
			if secret == "" {
				return nil, autherrors.ErrSecretNotConfigured
			}
			return []byte(secret), nil
		}

		kid, _ := token.Header["kid"].(string)
		if len(cfg.AllowedKIDs) > 0 && kid != "" && !isAllowedKID(cfg.AllowedKIDs, kid) {
			return nil, autherrors.ErrInvalidKeyID
		}

		if cfg.JWKSURL == "" {
			return nil, autherrors.ErrAuthConfigMissing
		}

		key, err := getJWKSPublicKey(cfg, kid)
		if err != nil {
			return nil, err
		}

		return key, nil
	}, parserOpts...)

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "signing method") {
			return nil, autherrors.ErrInvalidAlgorithm
		}
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, autherrors.ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenInvalidAudience):
			return nil, autherrors.ErrInvalidAudience
		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			return nil, autherrors.ErrInvalidIssuer
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, autherrors.ErrInvalidToken
		case errors.Is(err, autherrors.ErrInvalidKeyID):
			return nil, autherrors.ErrInvalidKeyID
		case errors.Is(err, autherrors.ErrJWKSFetchFailed):
			return nil, autherrors.ErrJWKSFetchFailed
		case errors.Is(err, autherrors.ErrAuthConfigMissing):
			return nil, autherrors.ErrAuthConfigMissing
		case strings.Contains(strings.ToLower(err.Error()), "unexpected signing method"):
			return nil, autherrors.ErrInvalidAlgorithm
		default:
			return nil, autherrors.ErrInvalidToken
		}
	}

	if claims, ok := token.Claims.(*types.Claims); ok && token.Valid {
		if err := validateClaimsDetails(claims, cfg); err != nil {
			return nil, err
		}
		// Optionally enforce required scopes at validation time
		if len(cfg.RequiredScopes) > 0 {
			if !claims.HasAllScopes(cfg.RequiredScopes...) {
				return nil, autherrors.ErrInsufficientPermissions
			}
		}
		return claims, nil
	}

	return nil, autherrors.ErrInvalidToken
}

func isAllowedKID(allowed []string, kid string) bool {
	for _, k := range allowed {
		if k == kid {
			return true
		}
	}
	return false
}

type jwksState struct {
	mu        sync.RWMutex
	cache     map[string]interface{}
	expiresAt time.Time
	url       string
	lastErr   error
}

var globalJWKS = jwksState{}

// ResetJWKSCacheForTests clears JWKS cache and errors for test isolation.
func ResetJWKSCacheForTests() {
	globalJWKS.mu.Lock()
	defer globalJWKS.mu.Unlock()

	globalJWKS.cache = nil
	globalJWKS.expiresAt = time.Time{}
	globalJWKS.url = ""
	globalJWKS.lastErr = nil
}

func getJWKSPublicKey(cfg ValidationConfig, kid string) (interface{}, error) {
	now := time.Now()
	globalJWKS.mu.RLock()
	if cfg.JWKSURL == globalJWKS.url && now.Before(globalJWKS.expiresAt) && globalJWKS.cache != nil {
		if kid == "" && len(globalJWKS.cache) == 1 {
			for _, key := range globalJWKS.cache {
				globalJWKS.mu.RUnlock()
				return key, nil
			}
		}
		if key, ok := globalJWKS.cache[kid]; ok && key != nil {
			globalJWKS.mu.RUnlock()
			return key, nil
		}
	}
	globalJWKS.mu.RUnlock()

	globalJWKS.mu.Lock()
	defer globalJWKS.mu.Unlock()

	if cfg.JWKSURL != globalJWKS.url || now.After(globalJWKS.expiresAt) || globalJWKS.cache == nil {
		keys, err := fetchJWKS(cfg)
		if err != nil {
			globalJWKS.lastErr = err
			return nil, autherrors.ErrJWKSFetchFailed
		}

		globalJWKS.cache = keys
		globalJWKS.url = cfg.JWKSURL
		globalJWKS.expiresAt = now.Add(cfg.JWKSCacheTTL)
		globalJWKS.lastErr = nil
	}

	if kid == "" && len(globalJWKS.cache) == 1 {
		for _, key := range globalJWKS.cache {
			return key, nil
		}
	}

	if key, ok := globalJWKS.cache[kid]; ok && key != nil {
		return key, nil
	}

	return nil, autherrors.ErrInvalidKeyID
}

func fetchJWKS(cfg ValidationConfig) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(cfg.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var body jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	result := make(map[string]interface{})
	for _, key := range body.Keys {
		if key.Kid == "" {
			continue
		}

		pubKey, err := key.toPublicKey()
		if err != nil {
			return nil, err
		}
		result[key.Kid] = pubKey
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("jwks contains no usable keys")
	}

	return result, nil
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (k jwkKey) toPublicKey() (interface{}, error) {
	switch k.Kty {
	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("decode modulus: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("decode exponent: %w", err)
		}

		eInt := 0
		for _, b := range eBytes {
			eInt = (eInt << 8) + int(b)
		}
		if eInt == 0 {
			return nil, fmt.Errorf("invalid exponent")
		}

		return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: eInt}, nil
	default:
		return nil, fmt.Errorf("unsupported jwk kty: %s", k.Kty)
	}
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

func validateClaimsDetails(claims *types.Claims, cfg ValidationConfig) error {
	if claims == nil {
		return autherrors.ErrInvalidToken
	}

	tenant := strings.TrimSpace(claims.TenantID)
	if tenant == "" {
		return autherrors.ErrInvalidToken
	}
	if tenant != "platform" {
		if _, err := uuid.Parse(tenant); err != nil {
			return autherrors.ErrInvalidToken
		}
	}

	if err := validateAudience(claims, cfg); err != nil {
		return err
	}

	if strings.TrimSpace(claims.Service) != "" {
		if len(claims.Scopes) == 0 {
			return autherrors.ErrInsufficientPermissions
		}
		return nil
	}

	if _, err := uuid.Parse(strings.TrimSpace(claims.UserID)); err != nil {
		return autherrors.ErrInvalidToken
	}

	switch claims.Role {
	case "user", "admin", "superadmin":
		return nil
	default:
		return autherrors.ErrInvalidRole
	}
}

func validateAudience(claims *types.Claims, cfg ValidationConfig) error {
	if claims == nil {
		return autherrors.ErrInvalidToken
	}

	expectedAudience := cfg.Audience
	if strings.TrimSpace(claims.Service) != "" && cfg.ServiceAudience != "" {
		expectedAudience = cfg.ServiceAudience
	}

	if expectedAudience == "" {
		return nil
	}

	if len(claims.Audience) == 0 {
		return autherrors.ErrInvalidAudience
	}

	if !audienceContains(claims.Audience, expectedAudience) {
		return autherrors.ErrInvalidAudience
	}

	return nil
}

func audienceContains(audiences jwt.ClaimStrings, target string) bool {
	for _, aud := range audiences {
		if aud == target {
			return true
		}
	}
	return false
}

// ResetSecretForTesting resets the JWT secret and the sync.Once instance for testing purposes.
func ResetSecretForTesting() {
	jwtSecret = ""
	ensureSecretOnce = sync.Once{}
	ensureSecretErr = nil
}
