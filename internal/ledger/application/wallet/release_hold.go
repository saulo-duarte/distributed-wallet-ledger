package wallet

import (
	"context"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type ReleaseHoldUseCase struct {
	holds HoldRepository
}

func NewReleaseHoldUseCase(holds HoldRepository) ReleaseHoldUseCase {
	return ReleaseHoldUseCase{holds: holds}
}

func (uc ReleaseHoldUseCase) Execute(
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

	if err := hold.Release(time.Now().UTC()); err != nil {
		return domain.Hold{}, err
	}

	if err := uc.holds.UpdateStatus(
		ctx,
		hold.ID(),
		domain.HoldStatusReleased,
	); err != nil {
		return domain.Hold{}, err
	}

	return hold, nil
}
