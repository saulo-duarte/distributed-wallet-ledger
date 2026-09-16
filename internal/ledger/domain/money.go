package domain

import "math"

type Money struct {
	currency         Currency
	amountMinorUnits int64
}

func NewMoney(currency Currency, amount int64) (Money, error) {
	m := Money{
		currency:         currency,
		amountMinorUnits: amount,
	}

	if err := m.Validate(); err != nil {
		return Money{}, err
	}

	return m, nil
}

func (m Money) Currency() Currency {
	return m.currency
}

func (m Money) AmountMinorUnits() int64 {
	return m.amountMinorUnits
}

func (m Money) IsZero() bool {
	if m.amountMinorUnits == 0 {
		return true
	}
	return false
}

func (m Money) IsPositive() bool {
	if m.amountMinorUnits > 0 {
		return true
	}
	return false
}

func (m Money) Add(other Money) (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}

	if err := other.Validate(); err != nil {
		return Money{}, err
	}

	if !m.currency.Equal(other.currency) {
		return Money{}, ErrCurrencyMismatch
	}

	if math.MaxInt64-m.amountMinorUnits < other.amountMinorUnits {
		return Money{}, ErrAmountOverflow
	}

	return Money{
		currency:         m.currency,
		amountMinorUnits: m.amountMinorUnits + other.amountMinorUnits,
	}, nil
}

func (m Money) Equal(other Money) bool {
	if m.currency != other.currency {
		return false
	}

	if m.amountMinorUnits != other.amountMinorUnits {
		return false
	}

	return true
}

func (m Money) Validate() error {
	if err := m.currency.Validate(); err != nil {
		return err
	}

	if m.amountMinorUnits < 0 {
		return ErrAmountMustNotBeNegative
	}

	return nil
}
