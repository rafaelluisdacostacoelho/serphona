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

// QuotaDTO represents tenant quotas and current usage.
type QuotaDTO struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	MaxAPIKeys         int       `json:"max_api_keys"`
	MaxUsers           int       `json:"max_users"`
	MaxCallsPerMonth   int       `json:"max_calls_per_month"`
	MaxMinutesPerMonth int       `json:"max_minutes_per_month"`
	MaxStorageGB       int       `json:"max_storage_gb"`
	UsedCalls          int       `json:"used_calls"`
	UsedMinutes        int       `json:"used_minutes"`
	UsedStorageGB      float64   `json:"used_storage_gb"`
	ResetAt            time.Time `json:"reset_at"`
}
