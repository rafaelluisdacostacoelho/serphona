package customer

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Customer represents a billing customer linked to a tenant
type Customer struct {
	ID               uuid.UUID         `json:"id" gorm:"type:uuid;primary_key"`
	TenantID         uuid.UUID         `json:"tenant_id" gorm:"type:uuid;not null;index"`
	StripeCustomerID string            `json:"stripe_customer_id" gorm:"unique;not null"`
	Email            string            `json:"email" gorm:"not null"`
	Name             string            `json:"name"`
	Phone            string            `json:"phone"`
	Metadata         datatypes.JSONMap `json:"metadata" gorm:"type:jsonb"`
	CreatedAt        time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for Customer
func (Customer) TableName() string {
	return "customers"
}

// NewCustomer creates a new customer instance
func NewCustomer(tenantID uuid.UUID, email, name string) *Customer {
	return &Customer{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Email:     email,
		Name:      name,
		Metadata:  datatypes.JSONMap{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// SetStripeCustomerID sets the Stripe customer ID
func (c *Customer) SetStripeCustomerID(stripeID string) {
	c.StripeCustomerID = stripeID
	c.UpdatedAt = time.Now()
}

// UpdateMetadata updates customer metadata
func (c *Customer) UpdateMetadata(key string, value interface{}) {
	if c.Metadata == nil {
		c.Metadata = datatypes.JSONMap{}
	}
	c.Metadata[key] = value
	c.UpdatedAt = time.Now()
}
