package tenant

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// Service handles tenant operations
type Service struct {
	tenantAPIURL string
	httpClient   *http.Client
}

// NewService creates a new tenant service
func NewService(tenantAPIURL string) *Service {
	return &Service{
		tenantAPIURL: tenantAPIURL,
		httpClient:   &http.Client{},
	}
}

// CreateTenant creates a new tenant via tenant-manager service
func (s *Service) CreateTenant(ctx context.Context, name string) (uuid.UUID, error) {
	// For now, we'll generate a UUID locally
	// In production, this would call the tenant-manager service
	tenantID := uuid.New()

	// TODO: Call tenant-manager API to create tenant
	// Example:
	// req := TenantCreateRequest{Name: name}
	// resp, err := s.httpClient.Post(s.tenantAPIURL+"/tenants", "application/json", body)

	fmt.Printf("Tenant created with ID: %s (local generation, integrate with tenant-manager later)\n", tenantID)

	return tenantID, nil
}
