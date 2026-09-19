package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

type fakeHoldAuthorizer struct {
	hold domain.Hold
	err  error
}

func (f *fakeHoldAuthorizer) Execute(
	_ context.Context,
	_ wallet.CreateHoldCommand,
) (domain.Hold, error) {
	if f.err != nil {
		return domain.Hold{}, f.err
	}
	return f.hold, nil
}

type fakeAntiFraudChecker struct {
	response AntiFraudResponse
	err      error
}

func (f *fakeAntiFraudChecker) Evaluate(
	_ context.Context,
	_ AntiFraudRequest,
) (AntiFraudResponse, error) {
	if f.err != nil {
		return AntiFraudResponse{}, f.err
	}
	return f.response, nil
}

type fakePaymentGateway struct {
	response ExternalPaymentResponse
	err      error
}

func (f *fakePaymentGateway) ProcessPayment(
	_ context.Context,
	_ ExternalPaymentRequest,
) (ExternalPaymentResponse, error) {
	if f.err != nil {
		return ExternalPaymentResponse{}, f.err
	}
	return f.response, nil
}

type fakeHoldCapturer struct {
	result wallet.CaptureHoldResult
	err    error
}

func (f *fakeHoldCapturer) Execute(
	_ context.Context,
	_ wallet.CaptureHoldCommand,
) (wallet.CaptureHoldResult, error) {
	if f.err != nil {
		return wallet.CaptureHoldResult{}, f.err
	}
	return f.result, nil
}

type fakeHoldReleaser struct {
	releasedHoldID domain.HoldID
	callsCount     int
	err            error
}

func (f *fakeHoldReleaser) Execute(
	_ context.Context,
	holdID domain.HoldID,
) (domain.Hold, error) {
	f.callsCount++
	f.releasedHoldID = holdID
	if f.err != nil {
		return domain.Hold{}, f.err
	}
	return domain.Hold{}, nil
}

