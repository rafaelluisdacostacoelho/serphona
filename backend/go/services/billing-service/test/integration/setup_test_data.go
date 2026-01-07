package integration

import (
	"context"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/postgres"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

func SetupTestData(db *gorm.DB) {
	repo := postgres.NewWalletRepository(db)
	ctx := context.Background()

	// AutoMigrate to ensure tables exist
	if err := db.AutoMigrate(&domain.Tenant{}, &wallet.Wallet{}); err != nil {
		log.Fatalf("failed to migrate tables: %v", err)
	}

	// Create a test tenant
	tenant := &domain.Tenant{
		ID:   uuid.MustParse("6fc298ff-6aa8-4c6e-be90-488ae714d52c"),
		Name: "Test Tenant",
	}

	if err := db.Create(tenant).Error; err != nil {
		log.Fatalf("failed to create test tenant: %v", err)
	}

	// Create a test wallet
	wallet := &wallet.Wallet{
		ID:       uuid.New(),
		TenantID: uuid.MustParse("6fc298ff-6aa8-4c6e-be90-488ae714d52c"),
		Balance:  1000,
	}

	if err := repo.Create(ctx, wallet); err != nil {
		log.Fatalf("failed to create test wallet: %v", err)
	}

	log.Println("Test data setup complete.")
}
