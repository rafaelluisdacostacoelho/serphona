package wallet

import "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"

// ConfigPricingTable implements PricingTable using in-memory config.
type ConfigPricingTable struct {
	plans       map[string]Pricing
	defaultPlan string
}

// NewConfigPricingTable builds a pricing table from config.
func NewConfigPricingTable(cfg config.PricingConfig) PricingTable {
	plans := make(map[string]Pricing, len(cfg.Plans))
	for plan, p := range cfg.Plans {
		plans[plan] = Pricing{
			CallCents:       p.CallCents,
			MinuteCents:     p.MinuteCents,
			MessageCents:    p.MessageCents,
			APIRequestCents: p.APIRequestCents,
			StorageGBCents:  p.StorageGBCents,
		}
	}
	return ConfigPricingTable{plans: plans, defaultPlan: cfg.DefaultPlan}
}

// ForPlan returns pricing for the given plan, falling back to default.
func (p ConfigPricingTable) ForPlan(plan string) Pricing {
	if val, ok := p.plans[plan]; ok {
		return val
	}
	return p.plans[p.defaultPlan]
}