func TestPaymentSagaOrchestrator_Execute(t *testing.T) {
	t.Parallel()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}
	money, err := domain.NewMoney(currency, 5000)
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}

	holdID := domain.HoldID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	walletID := domain.WalletID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")
	settlementAccountID := domain.AccountID("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33")
	transactionID := domain.TransactionID("d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a44")
	journalEntryID := domain.JournalEntryID("e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a55")
	walletPostingID := domain.PostingID("f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a66")
	settlementPostingID := domain.PostingID("10eebc99-9c0b-4ef8-bb6d-6bb9bd380a77")

	now := time.Now().UTC()
	authorizedHold, err := domain.ReconstituteHold(
		holdID,
		walletID,
		money,
		domain.HoldStatusAuthorized,
		now,
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create hold: %v", err)
	}

	capturedHold, err := domain.ReconstituteHold(
		holdID,
		walletID,
		money,
		domain.HoldStatusCaptured,
		now,
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create captured hold: %v", err)
	}

	postingWallet, err := domain.NewPosting(walletPostingID, domain.AccountID("acc-1"), domain.PostingDirectionDebit, money)
	if err != nil {
		t.Fatalf("failed to create posting: %v", err)
	}
	postingSettlement, err := domain.NewPosting(settlementPostingID, settlementAccountID, domain.PostingDirectionCredit, money)
	if err != nil {
		t.Fatalf("failed to create posting: %v", err)
	}
	journalEntry, err := domain.NewJournalEntry(journalEntryID, transactionID, currency, []domain.Posting{postingWallet, postingSettlement})
	if err != nil {
		t.Fatalf("failed to create journal entry: %v", err)
	}
	postedTx, err := domain.NewTransaction(transactionID, "checkout payment", journalEntry)
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}

	validCommand := ProcessPaymentCommand{
		HoldID:              holdID,
		WalletID:            walletID,
		SettlementAccountID: settlementAccountID,
		TransactionID:       transactionID,
		JournalEntryID:      journalEntryID,
		WalletPostingID:     walletPostingID,
		SettlementPostingID: settlementPostingID,
		AmountMinorUnits:    5000,
		Currency:            currency,
		Recipient:           "merchant_clean",
		Description:         "checkout payment",
		HoldExpiresAt:       now.Add(time.Hour),
		IdempotencyKey:      "pay_idem_001",
		RequestHash:         "pay_hash_001",
	}

	t.Run("successful end to end saga execution", func(t *testing.T) {
		t.Parallel()

		authorizer := &fakeHoldAuthorizer{hold: authorizedHold}
		antiFraud := &fakeAntiFraudChecker{response: AntiFraudResponse{Approved: true, RiskScore: 10}}
		gateway := &fakePaymentGateway{response: ExternalPaymentResponse{Success: true, TransactionID: "gw_tx_001"}}
		capturer := &fakeHoldCapturer{result: wallet.CaptureHoldResult{Hold: capturedHold, Transaction: postedTx}}
		releaser := &fakeHoldReleaser{}

		orchestrator := NewPaymentSagaOrchestrator(authorizer, antiFraud, capturer, releaser, gateway)

		result, err := orchestrator.Execute(context.Background(), validCommand)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "COMPLETED" {
			t.Fatalf("expected status COMPLETED, got %s", result.Status)
		}
		if result.HoldID != holdID {
			t.Fatalf("expected hold id %v, got %v", holdID, result.HoldID)
		}
		if result.TransactionID != transactionID {
			t.Fatalf("expected transaction id %v, got %v", transactionID, result.TransactionID)
		}
		if releaser.callsCount != 0 {
			t.Fatalf("expected 0 compensation calls, got %d", releaser.callsCount)
		}
	})

	t.Run("anti fraud rejection triggers compensation and releases hold", func(t *testing.T) {
		t.Parallel()

		authorizer := &fakeHoldAuthorizer{hold: authorizedHold}
		antiFraud := &fakeAntiFraudChecker{response: AntiFraudResponse{Approved: false, RiskScore: 90, RejectReason: "high risk score"}}
		gateway := &fakePaymentGateway{response: ExternalPaymentResponse{Success: true}}
		capturer := &fakeHoldCapturer{}
		releaser := &fakeHoldReleaser{}

		orchestrator := NewPaymentSagaOrchestrator(authorizer, antiFraud, capturer, releaser, gateway)

		_, err := orchestrator.Execute(context.Background(), validCommand)
		if !errors.Is(err, ErrAntiFraudRejected) {
			t.Fatalf("expected ErrAntiFraudRejected, got %v", err)
		}
		if releaser.callsCount != 1 {
			t.Fatalf("expected 1 compensation release call, got %d", releaser.callsCount)
		}
		if releaser.releasedHoldID != holdID {
			t.Fatalf("expected released hold id %v, got %v", holdID, releaser.releasedHoldID)
		}
	})

	t.Run("gateway failure triggers compensation and releases hold", func(t *testing.T) {
		t.Parallel()

		authorizer := &fakeHoldAuthorizer{hold: authorizedHold}
		antiFraud := &fakeAntiFraudChecker{response: AntiFraudResponse{Approved: true, RiskScore: 10}}
		gateway := &fakePaymentGateway{response: ExternalPaymentResponse{Success: false, ErrorMessage: "card declined"}}
		capturer := &fakeHoldCapturer{}
		releaser := &fakeHoldReleaser{}

		orchestrator := NewPaymentSagaOrchestrator(authorizer, antiFraud, capturer, releaser, gateway)

		_, err := orchestrator.Execute(context.Background(), validCommand)
		if !errors.Is(err, ErrExternalPaymentFailed) {
			t.Fatalf("expected ErrExternalPaymentFailed, got %v", err)
		}
		if releaser.callsCount != 1 {
			t.Fatalf("expected 1 compensation release call, got %d", releaser.callsCount)
		}
	})

	t.Run("hold authorization failure does not invoke anti-fraud or gateway", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("insufficient balance")
		authorizer := &fakeHoldAuthorizer{err: expectedErr}
		antiFraud := &fakeAntiFraudChecker{}
		gateway := &fakePaymentGateway{}
		capturer := &fakeHoldCapturer{}
		releaser := &fakeHoldReleaser{}

		orchestrator := NewPaymentSagaOrchestrator(authorizer, antiFraud, capturer, releaser, gateway)

		_, err := orchestrator.Execute(context.Background(), validCommand)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if releaser.callsCount != 0 {
			t.Fatalf("expected 0 compensation calls, got %d", releaser.callsCount)
		}
	})
}
