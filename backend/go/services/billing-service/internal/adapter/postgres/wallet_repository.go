package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

// WalletRepository implements wallet persistence using GORM.
type WalletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *WalletRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Wallet, error) {
	var w domain.Wallet
	if err := r.db.WithContext(ctx).First(&w, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.Wallet, error) {
	var w domain.Wallet
	if err := r.db.WithContext(ctx).First(&w, "tenant_id = ?", tenantID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) Update(ctx context.Context, wallet *domain.Wallet) error {
	return r.db.WithContext(ctx).Save(wallet).Error
}

func (r *WalletRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Wallet{}, "id = ?", id).Error
}

func (r *WalletRepository) CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *WalletRepository) FindTransactionsByWalletID(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]*domain.WalletTransaction, error) {
	var txs []*domain.WalletTransaction
	if err := r.db.WithContext(ctx).Where("wallet_id = ?", walletID).Offset(offset).Limit(limit).Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *WalletRepository) FindTransactionByID(ctx context.Context, id uuid.UUID) (*domain.WalletTransaction, error) {
	var tx domain.WalletTransaction
	if err := r.db.WithContext(ctx).First(&tx, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *WalletRepository) FindTransactionByReference(ctx context.Context, walletID uuid.UUID, reference string) (*domain.WalletTransaction, error) {
	if reference == "" {
		return nil, nil
	}

	var tx domain.WalletTransaction
	if err := r.db.WithContext(ctx).First(&tx, "wallet_id = ? AND reference = ?", walletID, reference).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}
