package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
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
	if err := authmw.EnforceTenant(ctx, wallet.TenantID.String()); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *WalletRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Wallet, error) {
	var w domain.Wallet
	tenantID, err := authmw.TenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&w).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.Wallet, error) {
	var w domain.Wallet
	if err := authmw.EnforceTenant(ctx, tenantID.String()); err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).First(&w, "tenant_id = ?", tenantID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) Update(ctx context.Context, wallet *domain.Wallet) error {
	if err := authmw.EnforceTenant(ctx, wallet.TenantID.String()); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(wallet).Error
}

func (r *WalletRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tenantID, err := authmw.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	query := r.db.WithContext(ctx).Where("id = ?", id)
	if !isPlatformTenant(tenantID) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Delete(&domain.Wallet{}).Error
}

func (r *WalletRepository) CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error {
	if err := r.ensureWalletTenant(ctx, tx.WalletID); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *WalletRepository) FindTransactionsByWalletID(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]*domain.WalletTransaction, error) {
	if err := r.ensureWalletTenant(ctx, walletID); err != nil {
		return nil, err
	}

	var txs []*domain.WalletTransaction
	if err := r.db.WithContext(ctx).Where("wallet_id = ?", walletID).Offset(offset).Limit(limit).Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *WalletRepository) FindTransactionByID(ctx context.Context, id uuid.UUID) (*domain.WalletTransaction, error) {
	tenantID, err := authmw.TenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var tx domain.WalletTransaction

	query := r.db.WithContext(ctx).
		Joins("JOIN wallets ON wallets.id = wallet_transactions.wallet_id").
		Where("wallet_transactions.id = ?", id)

	if !isPlatformTenant(tenantID) {
		query = query.Where("wallets.tenant_id = ?", tenantID)
	}

	if err := query.First(&tx).Error; err != nil {
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
	if err := r.ensureWalletTenant(ctx, walletID); err != nil {
		return nil, err
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

func (r *WalletRepository) ensureWalletTenant(ctx context.Context, walletID uuid.UUID) error {
	tenantID, err := authmw.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	if isPlatformTenant(tenantID) {
		return nil
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.Wallet{}).Where("id = ? AND tenant_id = ?", walletID, tenantID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return autherrors.ErrInsufficientPermissions
	}

	return nil
}

func isPlatformTenant(tenantID string) bool {
	return tenantID == "platform"
}
