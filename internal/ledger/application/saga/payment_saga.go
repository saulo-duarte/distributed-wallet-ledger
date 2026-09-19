package saga

import (
	"context"
	"fmt"
	"strings"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

type ProcessPaymentCommand struct {
	HoldID              domain.HoldID
	WalletID            domain.WalletID
	SettlementAccountID domain.AccountID
	TransactionID       domain.TransactionID
	JournalEntryID      domain.JournalEntryID
	WalletPostingID     domain.PostingID
	SettlementPostingID domain.PostingID
	AmountMinorUnits    int64
	Currency            domain.Currency
	Recipient           string
	Description         string
	HoldExpiresAt       time.Time
	IdempotencyKey      string
	RequestHash         string
}

type ProcessPaymentResult struct {
	HoldID               domain.HoldID
	TransactionID        domain.TransactionID
	GatewayTransactionID string
	Status               string
}

type PaymentSagaOrchestrator struct {
	authorizer HoldAuthorizer
	antiFraud  AntiFraudChecker
	capturer   HoldCapturer
	releaser   HoldReleaser
	gateway    PaymentGateway
}

func NewPaymentSagaOrchestrator(
	authorizer HoldAuthorizer,
	antiFraud AntiFraudChecker,
	capturer HoldCapturer,
	releaser HoldReleaser,
	gateway PaymentGateway,
) PaymentSagaOrchestrator {
	return PaymentSagaOrchestrator{
		authorizer: authorizer,
		antiFraud:  antiFraud,
		capturer:   capturer,
		releaser:   releaser,
		gateway:    gateway,
	}
}

func (s PaymentSagaOrchestrator) Execute(
	ctx context.Context,
	cmd ProcessPaymentCommand,
) (ProcessPaymentResult, error) {
	if cmd.HoldID.IsZero() || cmd.WalletID.IsZero() || cmd.SettlementAccountID.IsZero() {
		return ProcessPaymentResult{}, domain.ErrInvalidID
	}
	if cmd.TransactionID.IsZero() || cmd.JournalEntryID.IsZero() ||
		cmd.WalletPostingID.IsZero() || cmd.SettlementPostingID.IsZero() {
		return ProcessPaymentResult{}, domain.ErrInvalidID
	}
	if cmd.AmountMinorUnits <= 0 {
		return ProcessPaymentResult{}, domain.ErrAmountMustBePositive
	}
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return ProcessPaymentResult{}, wallet.ErrEmptyIdempotencyKey
	}
	if strings.TrimSpace(cmd.RequestHash) == "" {
		return ProcessPaymentResult{}, wallet.ErrEmptyRequestHash
	}

	hold, err := s.authorizer.Execute(ctx, wallet.CreateHoldCommand{
		ID:               cmd.HoldID,
		WalletID:         cmd.WalletID,
		AmountMinorUnits: cmd.AmountMinorUnits,
		ExpiresAt:        cmd.HoldExpiresAt,
		IdempotencyKey:   cmd.IdempotencyKey,
		RequestHash:      cmd.RequestHash,
	})
	if err != nil {
		return ProcessPaymentResult{}, fmt.Errorf("step 1 hold authorization: %w", err)
	}

	fraudResp, err := s.antiFraud.Evaluate(ctx, AntiFraudRequest{
		WalletID:         cmd.WalletID,
		AmountMinorUnits: cmd.AmountMinorUnits,
		Currency:         cmd.Currency,
		Recipient:        cmd.Recipient,
	})
	if err != nil || !fraudResp.Approved {
		compensateErr := s.compensateHold(ctx, hold.ID())
		if compensateErr != nil {
			return ProcessPaymentResult{}, fmt.Errorf("%w: %v (anti-fraud err: %v)", ErrSagaCompensationFailed, compensateErr, err)
		}
		if err != nil {
			return ProcessPaymentResult{}, fmt.Errorf("%w: %v", ErrAntiFraudRejected, err)
		}
		return ProcessPaymentResult{}, fmt.Errorf("%w: %s", ErrAntiFraudRejected, fraudResp.RejectReason)
	}

	gatewayResp, err := s.gateway.ProcessPayment(ctx, ExternalPaymentRequest{
		PaymentID:        cmd.IdempotencyKey,
		AmountMinorUnits: cmd.AmountMinorUnits,
		Currency:         cmd.Currency,
		Recipient:        cmd.Recipient,
	})

	if err != nil || !gatewayResp.Success {
		compensateErr := s.compensateHold(ctx, hold.ID())
		if compensateErr != nil {
			return ProcessPaymentResult{}, fmt.Errorf("%w: %v (original gateway err: %v)", ErrSagaCompensationFailed, compensateErr, err)
		}
		if err != nil {
			return ProcessPaymentResult{}, fmt.Errorf("%w: %v", ErrExternalPaymentFailed, err)
		}
		return ProcessPaymentResult{}, fmt.Errorf("%w: %s", ErrExternalPaymentFailed, gatewayResp.ErrorMessage)
	}

	captureResult, err := s.capturer.Execute(ctx, wallet.CaptureHoldCommand{
		HoldID:              hold.ID(),
		SettlementAccountID: cmd.SettlementAccountID,
		TransactionID:       cmd.TransactionID,
		JournalEntryID:      cmd.JournalEntryID,
		WalletPostingID:     cmd.WalletPostingID,
		SettlementPostingID: cmd.SettlementPostingID,
		Description:         cmd.Description,
		IdempotencyKey:      cmd.IdempotencyKey + ":capture",
		RequestHash:         cmd.RequestHash + ":capture",
	})
	if err != nil {
		return ProcessPaymentResult{}, fmt.Errorf("step 4 hold capture: %w", err)
	}

	return ProcessPaymentResult{
		HoldID:               captureResult.Hold.ID(),
		TransactionID:        captureResult.Transaction.ID(),
		GatewayTransactionID: gatewayResp.TransactionID,
		Status:               "COMPLETED",
	}, nil
}

func (s PaymentSagaOrchestrator) compensateHold(
	ctx context.Context,
	holdID domain.HoldID,
) error {
	_, err := s.releaser.Execute(ctx, holdID)
	if err != nil {
		return err
	}
	return nil
}
