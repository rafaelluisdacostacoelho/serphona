package customer

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for customer persistence
type Repository interface {
	Create(ctx context.Context, customer *Customer) error
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*Customer, error)
	FindByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*Customer, error)
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id uuid.UUID) error
}
