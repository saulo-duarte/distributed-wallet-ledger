package domain

import (
	"errors"
	"math"
	"testing"
)

func FuzzNewMoney(f *testing.F) {
	f.Add("BRL", int64(0))
	f.Add("BRL", int64(10000))
	f.Add("USD", int64(math.MaxInt64))
	f.Add("BRL", int64(-1))
	f.Add("invalid", int64(100))

	f.Fuzz(func(t *testing.T, currencyCode string, amount int64) {
		currency, currencyErr := NewCurrency(currencyCode)
		if currencyErr != nil {
			return
		}

		money, err := NewMoney(currency, amount)
		if amount < 0 {
			if !errors.Is(err, ErrAmountMustNotBeNegative) {
				t.Fatalf("NewMoney() error = %v, want %v", err, ErrAmountMustNotBeNegative)
			}
			return
		}

		if err != nil {
			t.Fatalf("NewMoney() rejected a non-negative amount: %v", err)
		}
		if money.AmountMinorUnits() != amount {
			t.Fatalf("AmountMinorUnits() = %d, want %d", money.AmountMinorUnits(), amount)
		}
		if !money.Currency().Equal(currency) {
			t.Fatalf("Money currency = %q, want %q", money.Currency(), currency)
		}
	})
}

func FuzzMoneyAdd(f *testing.F) {
	f.Add("BRL", int64(100), int64(50))
	f.Add("BRL", int64(0), int64(0))
	f.Add("BRL", int64(math.MaxInt64), int64(1))
	f.Add("BRL", int64(-1), int64(100))
	f.Add("invalid", int64(100), int64(100))

	f.Fuzz(func(t *testing.T, currencyCode string, firstAmount, secondAmount int64) {
		currency, currencyErr := NewCurrency(currencyCode)
		if currencyErr != nil {
			return
		}

		first, firstErr := NewMoney(currency, firstAmount)
		second, secondErr := NewMoney(currency, secondAmount)
		if firstErr != nil || secondErr != nil {
			return
		}

		result, err := first.Add(second)
		overflows := math.MaxInt64-firstAmount < secondAmount
		if overflows {
			if !errors.Is(err, ErrAmountOverflow) {
				t.Fatalf("Add() error = %v, want %v", err, ErrAmountOverflow)
			}
			return
		}

		if err != nil {
			t.Fatalf("Add() returned an unexpected error: %v", err)
		}

		want := firstAmount + secondAmount
		if result.AmountMinorUnits() != want {
			t.Fatalf("Add() amount = %d, want %d", result.AmountMinorUnits(), want)
		}
		if !result.Currency().Equal(currency) {
			t.Fatalf("Add() currency = %q, want %q", result.Currency(), currency)
		}
	})
}
