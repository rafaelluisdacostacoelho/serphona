// Package tenant contains the application layer for tenant use cases.
package tenant

import "errors"

// ListTenantsQuery represents the query to list tenants.
type ListTenantsQuery struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Status   string `json:"status,omitempty"`
	Search   string `json:"search,omitempty"`
}

// Validate validates the list tenants query.
func (q ListTenantsQuery) Validate() error {
	if q.Page < 1 {
		return errors.New("page must be greater than 0")
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		return errors.New("page_size must be between 1 and 100")
	}
	return nil
}

// ListTenantsResult represents the result of listing tenants.
type ListTenantsResult struct {
	Tenants    []*TenantDTO `json:"tenants"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}
