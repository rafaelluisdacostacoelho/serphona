package postgres

import (
	"context"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
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
	if err := authmw.EnforceTenant(ctx, cust.TenantID.String()); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(cust).Error
}

func (r *customerRepository) FindByID(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	var cust customer.Customer
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if tenantID, err := authmw.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &cust, nil
}

func (r *customerRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*customer.Customer, error) {
	var cust customer.Customer
	if err := authmw.EnforceTenant(ctx, tenantID.String()); err != nil {
		return nil, err
	}
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &cust, nil
}

func (r *customerRepository) FindByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*customer.Customer, error) {
	var cust customer.Customer
	query := r.db.WithContext(ctx).Where("stripe_customer_id = ?", stripeCustomerID)
	if tenantID, err := authmw.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &cust, nil
}

func (r *customerRepository) Update(ctx context.Context, cust *customer.Customer) error {
	if err := authmw.EnforceTenant(ctx, cust.TenantID.String()); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(cust).Error
}

func (r *customerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if tenantID, err := authmw.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	return query.Delete(&customer.Customer{}).Error
}
