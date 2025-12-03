// Package tenant contains the application layer for tenant use cases.
package tenant

import (
	"time"

	"github.com/google/uuid"

	"tenant-manager/internal/domain/tenant"
)

// TenantDTO is the data transfer object for tenant.
type TenantDTO struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Email        string          `json:"email"`
	Phone        string          `json:"phone,omitempty"`
	Status       string          `json:"status"`
	Plan         string          `json:"plan"`
	Settings     tenant.Settings `json:"settings"`
	Metadata     tenant.Metadata `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	BillingEmail string          `json:"billing_email,omitempty"`
}
