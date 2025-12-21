package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/customer"
	"gorm.io/gorm"
)

type customerRepository struct {
	db *gorm.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *gorm.DB) customer.Repository {
	return &customerRepository{db: db}
}

func (r *customerRepository) Create(ctx context.Context, cust *customer.Customer) error {
	return r.db.WithContext(ctx).Create(cust).Error
}

func (r *customerRepository) FindByID(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	var cust customer.Customer
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &cust, nil
}

func (r *customerRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*customer.Customer, error) {
	var cust customer.Customer
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &cust, nil
}

func (r *customerRepository) FindByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*customer.Customer, error) {
	var cust customer.Customer
	err := r.db.WithContext(ctx).Where("stripe_customer_id = ?", stripeCustomerID).First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &cust, nil
}

func (r *customerRepository) Update(ctx context.Context, cust *customer.Customer) error {
	return r.db.WithContext(ctx).Save(cust).Error
}

func (r *customerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&customer.Customer{}, "id = ?", id).Error
}
