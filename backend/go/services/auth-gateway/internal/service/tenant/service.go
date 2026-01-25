package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// Service handles tenant operations
type Service struct {
	tenantAPIURL    string
	httpClient      *http.Client
	serviceName     string
	serviceInstance string
	serviceToken    string
	audience        string
}

// NewService creates a new tenant service.
func NewService(tenantAPIURL, serviceName, serviceInstance, serviceToken, audience string) *Service {
	return &Service{
		tenantAPIURL: tenantAPIURL,
		httpClient: &http.Client{
			Transport: authclient.WithServiceTransport(nil, serviceName, serviceInstance),
		},
		serviceName:     serviceName,
		serviceInstance: serviceInstance,
		serviceToken:    serviceToken,
		audience:        audience,
	}
}

// CreateTenant creates a new tenant via tenant-manager service
func (s *Service) CreateTenant(ctx context.Context, name, email, plan, billingEmail, phone string) (uuid.UUID, error) {
	payload := map[string]string{
		"name":          strings.TrimSpace(name),
		"email":         strings.TrimSpace(email),
		"plan":          strings.TrimSpace(plan),
		"billing_email": strings.TrimSpace(billingEmail),
		"phone":         strings.TrimSpace(phone),
	}

	if payload["plan"] == "" {
		payload["plan"] = "starter"
	}

	if !isAllowedPlan(payload["plan"]) {
		return uuid.Nil, fmt.Errorf("invalid plan %q (allowed: starter, professional, enterprise)", payload["plan"])
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal tenant payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/v1/tenants", strings.TrimRight(s.tenantAPIURL, "/")), bytes.NewReader(body))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create tenant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if s.serviceToken != "" && req.Header.Get("Authorization") == "" {
		if err := validateAudience(s.serviceToken, s.audience); err != nil {
			return uuid.Nil, err
		}
		req.Header.Set("Authorization", "Bearer "+s.serviceToken)
	}

	// Adiciona validação do TenantID no contexto
	if tenantID, err := authmw.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		req.Header = authmw.EnsureTenantHeader(req.Header, tenantID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("tenant-manager request failed: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return uuid.Nil, fmt.Errorf("tenant-manager returned status %d: %s", resp.StatusCode, describeError(resp))
	}
	defer resp.Body.Close()

	// tenant-manager wraps successful responses in an envelope {"data": {...}}
	var parsed struct {
		ID   string `json:"id"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode tenant-manager response: %w", err)
	}

	id := parsed.ID
	if id == "" {
		id = parsed.Data.ID
	}
	if id == "" {
		return uuid.Nil, fmt.Errorf("invalid tenant id from tenant-manager: empty id")
	}

	createdID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant id from tenant-manager: %w", err)
	}

	return createdID, nil
}

func isAllowedPlan(plan string) bool {
	switch plan {
	case "starter", "professional", "enterprise":
		return true
	default:
		return false
	}
}

func describeError(resp *http.Response) string {
	if resp == nil {
		return "no response"
	}
	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		if payload.Error.Code != "" || payload.Error.Message != "" {
			return strings.TrimSpace(fmt.Sprintf("%s %s", payload.Error.Code, payload.Error.Message))
		}
	}
	return strings.TrimSpace(string(body))
}

func validateAudience(token, expected string) error {
	if expected == "" {
		return nil
	}

	// Allow opaque tokens (non-JWT) when no audience claim can be inspected.
	if strings.Count(token, ".") != 2 {
		return nil
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := &jwt.RegisteredClaims{}

	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		return nil
	}

	for _, aud := range claims.Audience {
		if aud == expected {
			return nil
		}
	}

	return fmt.Errorf("service token audience mismatch")
}
