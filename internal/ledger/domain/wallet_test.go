package domain

import (
	"errors"
	"testing"
)

func TestWalletStatus_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status WalletStatus
		err    error
	}{
		{
			name:   "open status is valid",
			status: WalletStatusOpen,
		},
		{
			name:   "suspended status is valid",
			status: WalletStatusSuspended,
		},
		{
			name:   "closed status is valid",
			status: WalletStatusClosed,
		},
		{
			name:   "unknown status is invalid",
			status: WalletStatus("blocked"),
			err:    ErrInvalidWalletStatus,
		},
		{
			name:   "empty status is invalid",
			status: WalletStatus(""),
			err:    ErrInvalidWalletStatus,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.status.Validate()

			if !errors.Is(err, tt.err) {
				t.Fatalf(
					"Validate() error = %v, want %v",
					err,
					tt.err,
				)
			}
		})
	}
}

func TestNewWallet(t *testing.T) {
	t.Parallel()

	validCurrency := fakeValidCurrency("BRL")

	tests := []struct {
		name            string
		id              WalletID
		ownerID         OwnerID
		ledgerAccountID AccountID
		currency        Currency
		err             error
	}{
		{
			name:            "creates an open wallet",
			id:              WalletID("wallet-001"),
			ownerID:         OwnerID("owner-001"),
			ledgerAccountID: AccountID("account-001"),
			currency:        validCurrency,
		},
		{
			name:            "rejects an empty wallet ID",
			id:              WalletID(""),
			ownerID:         OwnerID("owner-001"),
			ledgerAccountID: AccountID("account-001"),
			currency:        validCurrency,
			err:             ErrInvalidID,
		},
		{
			name:            "rejects an empty owner ID",
			id:              WalletID("wallet-001"),
			ownerID:         OwnerID(""),
			ledgerAccountID: AccountID("account-001"),
			currency:        validCurrency,
			err:             ErrEmptyWalletOwnerID,
		},
		{
			name:            "rejects an empty ledger account ID",
			id:              WalletID("wallet-001"),
			ownerID:         OwnerID("owner-001"),
			ledgerAccountID: AccountID(""),
			currency:        validCurrency,
			err:             ErrInvalidID,
		},
		{
			name:            "rejects an invalid currency",
			id:              WalletID("wallet-001"),
			ownerID:         OwnerID("owner-001"),
			ledgerAccountID: AccountID("account-001"),
			currency:        Currency{code: "BR"},
			err:             ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wallet, err := NewWallet(
				tt.id,
				tt.ownerID,
				tt.ledgerAccountID,
				tt.currency,
			)

			if !errors.Is(err, tt.err) {
				t.Fatalf(
					"NewWallet() error = %v, want %v",
					err,
					tt.err,
				)
			}

			if tt.err != nil {
				return
			}

			if wallet.ID() != tt.id {
				t.Errorf(
					"ID() = %q, want %q",
					wallet.ID(),
					tt.id,
				)
			}

			if wallet.OwnerID() != tt.ownerID {
				t.Errorf(
					"OwnerID() = %q, want %q",
					wallet.OwnerID(),
					tt.ownerID,
				)
			}

			if wallet.LedgerAccountID() != tt.ledgerAccountID {
				t.Errorf(
					"LedgerAccountID() = %q, want %q",
					wallet.LedgerAccountID(),
					tt.ledgerAccountID,
				)
			}

			if !wallet.Currency().Equal(tt.currency) {
				t.Errorf(
					"Currency() = %q, want %q",
					wallet.Currency(),
					tt.currency,
				)
			}

			if !wallet.IsOpen() {
				t.Error("new wallet should be open")
			}
		})
	}
}

func TestWallet_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		wallet Wallet
		err    error
	}{
		{
			name:   "accepts a valid wallet",
			wallet: newValidTestWallet(t),
		},
		{
			name: "rejects an invalid status",
			wallet: Wallet{
				id:              WalletID("wallet-001"),
				ownerID:         OwnerID("owner-001"),
				ledgerAccountID: AccountID("account-001"),
				currency:        fakeValidCurrency("BRL"),
				status:          WalletStatus("blocked"),
			},
			err: ErrInvalidWalletStatus,
		},
		{
			name: "rejects an empty owner ID",
			wallet: Wallet{
				id:              WalletID("wallet-001"),
				ownerID:         OwnerID(""),
				ledgerAccountID: AccountID("account-001"),
				currency:        fakeValidCurrency("BRL"),
				status:          WalletStatusOpen,
			},
			err: ErrEmptyWalletOwnerID,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.wallet.Validate()

			if !errors.Is(err, tt.err) {
				t.Fatalf(
					"Validate() error = %v, want %v",
					err,
					tt.err,
				)
			}
		})
	}
}

