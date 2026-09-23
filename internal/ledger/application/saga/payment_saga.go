package saga

import (
	"context"
	"fmt"
	"strings"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/observability"
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
	metrics    *observability.Metrics
}

func NewPaymentSagaOrchestrator(
	authorizer HoldAuthorizer,
	antiFraud AntiFraudChecker,
	capturer HoldCapturer,
	releaser HoldReleaser,
	gateway PaymentGateway,
	metrics ...*observability.Metrics,
) PaymentSagaOrchestrator {
	var collector *observability.Metrics
	if len(metrics) > 0 {
		collector = metrics[0]
	}

	return PaymentSagaOrchestrator{
		authorizer: authorizer,
		antiFraud:  antiFraud,
		capturer:   capturer,
		releaser:   releaser,
		gateway:    gateway,
		metrics:    collector,
	}
}

func (s PaymentSagaOrchestrator) Execute(
	ctx context.Context,
	cmd ProcessPaymentCommand,
) (result ProcessPaymentResult, executionErr error) {
	defer func() {
		if s.metrics == nil {
			return
		}
		status := "success"
		if executionErr != nil {
			status = "failed"
		}
		s.metrics.RecordSaga("checkout", status)
	}()

	ctx, span := observability.StartSpan(ctx, "saga.payment_orchestration")
	defer span.End()

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

	holdCtx, holdSpan := observability.StartSpan(ctx, "saga.step1_hold_authorization")
	hold, err := s.authorizer.Execute(holdCtx, wallet.CreateHoldCommand{
		ID:               cmd.HoldID,
		WalletID:         cmd.WalletID,
		AmountMinorUnits: cmd.AmountMinorUnits,
		ExpiresAt:        cmd.HoldExpiresAt,
		IdempotencyKey:   cmd.IdempotencyKey,
		RequestHash:      cmd.RequestHash,
	})
	holdSpan.End()
	if err != nil {
		return ProcessPaymentResult{}, fmt.Errorf("step 1 hold authorization: %w", err)
	}
	if hold.IsCaptured() {
		// A retry after the response was lost must not call anti-fraud or the
		// external gateway again. The deterministic command IDs and request
		// hash guarantee that this is the same checkout attempt.
		return ProcessPaymentResult{
			HoldID:        hold.ID(),
			TransactionID: cmd.TransactionID,
			Status:        "COMPLETED",
		}, nil
	}

	fraudCtx, fraudSpan := observability.StartSpan(ctx, "saga.step2_antifraud_evaluation")
	fraudResp, err := s.antiFraud.Evaluate(fraudCtx, AntiFraudRequest{
		WalletID:         cmd.WalletID,
		AmountMinorUnits: cmd.AmountMinorUnits,
		Currency:         cmd.Currency,
		Recipient:        cmd.Recipient,
	})
	fraudSpan.End()
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

	gwCtx, gwSpan := observability.StartSpan(ctx, "saga.step3_gateway_processing")
	gatewayResp, err := s.gateway.ProcessPayment(gwCtx, ExternalPaymentRequest{
		PaymentID:        cmd.IdempotencyKey,
		AmountMinorUnits: cmd.AmountMinorUnits,
		Currency:         cmd.Currency,
		Recipient:        cmd.Recipient,
	})
	gwSpan.End()

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

	capCtx, capSpan := observability.StartSpan(ctx, "saga.step4_hold_capture")
	captureResult, err := s.capturer.Execute(capCtx, wallet.CaptureHoldCommand{
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
	capSpan.End()
	if err != nil {
		compensateErr := s.compensateHold(ctx, hold.ID())
		if compensateErr != nil {
			return ProcessPaymentResult{}, fmt.Errorf("%w: %v (capture err: %v)", ErrSagaCompensationFailed, compensateErr, err)
		}
		return ProcessPaymentResult{}, fmt.Errorf("step 4 capture hold: %w", err)
	}

	return ProcessPaymentResult{
		HoldID:               hold.ID(),
		TransactionID:        captureResult.Transaction.ID(),
		GatewayTransactionID: gatewayResp.TransactionID,
		Status:               "COMPLETED",
	}, nil
}

func (s PaymentSagaOrchestrator) compensateHold(
	ctx context.Context,
	holdID domain.HoldID,
) error {
	compCtx, compSpan := observability.StartSpan(ctx, "saga.compensation_release_hold")
	defer compSpan.End()

	_, err := s.releaser.Execute(compCtx, holdID)
	return err
}
