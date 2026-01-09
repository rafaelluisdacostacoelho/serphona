package wallet

import (
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
)

func TestConfigPricingTable(t *testing.T) {
	cfg := config.PricingConfig{
		DefaultPlan: "starter",
		Plans: map[string]config.PlanPricing{
			"starter":      {CallCents: 1, MinuteCents: 2, MessageCents: 3, APIRequestCents: 4, StorageGBCents: 5},
			"professional": {CallCents: 10, MinuteCents: 20, MessageCents: 30, APIRequestCents: 40, StorageGBCents: 50},
		},
	}

	table := NewConfigPricingTable(cfg)

	got := table.ForPlan("professional")
	if got.CallCents != 10 || got.MinuteCents != 20 || got.MessageCents != 30 || got.APIRequestCents != 40 || got.StorageGBCents != 50 {
		t.Fatalf("unexpected professional plan pricing %+v", got)
	}

	fallback := table.ForPlan("unknown")
	if fallback.CallCents != 1 || fallback.MinuteCents != 2 || fallback.MessageCents != 3 || fallback.APIRequestCents != 4 || fallback.StorageGBCents != 5 {
		t.Fatalf("expected fallback to starter pricing, got %+v", fallback)
	}
}
