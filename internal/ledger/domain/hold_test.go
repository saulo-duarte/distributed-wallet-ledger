package domain

import (
	"errors"
	"testing"
	"time"
)

func TestHoldStatusValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status HoldStatus
		want   error
	}{
		{name: "authorized", status: HoldStatusAuthorized},
		{name: "captured", status: HoldStatusCaptured},
		{name: "released", status: HoldStatusReleased},
		{name: "expired", status: HoldStatusExpired},
		{
			name:   "unknown status",
			status: HoldStatus("pending"),
			want:   ErrInvalidHoldStatus,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.status.Validate(); !errors.Is(err, tt.want) {
				t.Fatalf(
					"Validate() error = %v, want %v",
					err,
					tt.want,
				)
			}
		})
	}
}

func TestNewHold(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)
	amount := newHoldMoney(t, 3000)

	hold, err := NewHold(
		HoldID("hold-001"),
		WalletID("wallet-001"),
		amount,
		createdAt,
		expiresAt,
	)
	if err != nil {
		t.Fatalf("NewHold() returned an unexpected error: %v", err)
	}

	if hold.ID() != HoldID("hold-001") {
		t.Fatalf("ID() = %q, want hold-001", hold.ID())
	}
	if hold.WalletID() != WalletID("wallet-001") {
		t.Fatalf("WalletID() = %q, want wallet-001", hold.WalletID())
	}
	if !hold.Amount().Equal(amount) {
		t.Fatalf("Amount() = %+v, want %+v", hold.Amount(), amount)
	}
	if hold.Status() != HoldStatusAuthorized {
		t.Fatalf("Status() = %q, want authorized", hold.Status())
	}
	if !hold.CreatedAt().Equal(createdAt) {
		t.Fatalf("CreatedAt() = %v, want %v", hold.CreatedAt(), createdAt)
	}
	if !hold.ExpiresAt().Equal(expiresAt) {
		t.Fatalf("ExpiresAt() = %v, want %v", hold.ExpiresAt(), expiresAt)
	}
	if !hold.IsAuthorized() {
		t.Fatal("new hold should be authorized")
	}
}

func TestNewHoldRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	validAmount := newHoldMoney(t, 1000)

	tests := []struct {
		name      string
		id        HoldID
		walletID  WalletID
		amount    Money
		createdAt time.Time
		expiresAt time.Time
		wantError error
	}{
		{
			name:      "empty hold id",
			id:        HoldID(""),
			walletID:  WalletID("wallet-001"),
			amount:    validAmount,
			createdAt: createdAt,
			expiresAt: createdAt.Add(time.Hour),
			wantError: ErrInvalidID,
		},
		{
			name:      "empty wallet id",
			id:        HoldID("hold-001"),
			walletID:  WalletID(""),
			amount:    validAmount,
			createdAt: createdAt,
			expiresAt: createdAt.Add(time.Hour),
			wantError: ErrInvalidID,
		},
		{
			name:      "zero amount",
			id:        HoldID("hold-001"),
			walletID:  WalletID("wallet-001"),
			amount:    newHoldMoney(t, 0),
			createdAt: createdAt,
			expiresAt: createdAt.Add(time.Hour),
			wantError: ErrAmountMustBePositive,
		},
		{
			name:      "invalid currency",
			id:        HoldID("hold-001"),
			walletID:  WalletID("wallet-001"),
			amount:    Money{currency: Currency{code: "BR"}, amountMinorUnits: 1000},
			createdAt: createdAt,
			expiresAt: createdAt.Add(time.Hour),
			wantError: ErrInvalidCurrency,
		},
		{
			name:      "missing creation time",
			id:        HoldID("hold-001"),
			walletID:  WalletID("wallet-001"),
			amount:    validAmount,
			createdAt: time.Time{},
			expiresAt: createdAt.Add(time.Hour),
			wantError: ErrInvalidHoldTimestamp,
		},
		{
			name:      "expiration before creation",
			id:        HoldID("hold-001"),
			walletID:  WalletID("wallet-001"),
			amount:    validAmount,
			createdAt: createdAt,
			expiresAt: createdAt.Add(-time.Minute),
			wantError: ErrInvalidHoldExpiration,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewHold(
				tt.id,
				tt.walletID,
				tt.amount,
				tt.createdAt,
				tt.expiresAt,
			)

			if !errors.Is(err, tt.wantError) {
				t.Fatalf("NewHold() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestHoldIsActive(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)
	hold := newTestHold(t, HoldStatusAuthorized, createdAt, expiresAt)

	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{
			name: "before expiration",
			at:   createdAt.Add(30 * time.Minute),
			want: true,
		},
		{
			name: "at expiration",
			at:   expiresAt,
			want: false,
		},
		{
			name: "after expiration",
			at:   expiresAt.Add(time.Minute),
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := hold.IsActive(tt.at); got != tt.want {
				t.Fatalf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHoldCapture(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)
	hold := newTestHold(t, HoldStatusAuthorized, createdAt, expiresAt)

	if err := hold.Capture(createdAt.Add(30 * time.Minute)); err != nil {
		t.Fatalf("Capture() returned an unexpected error: %v", err)
	}
	if !hold.IsCaptured() {
		t.Fatal("hold should be captured")
	}
}

func TestHoldCaptureRejectsExpiredHold(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)
	hold := newTestHold(t, HoldStatusAuthorized, createdAt, expiresAt)

	err := hold.Capture(expiresAt)
	if !errors.Is(err, ErrHoldExpired) {
		t.Fatalf("Capture() error = %v, want %v", err, ErrHoldExpired)
	}
	if !hold.IsAuthorized() {
		t.Fatal("expired capture attempt should not mutate hold status")
	}
}

func TestHoldRelease(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)
	hold := newTestHold(t, HoldStatusAuthorized, createdAt, expiresAt)

	if err := hold.Release(createdAt.Add(30 * time.Minute)); err != nil {
		t.Fatalf("Release() returned an unexpected error: %v", err)
	}
	if !hold.IsReleased() {
		t.Fatal("hold should be released")
	}
}

func TestHoldExpire(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)
	hold := newTestHold(t, HoldStatusAuthorized, createdAt, expiresAt)

	if err := hold.Expire(createdAt.Add(30 * time.Minute)); !errors.Is(err, ErrHoldNotExpired) {
		t.Fatalf("Expire() error = %v, want %v", err, ErrHoldNotExpired)
	}
	if !hold.IsAuthorized() {
		t.Fatal("hold should remain authorized before expiration")
	}

	if err := hold.Expire(expiresAt); err != nil {
		t.Fatalf("Expire() returned an unexpected error: %v", err)
	}
	if !hold.IsExpired() {
		t.Fatal("hold should be expired")
	}
}

func TestHoldRejectsTransitionsFromNonAuthorizedStatus(t *testing.T) {
	t.Parallel()

	createdAt := holdCreatedAt()
	expiresAt := createdAt.Add(time.Hour)

	statuses := []HoldStatus{
		HoldStatusCaptured,
		HoldStatusReleased,
		HoldStatusExpired,
	}

	for _, status := range statuses {
		status := status
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			hold := newTestHold(t, status, createdAt, expiresAt)
			at := createdAt.Add(30 * time.Minute)

			if err := hold.Capture(at); !errors.Is(err, ErrHoldNotAuthorized) {
				t.Errorf("Capture() error = %v, want %v", err, ErrHoldNotAuthorized)
			}
			if err := hold.Release(at); !errors.Is(err, ErrHoldNotAuthorized) {
				t.Errorf("Release() error = %v, want %v", err, ErrHoldNotAuthorized)
			}
			if err := hold.Expire(at); !errors.Is(err, ErrHoldNotAuthorized) {
				t.Errorf("Expire() error = %v, want %v", err, ErrHoldNotAuthorized)
			}
		})
	}
}

func newTestHold(
	t *testing.T,
	status HoldStatus,
	createdAt time.Time,
	expiresAt time.Time,
) Hold {
	t.Helper()

	hold, err := ReconstituteHold(
		HoldID("hold-001"),
		WalletID("wallet-001"),
		newHoldMoney(t, 3000),
		status,
		createdAt,
		expiresAt,
	)
	if err != nil {
		t.Fatalf("ReconstituteHold() returned an unexpected error: %v", err)
	}

	return hold
}

func newHoldMoney(t *testing.T, amount int64) Money {
	t.Helper()

	money, err := NewMoney(fakeValidCurrency("BRL"), amount)
	if err != nil {
		t.Fatalf("NewMoney() returned an unexpected error: %v", err)
	}

	return money
}

func holdCreatedAt() time.Time {
	return time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
}
