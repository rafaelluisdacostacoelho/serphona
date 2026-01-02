package postgres

import (
	"strings"

	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
	"gorm.io/gorm"
)

// PricingPlan represents the pricing_plans table.
type PricingPlan struct {
	PlanID          string `gorm:"column:plan_id;primaryKey"`
	DisplayName     string `gorm:"column:display_name"`
	IsDefault       bool   `gorm:"column:is_default"`
	CallCents       int64  `gorm:"column:call_cents"`
	MinuteCents     int64  `gorm:"column:minute_cents"`
	MessageCents    int64  `gorm:"column:message_cents"`
	APIRequestCents int64  `gorm:"column:api_request_cents"`
	StorageGBCents  int64  `gorm:"column:storage_gb_cents"`
	EffectiveFrom   string `gorm:"column:effective_from"`
}

// TableName maps PricingPlan to the pricing_plans table.
func (PricingPlan) TableName() string {
	return "pricing_plans"
}

// PricingTable resolves pricing from the database with optional fallback.
type PricingTable struct {
	db       *gorm.DB
	fallback walletapp.PricingTable
}

// NewPricingTable creates a DB-backed pricing table with a fallback (e.g., config).
func NewPricingTable(db *gorm.DB, fallback walletapp.PricingTable) walletapp.PricingTable {
	return &PricingTable{db: db, fallback: fallback}
}

// ForPlan returns pricing for the given plan, falling back to default or fallback table.
func (p *PricingTable) ForPlan(plan string) walletapp.Pricing {
	normalized := strings.ToLower(strings.TrimSpace(plan))

	var model PricingPlan
	if normalized != "" {
		if err := p.db.
			Where("plan_id = ?", normalized).
			Order("effective_from DESC").
			Limit(1).
			First(&model).Error; err == nil {
			return toPricing(model)
		}
	}

	if err := p.db.
		Where("is_default = true").
		Order("effective_from DESC").
		Limit(1).
		First(&model).Error; err == nil {
		return toPricing(model)
	}

	if p.fallback != nil {
		return p.fallback.ForPlan(normalized)
	}

	return walletapp.Pricing{}
}

func toPricing(model PricingPlan) walletapp.Pricing {
	return walletapp.Pricing{
		CallCents:       model.CallCents,
		MinuteCents:     model.MinuteCents,
		MessageCents:    model.MessageCents,
		APIRequestCents: model.APIRequestCents,
		StorageGBCents:  model.StorageGBCents,
	}
}
