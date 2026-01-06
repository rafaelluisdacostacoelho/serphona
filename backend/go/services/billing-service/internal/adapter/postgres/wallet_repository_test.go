package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWalletDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&domain.Wallet{}, &domain.WalletTransaction{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestWalletCreateEnforcesTenant(t *testing.T) {
	db := setupWalletDB(t)
	repo := NewWalletRepository(db)

	tenantID := uuid.New()
	wallet := domain.NewWallet(tenantID, 100, "USD")

	ctx := authmw.WithTenantID(context.Background(), tenantID.String())
	if err := repo.Create(ctx, wallet); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if err := repo.Create(context.Background(), wallet); !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized without tenant, got %v", err)
	}

	otherCtx := authmw.WithTenantID(context.Background(), uuid.New().String())
	if err := repo.Create(otherCtx, wallet); !errors.Is(err, autherrors.ErrInsufficientPermissions) {
		t.Fatalf("expected insufficient permissions, got %v", err)
	}
}

func TestWalletFindByIDScopesToTenant(t *testing.T) {
	db := setupWalletDB(t)
	repo := NewWalletRepository(db)

	tenantA := uuid.New()
	tenantB := uuid.New()

	walletA := domain.NewWallet(tenantA, 100, "USD")
	walletB := domain.NewWallet(tenantB, 200, "USD")

	if err := db.Create(walletA).Error; err != nil {
		t.Fatalf("seed walletA: %v", err)
	}
	if err := db.Create(walletB).Error; err != nil {
		t.Fatalf("seed walletB: %v", err)
	}

	ctxA := authmw.WithTenantID(context.Background(), tenantA.String())
	got, err := repo.FindByID(ctxA, walletA.ID)
	if err != nil {
		t.Fatalf("expected to find walletA, got %v", err)
	}
	if got == nil || got.ID != walletA.ID {
		t.Fatalf("expected walletA, got %+v", got)
	}

	if got, err := repo.FindByID(ctxA, walletB.ID); err != nil || got != nil {
		t.Fatalf("expected no access to walletB, got wallet=%+v err=%v", got, err)
	}

	if _, err := repo.FindByID(context.Background(), walletA.ID); !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized without tenant, got %v", err)
	}
}

func TestCreateTransactionEnforcesWalletTenant(t *testing.T) {
	db := setupWalletDB(t)
	repo := NewWalletRepository(db)

	tenant := uuid.New()
	wallet := domain.NewWallet(tenant, 100, "USD")
	if err := db.Create(wallet).Error; err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	ctxTenant := authmw.WithTenantID(context.Background(), tenant.String())
	tx := domain.NewWalletTransaction(wallet.ID, 10, domain.TransactionTypeDebit, "desc")
	if err := repo.CreateTransaction(ctxTenant, tx); err != nil {
		t.Fatalf("expected create transaction success, got %v", err)
	}

	otherCtx := authmw.WithTenantID(context.Background(), uuid.New().String())
	txOther := domain.NewWalletTransaction(wallet.ID, 15, domain.TransactionTypeDebit, "desc")
	if err := repo.CreateTransaction(otherCtx, txOther); !errors.Is(err, autherrors.ErrInsufficientPermissions) {
		t.Fatalf("expected insufficient permissions for other tenant, got %v", err)
	}

	platformCtx := authmw.WithTenantID(context.Background(), "platform")
	txPlatform := domain.NewWalletTransaction(wallet.ID, 20, domain.TransactionTypeDebit, "desc")
	if err := repo.CreateTransaction(platformCtx, txPlatform); err != nil {
		t.Fatalf("expected platform to bypass tenant restriction, got %v", err)
	}
}

func TestFindTransactionsByWalletIDEnforcesTenant(t *testing.T) {
	db := setupWalletDB(t)
	repo := NewWalletRepository(db)

	tenant := uuid.New()
	wallet := domain.NewWallet(tenant, 100, "USD")
	if err := db.Create(wallet).Error; err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	tx := domain.NewWalletTransaction(wallet.ID, 5, domain.TransactionTypeDebit, "desc")
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	tenantCtx := authmw.WithTenantID(context.Background(), tenant.String())
	txs, err := repo.FindTransactionsByWalletID(tenantCtx, wallet.ID, 0, 10)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(txs) != 1 || txs[0].ID != tx.ID {
		t.Fatalf("unexpected transactions: %+v", txs)
	}

	otherCtx := authmw.WithTenantID(context.Background(), uuid.New().String())
	if _, err := repo.FindTransactionsByWalletID(otherCtx, wallet.ID, 0, 10); !errors.Is(err, autherrors.ErrInsufficientPermissions) {
		t.Fatalf("expected insufficient permissions for other tenant, got %v", err)
	}
}

func TestFindTransactionByIDScopesToTenant(t *testing.T) {
	db := setupWalletDB(t)
	repo := NewWalletRepository(db)

	tenant := uuid.New()
	wallet := domain.NewWallet(tenant, 100, "USD")
	if err := db.Create(wallet).Error; err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	tx := domain.NewWalletTransaction(wallet.ID, 5, domain.TransactionTypeDebit, "desc")
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	tenantCtx := authmw.WithTenantID(context.Background(), tenant.String())
	found, err := repo.FindTransactionByID(tenantCtx, tx.ID)
	if err != nil {
		t.Fatalf("expected find to succeed, got %v", err)
	}
	if found == nil || found.ID != tx.ID {
		t.Fatalf("unexpected transaction: %+v", found)
	}

	otherCtx := authmw.WithTenantID(context.Background(), uuid.New().String())
	if found, err := repo.FindTransactionByID(otherCtx, tx.ID); err != nil || found != nil {
		t.Fatalf("expected no access for other tenant, got tx=%+v err=%v", found, err)
	}

	if _, err := repo.FindTransactionByID(context.Background(), tx.ID); !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized without tenant, got %v", err)
	}
}
