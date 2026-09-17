package wallet

import (
	"context"
	"strings"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type CreateHoldCommand struct {
	ID               domain.HoldID
	WalletID         domain.WalletID
	AmountMinorUnits int64
	ExpiresAt        time.Time
	IdempotencyKey   string
	RequestHash      string
}

type CreateHoldUseCase struct {
	wallets WalletReader
	holds   HoldRepository
}

func NewCreateHoldUseCase(
	wallets WalletReader,
	holds HoldRepository,
) CreateHoldUseCase {
	return CreateHoldUseCase{
		wallets: wallets,
		holds:   holds,
	}
}

func (uc CreateHoldUseCase) Execute(
	ctx context.Context,
	command CreateHoldCommand,
) (domain.Hold, error) {
	if command.ID.IsZero() || command.WalletID.IsZero() {
		return domain.Hold{}, domain.ErrInvalidID
	}

	if command.AmountMinorUnits < 0 {
		return domain.Hold{}, domain.ErrAmountMustNotBeNegative
	}

	if command.AmountMinorUnits == 0 {
		return domain.Hold{}, domain.ErrAmountMustBePositive
	}

	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return domain.Hold{}, ErrEmptyIdempotencyKey
	}

	if strings.TrimSpace(command.RequestHash) == "" {
		return domain.Hold{}, ErrEmptyRequestHash
	}

	foundWallet, err := uc.wallets.GetByID(ctx, command.WalletID)
	if err != nil {
		return domain.Hold{}, err
	}

	if err := foundWallet.CanOperate(); err != nil {
		return domain.Hold{}, err
	}

	money, err := domain.NewMoney(
		foundWallet.Currency(),
		command.AmountMinorUnits,
	)
	if err != nil {
		return domain.Hold{}, err
	}

	hold, err := domain.NewHold(
		command.ID,
		command.WalletID,
		money,
		time.Now().UTC(),
		command.ExpiresAt,
	)
	if err != nil {
		return domain.Hold{}, err
	}

	return uc.holds.Authorize(
		ctx,
		hold,
		strings.TrimSpace(command.IdempotencyKey),
		strings.TrimSpace(command.RequestHash),
	)
}
