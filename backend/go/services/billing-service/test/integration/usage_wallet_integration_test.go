//go:build integration
// +build integration

package integration

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	pgrepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/postgres"
	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

func TestUsageReportedDebitsWalletAndStoresTransaction(t *testing.T) {
	db := setupIntegrationDB(t)

	pricingCfg := config.PricingConfig{DefaultPlan: "starter", Plans: map[string]config.PlanPricing{
		"starter":      {CallCents: 5, MinuteCents: 10, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 15},
		"pro":          {CallCents: 4, MinuteCents: 8, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 12},
		"professional": {CallCents: 4, MinuteCents: 8, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 12},
	}}

	configPricing := walletapp.NewConfigPricingTable(pricingCfg)
	pricingTable := pgrepo.NewPricingTable(db, configPricing)
	walletRepo := pgrepo.NewWalletRepository(db)
	svc := walletapp.NewService(walletRepo, "USD", 100, pricingTable)

	evt := walletapp.UsageReported{
		TenantID:    uuid.New(),
		Period:      "2024-12",
		Source:      "integration-test",
		Calls:       1,
		Minutes:     2,
		Messages:    3,
		APIRequests: 4,
		StorageGB:   1.5,
		Plan:        "pro",
		RequestID:   "req-integration-1",
	}

	ctx := authmw.WithTenantID(context.Background(), evt.TenantID.String())

	if err := svc.DebitUsage(ctx, evt); err != nil {
		t.Fatalf("DebitUsage failed: %v", err)
	}

	wallet, err := walletRepo.FindByTenantID(ctx, evt.TenantID)
	if err != nil {
		t.Fatalf("find wallet: %v", err)
	}
	if wallet == nil {
		t.Fatalf("wallet was not created")
	}

	expectedAmount := int64(0)
	expectedAmount += int64(evt.Calls) * 4   // call 4
	expectedAmount += int64(evt.Minutes) * 8 // minute 8
	expectedAmount += int64(evt.Messages) * 2
	expectedAmount += int64(evt.APIRequests) * 1
	expectedAmount += int64(math.Round(1.5 * 12))

	if wallet.Balance != 100-expectedAmount {
		t.Fatalf("unexpected wallet balance: got %d want %d", wallet.Balance, 100-expectedAmount)
	}

	txs, err := walletRepo.FindTransactionsByWalletID(ctx, wallet.ID, 0, 10)
	if err != nil {
		t.Fatalf("find transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected one transaction, got %d", len(txs))
	}
	if txs[0].Reference != evt.RequestID {
		t.Fatalf("transaction reference mismatch: got %s want %s", txs[0].Reference, evt.RequestID)
	}
}

func setupIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormsqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&domain.Wallet{}, &domain.WalletTransaction{}, &pgrepo.PricingPlan{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// Seed pricing plans to exercise DB-backed pricing resolution
	seed := []pgrepo.PricingPlan{
		{PlanID: "starter", IsDefault: true, CallCents: 5, MinuteCents: 10, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 15, EffectiveFrom: time.Now().Format(time.RFC3339)},
		{PlanID: "pro", IsDefault: false, CallCents: 4, MinuteCents: 8, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 12, EffectiveFrom: time.Now().Format(time.RFC3339)},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed pricing_plans: %v", err)
	}

	return db
}
