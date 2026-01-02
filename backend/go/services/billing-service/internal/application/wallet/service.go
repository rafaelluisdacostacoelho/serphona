package wallet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

// Service provides wallet operations used by consumers.
type Service struct {
	repo            domain.Repository
	defaultCurrency string
	initialCredits  int64
	pricing         PricingTable
}

var ErrDuplicateRequest = errors.New("duplicate usage request")

// NewService creates a wallet service with pricing table.
func NewService(repo domain.Repository, defaultCurrency string, initialCredits int64, pricing PricingTable) *Service {
	return &Service{repo: repo, defaultCurrency: defaultCurrency, initialCredits: initialCredits, pricing: pricing}
}

// UsageReported is the incoming usage event payload.
type UsageReported struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	Period         string    `json:"period"`
	OccurredAt     string    `json:"occurred_at"`
	Source         string    `json:"source"`
	Calls          int       `json:"calls"`
	Minutes        int       `json:"minutes"`
	Messages       int       `json:"messages"`
	StorageGB      float64   `json:"storage_gb"`
	APIRequests    int       `json:"api_requests"`
	Plan           string    `json:"plan"` // This line remains unchanged
	SubscriptionID string    `json:"subscription_id"`
	RequestID      string    `json:"request_id"`
	TraceID        string    `json:"trace_id"`
}

// PricingTable resolves per-plan pricing; allows config or DB-backed implementations.
type PricingTable interface {
	ForPlan(plan string) Pricing
}

// Pricing holds per-dimension prices in cents.
type Pricing struct {
	CallCents       int64
	MinuteCents     int64
	MessageCents    int64
	APIRequestCents int64
	StorageGBCents  int64
}

// DebitUsage debits the tenant wallet based on usage.
func (s *Service) DebitUsage(ctx context.Context, evt UsageReported) error {
	amount := s.calculateAmount(evt)
	if amount <= 0 {
		return nil
	}

	wallet, err := s.repo.FindByTenantID(ctx, evt.TenantID)
	if err != nil {
		return fmt.Errorf("find wallet: %w", err)
	}
	if wallet == nil {
		wallet = domain.NewWallet(evt.TenantID, s.initialCredits, s.defaultCurrency)
		if err := s.repo.Create(ctx, wallet); err != nil {
			return fmt.Errorf("create wallet: %w", err)
		}
	}

	if evt.RequestID != "" {
		existing, err := s.repo.FindTransactionByReference(ctx, wallet.ID, evt.RequestID)
		if err != nil {
			return fmt.Errorf("find transaction by reference: %w", err)
		}
		if existing != nil {
			return ErrDuplicateRequest
		}
	}

	if err := wallet.Debit(amount); err != nil {
		return err
	}

	tx := domain.NewWalletTransaction(wallet.ID, amount, domain.TransactionTypeDebit, fmt.Sprintf("usage %s", evt.Period))
	tx.SetReference(evt.RequestID)
	tx.AddMetadata("source", evt.Source)
	tx.AddMetadata("period", evt.Period)

	if err := s.repo.Update(ctx, wallet); err != nil {
		return fmt.Errorf("update wallet: %w", err)
	}
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateRequest
		}
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (s *Service) calculateAmount(evt UsageReported) int64 {
	plan := strings.ToLower(strings.TrimSpace(evt.Plan))
	pricing := s.pricing.ForPlan(plan)

	var total int64
	total += int64(evt.Calls) * pricing.CallCents
	total += int64(evt.Minutes) * pricing.MinuteCents
	total += int64(evt.Messages) * pricing.MessageCents
	total += int64(evt.APIRequests) * pricing.APIRequestCents
	if evt.StorageGB > 0 && pricing.StorageGBCents > 0 {
		total += int64(math.Round(evt.StorageGB * float64(pricing.StorageGBCents)))
	}

	return total
}