func TestWallet_CanOperate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupWallet func(t *testing.T) Wallet
		err         error
	}{
		{
			name: "allows operations on an open wallet",
			setupWallet: func(t *testing.T) Wallet {
				return newValidTestWallet(t)
			},
		},
		{
			name: "rejects operations on a suspended wallet",
			setupWallet: func(t *testing.T) Wallet {
				wallet := newValidTestWallet(t)

				if err := wallet.Suspend(); err != nil {
					t.Fatalf("suspend wallet: %v", err)
				}

				return wallet
			},
			err: ErrWalletSuspended,
		},
		{
			name: "rejects operations on a closed wallet",
			setupWallet: func(t *testing.T) Wallet {
				wallet := newValidTestWallet(t)

				if err := wallet.Close(); err != nil {
					t.Fatalf("close wallet: %v", err)
				}

				return wallet
			},
			err: ErrWalletClosed,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wallet := tt.setupWallet(t)
			err := wallet.CanOperate()

			if !errors.Is(err, tt.err) {
				t.Fatalf(
					"CanOperate() error = %v, want %v",
					err,
					tt.err,
				)
			}
		})
	}
}

func TestWallet_CanOperateWithCurrency(t *testing.T) {
	t.Parallel()

	wallet := newValidTestWallet(t)

	tests := []struct {
		name     string
		currency Currency
		err      error
	}{
		{
			name:     "allows the wallet currency",
			currency: fakeValidCurrency("BRL"),
		},
		{
			name:     "rejects a different currency",
			currency: fakeValidCurrency("USD"),
			err:      ErrCurrencyMismatch,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := wallet.CanOperateWithCurrency(tt.currency)

			if !errors.Is(err, tt.err) {
				t.Fatalf(
					"CanOperateWithCurrency() error = %v, want %v",
					err,
					tt.err,
				)
			}
		})
	}
}

func TestWallet_Suspend(t *testing.T) {
	t.Parallel()

	t.Run("suspends an open wallet", func(t *testing.T) {
		t.Parallel()

		wallet := newValidTestWallet(t)

		if err := wallet.Suspend(); err != nil {
			t.Fatalf("Suspend() returned an error: %v", err)
		}

		if !wallet.IsSuspended() {
			t.Error("wallet should be suspended")
		}
	})

	t.Run("rejects suspending an already suspended wallet", func(t *testing.T) {
		t.Parallel()

		wallet := newValidTestWallet(t)

		if err := wallet.Suspend(); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}

		err := wallet.Suspend()
		if !errors.Is(err, ErrWalletAlreadySuspended) {
			t.Fatalf(
				"Suspend() error = %v, want %v",
				err,
				ErrWalletAlreadySuspended,
			)
		}
	})

	t.Run("rejects suspending a closed wallet", func(t *testing.T) {
		t.Parallel()

		wallet := newValidTestWallet(t)

		if err := wallet.Close(); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}

		err := wallet.Suspend()
		if !errors.Is(err, ErrWalletClosed) {
			t.Fatalf(
				"Suspend() error = %v, want %v",
				err,
				ErrWalletClosed,
			)
		}
	})
}

func TestWallet_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes an open wallet", func(t *testing.T) {
		t.Parallel()

		wallet := newValidTestWallet(t)

		if err := wallet.Close(); err != nil {
			t.Fatalf("Close() returned an error: %v", err)
		}

		if !wallet.IsClosed() {
			t.Error("wallet should be closed")
		}
	})

	t.Run("closes a suspended wallet", func(t *testing.T) {
		t.Parallel()

		wallet := newValidTestWallet(t)

		if err := wallet.Suspend(); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}

		if err := wallet.Close(); err != nil {
			t.Fatalf("Close() returned an error: %v", err)
		}

		if !wallet.IsClosed() {
			t.Error("wallet should be closed")
		}
	})

	t.Run("rejects closing an already closed wallet", func(t *testing.T) {
		t.Parallel()

		wallet := newValidTestWallet(t)

		if err := wallet.Close(); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}

		err := wallet.Close()
		if !errors.Is(err, ErrWalletAlreadyClosed) {
			t.Fatalf(
				"Close() error = %v, want %v",
				err,
				ErrWalletAlreadyClosed,
			)
		}
	})
}

func newValidTestWallet(t *testing.T) Wallet {
	t.Helper()

	wallet, err := NewWallet(
		WalletID("wallet-001"),
		OwnerID("owner-001"),
		AccountID("account-001"),
		fakeValidCurrency("BRL"),
	)
	if err != nil {
		t.Fatalf("failed to create valid wallet: %v", err)
	}

	return wallet
}
