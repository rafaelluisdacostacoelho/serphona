package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/customer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&customer.Customer{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestCreateEnforcesTenant(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)

	tenantID := uuid.New()
	ctx := authmw.WithTenantID(context.Background(), tenantID.String())
	cust := &customer.Customer{
		ID:               uuid.New(),
		TenantID:         tenantID,
		StripeCustomerID: "cus_123",
		Email:            "user@example.com",
	}

	if err := repo.Create(ctx, cust); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if err := repo.Create(context.Background(), cust); !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}

	otherTenant := uuid.New()
	otherCtx := authmw.WithTenantID(context.Background(), otherTenant.String())
	if err := repo.Create(otherCtx, cust); !errors.Is(err, autherrors.ErrInsufficientPermissions) {
		t.Fatalf("expected insufficient permissions, got %v", err)
	}
}

func TestFindByIDScopesToTenant(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)

	tenantA := uuid.New()
	tenantB := uuid.New()

	custA := &customer.Customer{ID: uuid.New(), TenantID: tenantA, StripeCustomerID: "cus_a", Email: "a@example.com"}
	custB := &customer.Customer{ID: uuid.New(), TenantID: tenantB, StripeCustomerID: "cus_b", Email: "b@example.com"}

	if err := db.Create(custA).Error; err != nil {
		t.Fatalf("failed to seed custA: %v", err)
	}
	if err := db.Create(custB).Error; err != nil {
		t.Fatalf("failed to seed custB: %v", err)
	}

	ctxA := authmw.WithTenantID(context.Background(), tenantA.String())
	found, err := repo.FindByID(ctxA, custA.ID)
	if err != nil {
		t.Fatalf("expected to find customer, got %v", err)
	}
	if found.ID != custA.ID {
		t.Fatalf("expected customer %s, got %s", custA.ID, found.ID)
	}

	if _, err := repo.FindByID(ctxA, custB.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found for other tenant, got %v", err)
	}

	if _, err := repo.FindByID(context.Background(), custA.ID); !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized when tenant missing, got %v", err)
	}
}
