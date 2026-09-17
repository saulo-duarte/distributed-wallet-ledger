package wallet

import (
	"context"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type ExpireHoldUseCase struct {
	holds HoldRepository
}

func NewExpireHoldUseCase(holds HoldRepository) ExpireHoldUseCase {
	return ExpireHoldUseCase{holds: holds}
}

func (uc ExpireHoldUseCase) Execute(
	ctx context.Context,
	holdID domain.HoldID,
) (domain.Hold, error) {
	if holdID.IsZero() {
		return domain.Hold{}, domain.ErrInvalidID
	}

	hold, err := uc.holds.GetByID(ctx, holdID)
	if err != nil {
		return domain.Hold{}, err
	}

	if err := hold.Expire(time.Now().UTC()); err != nil {
		return domain.Hold{}, err
	}

	if err := uc.holds.UpdateStatus(
		ctx,
		hold.ID(),
		domain.HoldStatusExpired,
	); err != nil {
		return domain.Hold{}, err
	}

	return hold, nil
}
