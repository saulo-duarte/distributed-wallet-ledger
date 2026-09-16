package domain

import (
	"errors"
	"math"
	"testing"
)

func TestNewMoney(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")

	tests := []struct {
		name         string
		currency     Currency
		amount       int64
		wantErr      error
		wantCurrency Currency
		wantAmount   int64
	}{
		{
			name:         "creates a positive amount",
			currency:     brl,
			amount:       10000,
			wantCurrency: brl,
			wantAmount:   10000,
		},
		{
			name:         "accepts a zero amount",
			currency:     brl,
			amount:       0,
			wantCurrency: brl,
			wantAmount:   0,
		},
		{
			name:     "rejects a negative amount",
			currency: brl,
			amount:   -1,
			wantErr:  ErrAmountMustNotBeNegative,
		},
		{
			name:     "rejects an invalid currency",
			currency: Currency{code: "BR"},
			amount:   10000,
			wantErr:  ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			money, err := NewMoney(tt.currency, tt.amount)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewMoney() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if !money.Currency().Equal(tt.wantCurrency) {
				t.Errorf("Currency() = %q, want %q", money.Currency(), tt.wantCurrency)
			}

			if money.AmountMinorUnits() != tt.wantAmount {
				t.Errorf("AmountMinorUnits() = %d, want %d", money.AmountMinorUnits(), tt.wantAmount)
			}
		})
	}
}

func TestMoney_IsZero(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")

	tests := []struct {
		name   string
		amount int64
		want   bool
	}{
		{name: "zero", amount: 0, want: true},
		{name: "positive", amount: 1, want: false},
		{name: "negative", amount: -1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			money := Money{currency: brl, amountMinorUnits: tt.amount}
			if got := money.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoney_IsPositive(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")

	tests := []struct {
		name   string
		amount int64
		want   bool
	}{
		{name: "zero", amount: 0, want: false},
		{name: "positive", amount: 1, want: true},
		{name: "negative", amount: -1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			money := Money{currency: brl, amountMinorUnits: tt.amount}
			if got := money.IsPositive(); got != tt.want {
				t.Errorf("IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoney_Add(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	usd := fakeValidCurrency("USD")

	tests := []struct {
		name       string
		first      Money
		second     Money
		wantAmount int64
		wantErr    error
	}{
		{
			name:       "adds amounts with the same currency",
			first:      Money{currency: brl, amountMinorUnits: 10000},
			second:     Money{currency: brl, amountMinorUnits: 5000},
			wantAmount: 15000,
		},
		{
			name:       "adds zero",
			first:      Money{currency: brl, amountMinorUnits: 10000},
			second:     Money{currency: brl, amountMinorUnits: 0},
			wantAmount: 10000,
		},
		{
			name:    "rejects different currencies",
			first:   Money{currency: brl, amountMinorUnits: 10000},
			second:  Money{currency: usd, amountMinorUnits: 10000},
			wantErr: ErrCurrencyMismatch,
		},
		{
			name:    "rejects overflow",
			first:   Money{currency: brl, amountMinorUnits: math.MaxInt64},
			second:  Money{currency: brl, amountMinorUnits: 1},
			wantErr: ErrAmountOverflow,
		},
		{
			name:    "rejects a negative operand",
			first:   Money{currency: brl, amountMinorUnits: 10000},
			second:  Money{currency: brl, amountMinorUnits: -1},
			wantErr: ErrAmountMustNotBeNegative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.first.Add(tt.second)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Add() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if result.AmountMinorUnits() != tt.wantAmount {
				t.Errorf("AmountMinorUnits() = %d, want %d", result.AmountMinorUnits(), tt.wantAmount)
			}

			if !result.Currency().Equal(tt.first.Currency()) {
				t.Errorf("Currency() = %q, want %q", result.Currency(), tt.first.Currency())
			}
		})
	}
}

func TestMoney_Equal(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	usd := fakeValidCurrency("USD")

	tests := []struct {
		name   string
		first  Money
		second Money
		want   bool
	}{
		{
			name:   "same currency and same amount",
			first:  Money{currency: brl, amountMinorUnits: 10000},
			second: Money{currency: brl, amountMinorUnits: 10000},
			want:   true,
		},
		{
			name:   "different amounts",
			first:  Money{currency: brl, amountMinorUnits: 10000},
			second: Money{currency: brl, amountMinorUnits: 5000},
			want:   false,
		},
		{
			name:   "different currencies",
			first:  Money{currency: brl, amountMinorUnits: 10000},
			second: Money{currency: usd, amountMinorUnits: 10000},
			want:   false,
		},
		{
			name:   "two zero amounts",
			first:  Money{currency: brl, amountMinorUnits: 0},
			second: Money{currency: brl, amountMinorUnits: 0},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.first.Equal(tt.second); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoney_Validate(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")

	tests := []struct {
		name    string
		money   Money
		wantErr error
	}{
		{
			name:    "valid amount",
			money:   Money{currency: brl, amountMinorUnits: 10000},
			wantErr: nil,
		},
		{
			name:    "zero value",
			money:   Money{},
			wantErr: ErrInvalidCurrency,
		},
		{
			name:    "negative amount",
			money:   Money{currency: brl, amountMinorUnits: -1},
			wantErr: ErrAmountMustNotBeNegative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.money.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
