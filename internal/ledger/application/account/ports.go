package account

import (
	"context"
	"financial-ledger/internal/ledger/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, account domain.Account) error
}
