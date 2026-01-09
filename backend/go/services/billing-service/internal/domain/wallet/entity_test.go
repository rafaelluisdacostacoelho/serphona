package wallet

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestWalletTableNames(t *testing.T) {
	if got := (&Wallet{}).TableName(); got != "wallets" {
		t.Fatalf("wallet table name mismatch, got %s", got)
	}
	if got := (&WalletTransaction{}).TableName(); got != "wallet_transactions" {
		t.Fatalf("wallet transaction table name mismatch, got %s", got)
	}
}

func TestWalletCreditAndDebit(t *testing.T) {
	tenantID := uuid.New()
	wallet := NewWallet(tenantID, 100, "USD")

	if err := wallet.Credit(50); err != nil {
		t.Fatalf("expected credit success, got %v", err)
	}
	if wallet.Balance != 150 {
		t.Fatalf("expected balance 150 after credit, got %d", wallet.Balance)
	}

	if err := wallet.Debit(40); err != nil {
		t.Fatalf("expected debit success, got %v", err)
	}
	if wallet.Balance != 110 {
		t.Fatalf("expected balance 110 after debit, got %d", wallet.Balance)
	}
}

func TestWalletValidationErrors(t *testing.T) {
	wallet := NewWallet(uuid.New(), 100, "USD")

	if err := wallet.Credit(0); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected invalid amount error on credit, got %v", err)
	}

	if err := wallet.Debit(-10); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected invalid amount error on debit, got %v", err)
	}

	if err := wallet.Debit(200); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance error, got %v", err)
	}
}

func TestWalletLocking(t *testing.T) {
	wallet := NewWallet(uuid.New(), 100, "USD")
	wallet.Lock()
	if !wallet.IsLocked {
		t.Fatalf("expected wallet locked")
	}

	if err := wallet.Credit(10); !errors.Is(err, ErrWalletLocked) {
		t.Fatalf("expected wallet locked error on credit, got %v", err)
	}
	if err := wallet.Debit(10); !errors.Is(err, ErrWalletLocked) {
		t.Fatalf("expected wallet locked error on debit, got %v", err)
	}

	wallet.Unlock()
	if wallet.IsLocked {
		t.Fatalf("expected wallet unlocked")
	}
}

func TestWalletHasSufficientBalance(t *testing.T) {
	wallet := NewWallet(uuid.New(), 75, "USD")
	if !wallet.HasSufficientBalance(50) {
		t.Fatalf("expected wallet to have enough balance")
	}
	if wallet.HasSufficientBalance(100) {
		t.Fatalf("expected wallet to not have enough balance")
	}
}

func TestWalletTransactionHelpers(t *testing.T) {
	walletID := uuid.New()
	tx := NewWalletTransaction(walletID, 25, TransactionTypeDebit, "usage charge")

	if tx.WalletID != walletID || tx.Amount != 25 || tx.Type != TransactionTypeDebit {
		t.Fatalf("unexpected transaction fields: %+v", tx)
	}

	tx.SetReference("ref-123")
	if tx.Reference != "ref-123" {
		t.Fatalf("expected reference to be set")
	}

	tx.AddMetadata("k", "v")
	if val, ok := tx.Metadata["k"]; !ok || val != "v" {
		t.Fatalf("expected metadata to be added, got %+v", tx.Metadata)
	}

	tx.Metadata = nil
	tx.AddMetadata("another", 10)
	if val, ok := tx.Metadata["another"]; !ok || val != 10 {
		t.Fatalf("expected metadata map to initialize on nil, got %+v", tx.Metadata)
	}
}
