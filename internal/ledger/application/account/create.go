package account

import (
	"context"
	"financial-ledger/internal/ledger/domain"
)

type CreateAccountCommand struct {
	ID           domain.AccountID
	Code         string
	Name         string
	CurrencyCode string
}

type CreateAccountUseCase struct {
	accounts AccountRepository
}

func NewCreateAccountUseCase(
	accounts AccountRepository,
) CreateAccountUseCase {
	return CreateAccountUseCase{
		accounts: accounts,
	}
}

func (uc CreateAccountUseCase) Execute(
	ctx context.Context,
	command CreateAccountCommand,
) (domain.Account, error) {
	currency, err := domain.NewCurrency(command.CurrencyCode)
	if err != nil {
		return domain.Account{}, err
	}

	account, err := domain.NewAccount(
		command.ID,
		command.Code,
		command.Name,
		currency,
	)
	if err != nil {
		return domain.Account{}, err
	}

	if err := uc.accounts.Create(ctx, account); err != nil {
		return domain.Account{}, err
	}

	return account, nil
}
