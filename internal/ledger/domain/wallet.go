package domain

type WalletStatus string

const (
	WalletStatusOpen      WalletStatus = "open"
	WalletStatusSuspended WalletStatus = "suspended"
	WalletStatusClosed    WalletStatus = "closed"
)

func (s WalletStatus) Validate() error {
	switch s {
	case WalletStatusOpen,
		WalletStatusSuspended,
		WalletStatusClosed:
		return nil
	default:
		return ErrInvalidWalletStatus
	}
}

type Wallet struct {
	id              WalletID
	ownerID         OwnerID
	ledgerAccountID AccountID
	currency        Currency
	status          WalletStatus
}

func NewWallet(
	id WalletID,
	ownerID OwnerID,
	ledgerAccountID AccountID,
	currency Currency,
) (Wallet, error) {
	return ReconstituteWallet(
		id,
		ownerID,
		ledgerAccountID,
		currency,
		WalletStatusOpen,
	)
}

func ReconstituteWallet(
	id WalletID,
	ownerID OwnerID,
	ledgerAccountID AccountID,
	currency Currency,
	status WalletStatus,
) (Wallet, error) {
	wallet := Wallet{
		id:              id,
		ownerID:         ownerID,
		ledgerAccountID: ledgerAccountID,
		currency:        currency,
		status:          status,
	}

	if err := wallet.Validate(); err != nil {
		return Wallet{}, err
	}

	return wallet, nil
}

func (w Wallet) ID() WalletID {
	return w.id
}

func (w Wallet) OwnerID() OwnerID {
	return w.ownerID
}

func (w Wallet) LedgerAccountID() AccountID {
	return w.ledgerAccountID
}

func (w Wallet) Currency() Currency {
	return w.currency
}

func (w Wallet) Status() WalletStatus {
	return w.status
}

func (w Wallet) IsOpen() bool {
	return w.status == WalletStatusOpen
}

func (w Wallet) IsSuspended() bool {
	return w.status == WalletStatusSuspended
}

func (w Wallet) IsClosed() bool {
	return w.status == WalletStatusClosed
}

func (w Wallet) CanOperate() error {
	if err := w.Validate(); err != nil {
		return err
	}

	switch w.status {
	case WalletStatusSuspended:
		return ErrWalletSuspended
	case WalletStatusClosed:
		return ErrWalletClosed
	default:
		return nil
	}
}

func (w Wallet) CanOperateWithCurrency(
	currency Currency,
) error {
	if err := w.CanOperate(); err != nil {
		return err
	}

	if !w.currency.Equal(currency) {
		return ErrCurrencyMismatch
	}

	return nil
}

func (w *Wallet) Suspend() error {
	if err := w.Validate(); err != nil {
		return err
	}

	if w.status == WalletStatusClosed {
		return ErrWalletClosed
	}

	if w.status == WalletStatusSuspended {
		return ErrWalletAlreadySuspended
	}

	w.status = WalletStatusSuspended
	return nil
}

func (w *Wallet) Close() error {
	if err := w.Validate(); err != nil {
		return err
	}

	if w.status == WalletStatusClosed {
		return ErrWalletAlreadyClosed
	}

	w.status = WalletStatusClosed
	return nil
}

func (w Wallet) Validate() error {
	if w.id.IsZero() {
		return ErrInvalidID
	}

	if w.ownerID.IsZero() {
		return ErrEmptyWalletOwnerID
	}

	if w.ledgerAccountID.IsZero() {
		return ErrInvalidID
	}

	if err := w.currency.Validate(); err != nil {
		return err
	}

	if err := w.status.Validate(); err != nil {
		return err
	}

	return nil
}
