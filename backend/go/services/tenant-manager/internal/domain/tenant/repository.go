// Package tenant contains the tenant domain model and business logic.
package tenant

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for tenant persistence.
// This is a port in hexagonal architecture - implementations are adapters.
type Repository interface {
	// Create persists a new tenant.
	Create(ctx context.Context, tenant *Tenant) error

	// GetByID retrieves a tenant by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)

	// GetBySlug retrieves a tenant by its slug.
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)

	// GetByEmail retrieves a tenant by its email.
	GetByEmail(ctx context.Context, email string) (*Tenant, error)

	// Update updates an existing tenant.
	Update(ctx context.Context, tenant *Tenant) error

	// Delete soft-deletes a tenant.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves tenants with pagination and filtering.
	List(ctx context.Context, filter ListFilter) (*ListResult, error)

	// UpdateSettings updates only the tenant settings.
	UpdateSettings(ctx context.Context, id uuid.UUID, settings Settings) error

	// GetQuota retrieves the quota for a tenant.
	GetQuota(ctx context.Context, tenantID uuid.UUID) (*Quota, error)

	// UpdateQuota updates the quota for a tenant.
	UpdateQuota(ctx context.Context, quota *Quota) error

	// IncrementUsage increments usage counters for a tenant.
	IncrementUsage(ctx context.Context, tenantID uuid.UUID, calls, minutes int) error

	// ExistsBySlug checks if a tenant with the given slug exists.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// ExistsByEmail checks if a tenant with the given email exists.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// ListFilter contains filter options for listing tenants.
type ListFilter struct {
	Status     *Status `json:"status,omitempty"`
	Plan       *Plan   `json:"plan,omitempty"`
	Search     string  `json:"search,omitempty"` // Search in name, email
	PageSize   int     `json:"page_size"`
	PageNumber int     `json:"page_number"`
	SortBy     string  `json:"sort_by"`
	SortOrder  string  `json:"sort_order"` // asc, desc
}

// ListResult contains the result of a list operation.
type ListResult struct {
	Tenants    []*Tenant `json:"tenants"`
	Total      int64     `json:"total"`
	PageSize   int       `json:"page_size"`
	PageNumber int       `json:"page_number"`
	TotalPages int       `json:"total_pages"`
}

// Cache defines the interface for tenant caching.
type Cache interface {
	// Get retrieves a tenant from cache.
	Get(ctx context.Context, key string) (*Tenant, error)

	// Set stores a tenant in cache.
	Set(ctx context.Context, key string, tenant *Tenant) error

	// Delete removes a tenant from cache.
	Delete(ctx context.Context, key string) error

	// GetSettings retrieves settings from cache.
	GetSettings(ctx context.Context, tenantID uuid.UUID) (*Settings, error)

	// SetSettings stores settings in cache.
	SetSettings(ctx context.Context, tenantID uuid.UUID, settings *Settings) error

	// Invalidate removes all cached data for a tenant.
	Invalidate(ctx context.Context, tenantID uuid.UUID) error
}

// EventPublisher defines the interface for publishing tenant events.
type EventPublisher interface {
	// PublishCreated publishes a tenant created event.
	PublishCreated(ctx context.Context, tenant *Tenant) error

	// PublishUpdated publishes a tenant updated event.
	PublishUpdated(ctx context.Context, tenant *Tenant) error

	// PublishDeleted publishes a tenant deleted event.
	PublishDeleted(ctx context.Context, tenantID uuid.UUID) error

	// PublishActivated publishes a tenant activated event.
	PublishActivated(ctx context.Context, tenant *Tenant) error

	// PublishSuspended publishes a tenant suspended event.
	PublishSuspended(ctx context.Context, tenant *Tenant) error

	// PublishSettingsUpdated publishes a settings updated event.
	PublishSettingsUpdated(ctx context.Context, tenantID uuid.UUID, settings *Settings) error
}
