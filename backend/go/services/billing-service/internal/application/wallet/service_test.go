package wallet

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

func TestCalculateAmountPerPlan(t *testing.T) {
	pricing := ConfigPricingTable{plans: map[string]Pricing{
		"starter": {CallCents: 5, MinuteCents: 10, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 15},
		"pro":     {CallCents: 4, MinuteCents: 8, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 12},
	}, defaultPlan: "starter"}

	svc := Service{pricing: pricing}

	evt := UsageReported{TenantID: uuid.New(), Calls: 1, Minutes: 2, Messages: 3, APIRequests: 4, StorageGB: 1.5, Plan: "pro"}
	amount := svc.calculateAmount(evt)

	expected := int64(0)
	expected += 1 * 4 // calls
	expected += 2 * 8 // minutes
	expected += 3 * 2 // messages
	expected += 4 * 1 // api requests
	expected += int64(math.Round(1.5 * 12))

	if amount != expected {
		t.Fatalf("unexpected amount: got %d want %d", amount, expected)
	}
}

func TestDebitUsageIsIdempotentByRequestID(t *testing.T) {
	repo := &fakeWalletRepo{}
	pricing := ConfigPricingTable{plans: map[string]Pricing{
		"starter": {CallCents: 5},
	}, defaultPlan: "starter"}

	svc := NewService(repo, "USD", 100, pricing)
	tenantID := uuid.New()
	evt := UsageReported{TenantID: tenantID, Calls: 1, Plan: "starter", RequestID: "req-1"}

	if err := svc.DebitUsage(context.Background(), evt); err != nil {
		t.Fatalf("first debit failed: %v", err)
	}

	firstBalance := repo.wallet.Balance
	if firstBalance != 95 {
		t.Fatalf("unexpected balance after first debit: %d", firstBalance)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected one transaction recorded, got %d", len(repo.transactions))
	}

	if err := svc.DebitUsage(context.Background(), evt); !errors.Is(err, ErrDuplicateRequest) {
		t.Fatalf("second debit should return ErrDuplicateRequest, got %v", err)
	}

	if repo.wallet.Balance != firstBalance {
		t.Fatalf("balance changed on duplicate request: got %d want %d", repo.wallet.Balance, firstBalance)
	}
	if len(repo.transactions) != 1 {
		t.Fatalf("transaction count changed on duplicate request: %d", len(repo.transactions))
	}
}

func TestDebitUsageReturnsDuplicateWhenReferenceExists(t *testing.T) {
	repo := &fakeWalletRepo{}
	pricing := ConfigPricingTable{plans: map[string]Pricing{
		"starter": {CallCents: 5},
	}, defaultPlan: "starter"}

	svc := NewService(repo, "USD", 100, pricing)
	tenantID := uuid.New()
	evt := UsageReported{TenantID: tenantID, Calls: 1, Plan: "starter", RequestID: "req-dup"}

	if err := svc.DebitUsage(context.Background(), evt); err != nil {
		t.Fatalf("first debit failed: %v", err)
	}

	if err := svc.DebitUsage(context.Background(), evt); !errors.Is(err, ErrDuplicateRequest) {
		t.Fatalf("expected ErrDuplicateRequest, got %v", err)
	}
}

func TestDebitUsageTreatsUniqueConstraintAsDuplicate(t *testing.T) {
	repo := &fakeWalletRepo{createTransactionErr: gorm.ErrDuplicatedKey}
	pricing := ConfigPricingTable{plans: map[string]Pricing{
		"starter": {CallCents: 5},
	}, defaultPlan: "starter"}

	svc := NewService(repo, "USD", 100, pricing)
	evt := UsageReported{TenantID: uuid.New(), Calls: 1, Plan: "starter", RequestID: "req-dup"}

	if err := svc.DebitUsage(context.Background(), evt); !errors.Is(err, ErrDuplicateRequest) {
		t.Fatalf("expected ErrDuplicateRequest from duplicate key, got %v", err)
	}
}

// fakeWalletRepo is a minimal in-memory repository for tests.
type fakeWalletRepo struct {
	wallet               *domain.Wallet
	transactions         map[string]*domain.WalletTransaction
	createTransactionErr error
}

func (r *fakeWalletRepo) Create(_ context.Context, wallet *domain.Wallet) error {
	r.wallet = wallet
	return nil
}

func (r *fakeWalletRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Wallet, error) {
	if r.wallet != nil && r.wallet.ID == id {
		return r.wallet, nil
	}
	return nil, nil
}

func (r *fakeWalletRepo) FindByTenantID(_ context.Context, tenantID uuid.UUID) (*domain.Wallet, error) {
	if r.wallet != nil && r.wallet.TenantID == tenantID {
		return r.wallet, nil
	}
	return nil, nil
}

func (r *fakeWalletRepo) Update(_ context.Context, wallet *domain.Wallet) error {
	r.wallet = wallet
	return nil
}

func (r *fakeWalletRepo) Delete(context.Context, uuid.UUID) error { return nil }

func (r *fakeWalletRepo) CreateTransaction(_ context.Context, tx *domain.WalletTransaction) error {
	if r.transactions == nil {
		r.transactions = make(map[string]*domain.WalletTransaction)
	}
	r.transactions[tx.ID.String()] = tx
	return r.createTransactionErr
}

func (r *fakeWalletRepo) FindTransactionsByWalletID(context.Context, uuid.UUID, int, int) ([]*domain.WalletTransaction, error) {
	var out []*domain.WalletTransaction
	for _, tx := range r.transactions {
		out = append(out, tx)
	}
	return out, nil
}

func (r *fakeWalletRepo) FindTransactionByID(_ context.Context, id uuid.UUID) (*domain.WalletTransaction, error) {
	if tx, ok := r.transactions[id.String()]; ok {
		return tx, nil
	}
	return nil, nil
}

func (r *fakeWalletRepo) FindTransactionByReference(_ context.Context, _ uuid.UUID, reference string) (*domain.WalletTransaction, error) {
	for _, tx := range r.transactions {
		if tx.Reference == reference {
			return tx, nil
		}
	}
	return nil, nil
}
