package postgres

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPricingTableReturnsPlanFromDB(t *testing.T) {
	db := setupPricingDB(t)

	proPricing := walletapp.Pricing{CallCents: 2, MinuteCents: 3, MessageCents: 4, APIRequestCents: 5, StorageGBCents: 6}
	insertPlan(t, db, "pro", false, proPricing)
	insertPlan(t, db, "starter", true, walletapp.Pricing{CallCents: 1})

	table := NewPricingTable(db, staticPricingTable{})

	got := table.ForPlan("PRO")
	if got != proPricing {
		t.Fatalf("unexpected pricing: %+v", got)
	}
}

func TestPricingTableFallsBackToDefaultPlan(t *testing.T) {
	db := setupPricingDB(t)

	defaultPricing := walletapp.Pricing{CallCents: 7, MinuteCents: 8, MessageCents: 9, APIRequestCents: 10, StorageGBCents: 11}
	insertPlan(t, db, "starter", true, defaultPricing)

	table := NewPricingTable(db, staticPricingTable{})

	got := table.ForPlan("unknown")
	if got != defaultPricing {
		t.Fatalf("expected default pricing, got %+v", got)
	}
}

func TestPricingTableFallsBackToConfigWhenDBEmpty(t *testing.T) {
	db := setupPricingDB(t)

	fallback := staticPricingTable{
		pricing: map[string]walletapp.Pricing{
			"starter": {CallCents: 1, MinuteCents: 1, MessageCents: 1, APIRequestCents: 1, StorageGBCents: 1},
		},
		defaultPlan: "starter",
	}

	table := NewPricingTable(db, fallback)

	got := table.ForPlan("missing")
	if got != fallback.pricing["starter"] {
		t.Fatalf("expected fallback pricing, got %+v", got)
	}
}

func setupPricingDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	ddl := `CREATE TABLE pricing_plans (
        plan_id VARCHAR(100) PRIMARY KEY,
        display_name VARCHAR(255),
        is_default BOOLEAN NOT NULL DEFAULT false,
        call_cents BIGINT NOT NULL DEFAULT 0,
        minute_cents BIGINT NOT NULL DEFAULT 0,
        message_cents BIGINT NOT NULL DEFAULT 0,
        api_request_cents BIGINT NOT NULL DEFAULT 0,
        storage_gb_cents BIGINT NOT NULL DEFAULT 0,
        effective_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`

	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create pricing_plans table: %v", err)
	}

	return db
}

func insertPlan(t *testing.T, db *gorm.DB, planID string, isDefault bool, pricing walletapp.Pricing) {
	t.Helper()

	query := `INSERT INTO pricing_plans (plan_id, is_default, call_cents, minute_cents, message_cents, api_request_cents, storage_gb_cents, effective_from) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if err := db.Exec(query, planID, isDefault, pricing.CallCents, pricing.MinuteCents, pricing.MessageCents, pricing.APIRequestCents, pricing.StorageGBCents, time.Now()).Error; err != nil {
		t.Fatalf("failed to insert plan %s: %v", planID, err)
	}
}

// staticPricingTable is a minimal in-memory PricingTable used for tests.
type staticPricingTable struct {
	pricing     map[string]walletapp.Pricing
	defaultPlan string
}

func (s staticPricingTable) ForPlan(plan string) walletapp.Pricing {
	normalized := strings.ToLower(strings.TrimSpace(plan))
	if val, ok := s.pricing[normalized]; ok {
		return val
	}
	if val, ok := s.pricing[s.defaultPlan]; ok {
		return val
	}
	return walletapp.Pricing{}
}
