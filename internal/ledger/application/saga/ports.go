package saga

import (
	"context"
	"errors"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

var (
	ErrExternalPaymentFailed  = errors.New("external payment processing failed")
	ErrAntiFraudRejected      = errors.New("payment rejected by anti-fraud check")
	ErrSagaCompensationFailed = errors.New("saga compensation failed to release hold")
)

type AntiFraudRequest struct {
	WalletID         domain.WalletID
	AmountMinorUnits int64
	Currency         domain.Currency
	Recipient        string
}

type AntiFraudResponse struct {
	Approved     bool
	RiskScore    int
	RejectReason string
}

type AntiFraudChecker interface {
	Evaluate(
		ctx context.Context,
		req AntiFraudRequest,
	) (AntiFraudResponse, error)
}

type ExternalPaymentRequest struct {
	PaymentID        string
	AmountMinorUnits int64
	Currency         domain.Currency
	Recipient        string
}

type ExternalPaymentResponse struct {
	Success       bool
	TransactionID string
	ErrorMessage  string
}

type PaymentGateway interface {
	ProcessPayment(
		ctx context.Context,
		req ExternalPaymentRequest,
	) (ExternalPaymentResponse, error)
}

type HoldAuthorizer interface {
	Execute(
		ctx context.Context,
		command wallet.CreateHoldCommand,
	) (domain.Hold, error)
}

type HoldCapturer interface {
	Execute(
		ctx context.Context,
		command wallet.CaptureHoldCommand,
	) (wallet.CaptureHoldResult, error)
}
type HoldReleaser interface {
	Execute(
		ctx context.Context,
		holdID domain.HoldID,
	) (domain.Hold, error)
}
