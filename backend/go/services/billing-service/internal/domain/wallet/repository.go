package wallet

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for wallet persistence
type Repository interface {
	Create(ctx context.Context, wallet *Wallet) error
	FindByID(ctx context.Context, id uuid.UUID) (*Wallet, error)
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*Wallet, error)
	Update(ctx context.Context, wallet *Wallet) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Transaction operations
	CreateTransaction(ctx context.Context, transaction *WalletTransaction) error
	FindTransactionsByWalletID(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]*WalletTransaction, error)
	FindTransactionByID(ctx context.Context, id uuid.UUID) (*WalletTransaction, error)
	FindTransactionByReference(ctx context.Context, walletID uuid.UUID, reference string) (*WalletTransaction, error)
}
