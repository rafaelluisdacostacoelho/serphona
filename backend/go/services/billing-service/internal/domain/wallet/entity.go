package wallet

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrWalletLocked        = errors.New("wallet is locked")
)

// TransactionType represents the type of wallet transaction
type TransactionType string

const (
	TransactionTypeCredit TransactionType = "credit"
	TransactionTypeDebit  TransactionType = "debit"
)

// Wallet represents a credit wallet for a tenant
type Wallet struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;unique;not null;index"`
	Balance   int64     `json:"balance" gorm:"not null;default:0"` // Balance in cents
	Currency  string    `json:"currency" gorm:"not null;default:'BRL'"`
	IsLocked  bool      `json:"is_locked" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for Wallet
func (*Wallet) TableName() string {
	return "wallets"
}

// NewWallet creates a new wallet instance
func NewWallet(tenantID uuid.UUID, initialBalance int64, currency string) *Wallet {
	return &Wallet{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Balance:   initialBalance,
		Currency:  currency,
		IsLocked:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Credit adds credits to the wallet
func (w *Wallet) Credit(amount int64) error {
	if w.IsLocked {
		return ErrWalletLocked
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	w.Balance += amount
	w.UpdatedAt = time.Now()
	return nil
}

// Debit removes credits from the wallet
func (w *Wallet) Debit(amount int64) error {
	if w.IsLocked {
		return ErrWalletLocked
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if w.Balance < amount {
		return ErrInsufficientBalance
	}
	w.Balance -= amount
	w.UpdatedAt = time.Now()
	return nil
}

// Lock locks the wallet
func (w *Wallet) Lock() {
	w.IsLocked = true
	w.UpdatedAt = time.Now()
}

// Unlock unlocks the wallet
func (w *Wallet) Unlock() {
	w.IsLocked = false
	w.UpdatedAt = time.Now()
}

// HasSufficientBalance checks if the wallet has enough balance
func (w *Wallet) HasSufficientBalance(amount int64) bool {
	return w.Balance >= amount
}

// WalletTransaction represents a transaction in a wallet
type WalletTransaction struct {
	ID          uuid.UUID              `json:"id" gorm:"type:uuid;primary_key"`
	WalletID    uuid.UUID              `json:"wallet_id" gorm:"type:uuid;not null;index"`
	Amount      int64                  `json:"amount" gorm:"not null"` // Amount in cents
	Type        TransactionType        `json:"type" gorm:"not null"`
	Description string                 `json:"description"`
	Reference   string                 `json:"reference"` // External reference (e.g., invoice ID, usage record ID)
	Metadata    map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt   time.Time              `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName specifies the table name for WalletTransaction
func (*WalletTransaction) TableName() string {
	return "wallet_transactions"
}

// NewWalletTransaction creates a new wallet transaction
func NewWalletTransaction(walletID uuid.UUID, amount int64, txType TransactionType, description string) *WalletTransaction {
	return &WalletTransaction{
		ID:          uuid.New(),
		WalletID:    walletID,
		Amount:      amount,
		Type:        txType,
		Description: description,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
	}
}

// SetReference sets the external reference for the transaction
func (wt *WalletTransaction) SetReference(reference string) {
	wt.Reference = reference
}

// AddMetadata adds metadata to the transaction
func (wt *WalletTransaction) AddMetadata(key string, value interface{}) {
	if wt.Metadata == nil {
		wt.Metadata = make(map[string]interface{})
	}
	wt.Metadata[key] = value
}
