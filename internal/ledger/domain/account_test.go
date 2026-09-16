package domain

import (
	"errors"
	"testing"
)

func TestAccountStatus_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  AccountStatus
		wantErr error
	}{
		{
			name:    "open status is valid",
			status:  AccountStatusOpen,
			wantErr: nil,
		},
		{
			name:    "closed status is valid",
			status:  AccountStatusClosed,
			wantErr: nil,
		},
		{
			name:    "unknown status is invalid",
			status:  AccountStatus("suspended"),
			wantErr: ErrInvalidAccountStatus,
		},
		{
			name:    "empty status is invalid",
			status:  AccountStatus(""),
			wantErr: ErrInvalidAccountStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.status.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAccount(t *testing.T) {
	t.Parallel()

	validID := fakeValidAccountID()
	validCurrency := fakeValidCurrency("BRL")

	tests := []struct {
		name         string
		id           AccountID
		code         string
		accountName  string
		currency     Currency
		wantErr      error
		expectedCode string
		expectedName string
	}{
		{
			name:         "creates an account and trims whitespace",
			id:           validID,
			code:         "  ACC-001  ",
			accountName:  "  Primary Account  ",
			currency:     validCurrency,
			wantErr:      nil,
			expectedCode: "ACC-001",
			expectedName: "Primary Account",
		},
		{
			name:        "rejects an empty ID",
			id:          fakeZeroID(),
			code:        "ACC-001",
			accountName: "Conta",
			currency:    validCurrency,
			wantErr:     ErrInvalidID,
		},
		{
			name:        "rejects an empty code after trimming",
			id:          validID,
			code:        "   ",
			accountName: "Conta",
			currency:    validCurrency,
			wantErr:     ErrEmptyAccountCode,
		},
		{
			name:        "rejects an empty name after trimming",
			id:          validID,
			code:        "ACC-001",
			accountName: "   ",
			currency:    validCurrency,
			wantErr:     ErrEmptyAccountName,
		},
		{
			name:        "rejects an invalid currency",
			id:          validID,
			code:        "ACC-001",
			accountName: "Conta",
			currency:    Currency{code: "BR"},
			wantErr:     ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			acc, err := NewAccount(tt.id, tt.code, tt.accountName, tt.currency)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewAccount() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if acc.Code() != tt.expectedCode {
					t.Errorf("Code() = %q, want %q", acc.Code(), tt.expectedCode)
				}
				if acc.Name() != tt.expectedName {
					t.Errorf("Name() = %q, want %q", acc.Name(), tt.expectedName)
				}
				if !acc.IsOpen() {
					t.Errorf("IsOpen() = false, a new account should be open")
				}
			}
		})
	}
}

func TestAccount_Validate(t *testing.T) {
	t.Parallel()

	account := Account{
		id:       fakeValidAccountID(),
		code:     "ACC-001",
		name:     "Conta Teste",
		currency: fakeValidCurrency("BRL"),
		status:   AccountStatus("suspended"),
	}

	err := account.Validate()
	if !errors.Is(err, ErrInvalidAccountStatus) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidAccountStatus)
	}
}

func TestAccount_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes an open account successfully", func(t *testing.T) {
		t.Parallel()
		acc := newValidTestAccount(t)

		err := acc.Close()
		if err != nil {
			t.Fatalf("Close() returned an unexpected error: %v", err)
		}
		if !acc.IsClosed() {
			t.Errorf("IsClosed() = false, want true after Close()")
		}
	})

	t.Run("rejects closing an already closed account", func(t *testing.T) {
		t.Parallel()
		acc := newValidTestAccount(t)

		if err := acc.Close(); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}

		err := acc.Close()
		if !errors.Is(err, ErrAccountAlreadyClosed) {
			t.Fatalf("Close() error = %v, want %v", err, ErrAccountAlreadyClosed)
		}
	})
}

func TestAccount_CanReceivePosting(t *testing.T) {
	t.Parallel()

	currencyBRL := fakeValidCurrency("BRL")
	currencyUSD := fakeValidCurrency("USD")

	tests := []struct {
		name            string
		setupAccount    func(t *testing.T) Account
		postingCurrency Currency
		wantErr         error
	}{
		{
			name: "succeeds with the same currency and an open account",
			setupAccount: func(t *testing.T) Account {
				return newValidTestAccountWithCurrency(t, currencyBRL)
			},
			postingCurrency: currencyBRL,
			wantErr:         nil,
		},
		{
			name: "rejects a closed account",
			setupAccount: func(t *testing.T) Account {
				acc := newValidTestAccountWithCurrency(t, currencyBRL)
				_ = acc.Close()
				return acc
			},
			postingCurrency: currencyBRL,
			wantErr:         ErrAccountClosed,
		},
		{
			name: "rejects a currency mismatch",
			setupAccount: func(t *testing.T) Account {
				return newValidTestAccountWithCurrency(t, currencyBRL)
			},
			postingCurrency: currencyUSD,
			wantErr:         ErrCurrencyMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			acc := tt.setupAccount(t)
			err := acc.CanReceivePosting(tt.postingCurrency)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CanReceivePosting() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func newValidTestAccount(t *testing.T) Account {
	t.Helper()
	return newValidTestAccountWithCurrency(t, fakeValidCurrency("BRL"))
}

func newValidTestAccountWithCurrency(t *testing.T, curr Currency) Account {
	t.Helper()
	acc, err := NewAccount(fakeValidAccountID(), "ACC-01", "Conta Teste", curr)
	if err != nil {
		t.Fatalf("failed to create a valid test account: %v", err)
	}
	return acc
}
