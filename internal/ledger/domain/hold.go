package domain

import "time"

type HoldStatus string

const (
	HoldStatusAuthorized HoldStatus = "authorized"
	HoldStatusCaptured   HoldStatus = "captured"
	HoldStatusReleased   HoldStatus = "released"
	HoldStatusExpired    HoldStatus = "expired"
)

func (s HoldStatus) Validate() error {
	switch s {
	case HoldStatusAuthorized,
		HoldStatusCaptured,
		HoldStatusReleased,
		HoldStatusExpired:
		return nil
	default:
		return ErrInvalidHoldStatus
	}
}

type Hold struct {
	id        HoldID
	walletID  WalletID
	amount    Money
	status    HoldStatus
	createdAt time.Time
	expiresAt time.Time
}

func NewHold(
	id HoldID,
	walletID WalletID,
	amount Money,
	createdAt time.Time,
	expiresAt time.Time,
) (Hold, error) {
	return ReconstituteHold(
		id,
		walletID,
		amount,
		HoldStatusAuthorized,
		createdAt,
		expiresAt,
	)
}

func ReconstituteHold(
	id HoldID,
	walletID WalletID,
	amount Money,
	status HoldStatus,
	createdAt time.Time,
	expiresAt time.Time,
) (Hold, error) {
	hold := Hold{
		id:        id,
		walletID:  walletID,
		amount:    amount,
		status:    status,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}

	if err := hold.Validate(); err != nil {
		return Hold{}, err
	}

	return hold, nil
}

func (h Hold) ID() HoldID {
	return h.id
}

func (h Hold) WalletID() WalletID {
	return h.walletID
}

func (h Hold) Amount() Money {
	return h.amount
}

func (h Hold) Status() HoldStatus {
	return h.status
}

func (h Hold) CreatedAt() time.Time {
	return h.createdAt
}

func (h Hold) ExpiresAt() time.Time {
	return h.expiresAt
}

func (h Hold) IsAuthorized() bool {
	return h.status == HoldStatusAuthorized
}

func (h Hold) IsCaptured() bool {
	return h.status == HoldStatusCaptured
}

func (h Hold) IsReleased() bool {
	return h.status == HoldStatusReleased
}

func (h Hold) IsExpired() bool {
	return h.status == HoldStatusExpired
}

func (h Hold) IsActive(at time.Time) bool {
	return h.status == HoldStatusAuthorized &&
		at.Before(h.expiresAt)
}

func (h *Hold) Capture(at time.Time) error {
	if err := h.Validate(); err != nil {
		return err
	}

	if h.status != HoldStatusAuthorized {
		return ErrHoldNotAuthorized
	}

	if !at.Before(h.expiresAt) {
		return ErrHoldExpired
	}

	h.status = HoldStatusCaptured
	return nil
}

func (h *Hold) Release(at time.Time) error {
	if err := h.Validate(); err != nil {
		return err
	}

	if h.status != HoldStatusAuthorized {
		return ErrHoldNotAuthorized
	}

	if !at.Before(h.expiresAt) {
		return ErrHoldExpired
	}

	h.status = HoldStatusReleased
	return nil
}

func (h *Hold) Expire(at time.Time) error {
	if err := h.Validate(); err != nil {
		return err
	}

	if h.status != HoldStatusAuthorized {
		return ErrHoldNotAuthorized
	}

	if at.Before(h.expiresAt) {
		return ErrHoldNotExpired
	}

	h.status = HoldStatusExpired
	return nil
}

func (h Hold) Validate() error {
	if h.id.IsZero() || h.walletID.IsZero() {
		return ErrInvalidID
	}

	if err := h.amount.Validate(); err != nil {
		return err
	}

	if !h.amount.IsPositive() {
		return ErrAmountMustBePositive
	}

	if err := h.status.Validate(); err != nil {
		return err
	}

	if h.createdAt.IsZero() || h.expiresAt.IsZero() {
		return ErrInvalidHoldTimestamp
	}

	if !h.expiresAt.After(h.createdAt) {
		return ErrInvalidHoldExpiration
	}

	return nil
}
