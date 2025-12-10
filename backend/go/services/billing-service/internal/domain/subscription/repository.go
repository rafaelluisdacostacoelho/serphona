package subscription

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for subscription persistence
type Repository interface {
	Create(ctx context.Context, subscription *Subscription) error
	FindByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
	FindByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*Subscription, error)
	FindByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*Subscription, error)
	FindActiveByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*Subscription, error)
	Update(ctx context.Context, subscription *Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*Subscription, error)
}
