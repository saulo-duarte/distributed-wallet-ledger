package domain

import "strings"

type AccountStatus string

const (
	AccountStatusOpen   AccountStatus = "open"
	AccountStatusClosed AccountStatus = "closed"
)

func (s AccountStatus) Validate() error {
	switch s {
	case AccountStatusOpen, AccountStatusClosed:
		return nil
	default:
		return ErrInvalidAccountStatus
	}
}

type Account struct {
	id       AccountID
	code     string
	name     string
	currency Currency
	status   AccountStatus
}

func NewAccount(
	id AccountID,
	code string,
	name string,
	currency Currency,
) (Account, error) {
	account := Account{
		id:       id,
		code:     strings.TrimSpace(code),
		name:     strings.TrimSpace(name),
		currency: currency,
		status:   AccountStatusOpen,
	}

	if err := account.Validate(); err != nil {
		return Account{}, err
	}

	return account, nil
}

func (a Account) ID() AccountID {
	return a.id
}

func (a Account) Code() string {
	return a.code
}

func (a Account) Name() string {
	return a.name
}

func (a Account) Currency() Currency {
	return a.currency
}

func (a Account) Status() AccountStatus {
	return a.status
}

func (a Account) IsOpen() bool {
	return a.status == AccountStatusOpen
}

func (a Account) IsClosed() bool {
	return a.status == AccountStatusClosed
}

func (a Account) CanReceivePosting(currency Currency) error {
	if err := a.Validate(); err != nil {
		return err
	}

	if a.status == AccountStatusClosed {
		return ErrAccountClosed
	}

	if !a.currency.Equal(currency) {
		return ErrCurrencyMismatch
	}

	return nil
}

func (a *Account) Close() error {
	if a.status == AccountStatusClosed {
		return ErrAccountAlreadyClosed
	}

	a.status = AccountStatusClosed
	return nil
}

func (a Account) Validate() error {
	if a.id.IsZero() {
		return ErrInvalidID
	}

	if a.code == "" {
		return ErrEmptyAccountCode
	}

	if a.name == "" {
		return ErrEmptyAccountName
	}

	if err := a.currency.Validate(); err != nil {
		return err
	}

	if err := a.status.Validate(); err != nil {
		return err
	}

	return nil
}
