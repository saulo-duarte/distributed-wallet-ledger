package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

type fakeSagaHoldAuthorizer struct {
	hold domain.Hold
	err  error
}

func (f *fakeSagaHoldAuthorizer) Execute(
	_ context.Context,
	_ wallet.CreateHoldCommand,
) (domain.Hold, error) {
	if f.err != nil {
		return domain.Hold{}, f.err
	}
	return f.hold, nil
}

type fakeSagaAntiFraudChecker struct {
	response saga.AntiFraudResponse
	err      error
}

func (f *fakeSagaAntiFraudChecker) Evaluate(
	_ context.Context,
	_ saga.AntiFraudRequest,
) (saga.AntiFraudResponse, error) {
	if f.err != nil {
		return saga.AntiFraudResponse{}, f.err
	}
	return f.response, nil
}

type fakeSagaPaymentGateway struct {
	response saga.ExternalPaymentResponse
	err      error
}

func (f *fakeSagaPaymentGateway) ProcessPayment(
	_ context.Context,
	_ saga.ExternalPaymentRequest,
) (saga.ExternalPaymentResponse, error) {
	if f.err != nil {
		return saga.ExternalPaymentResponse{}, f.err
	}
	return f.response, nil
}

type fakeSagaHoldCapturer struct {
	result wallet.CaptureHoldResult
	err    error
}

func (f *fakeSagaHoldCapturer) Execute(
	_ context.Context,
	_ wallet.CaptureHoldCommand,
) (wallet.CaptureHoldResult, error) {
	if f.err != nil {
		return wallet.CaptureHoldResult{}, f.err
	}
	return f.result, nil
}

type fakeSagaHoldReleaser struct {
	callsCount int
}

func (f *fakeSagaHoldReleaser) Execute(
	_ context.Context,
	_ domain.HoldID,
) (domain.Hold, error) {
	f.callsCount++
	return domain.Hold{}, nil
}

func TestPaymentHandler_Checkout(t *testing.T) {
	t.Parallel()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}
	money, err := domain.NewMoney(currency, 2500)
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
	authHold, _ := domain.ReconstituteHold(holdID, walletID, money, domain.HoldStatusAuthorized, now, now.Add(time.Hour))
	captHold, _ := domain.ReconstituteHold(holdID, walletID, money, domain.HoldStatusCaptured, now, now.Add(time.Hour))

	pWallet, _ := domain.NewPosting(walletPostingID, domain.AccountID("acc-1"), domain.PostingDirectionDebit, money)
	pSettlement, _ := domain.NewPosting(settlementPostingID, settlementAccountID, domain.PostingDirectionCredit, money)
	jEntry, _ := domain.NewJournalEntry(journalEntryID, transactionID, currency, []domain.Posting{pWallet, pSettlement})
	tx, _ := domain.NewTransaction(transactionID, "Checkout payment", jEntry)

	baseReqBody := map[string]any{
		"hold_id":               holdID.String(),
		"wallet_id":             walletID.String(),
		"settlement_account_id": settlementAccountID.String(),
		"transaction_id":        transactionID.String(),
		"journal_entry_id":      journalEntryID.String(),
		"wallet_posting_id":     walletPostingID.String(),
		"settlement_posting_id": settlementPostingID.String(),
		"amount_minor_units":    2500,
		"currency":              "BRL",
		"recipient":             "super_market",
		"description":           "Checkout payment",
		"hold_expires_in_sec":   3600,
	}

	t.Run("successful checkout returns 201 Created", func(t *testing.T) {
		t.Parallel()

		authorizer := &fakeSagaHoldAuthorizer{hold: authHold}
		antiFraud := &fakeSagaAntiFraudChecker{response: saga.AntiFraudResponse{Approved: true, RiskScore: 10}}
		gateway := &fakeSagaPaymentGateway{response: saga.ExternalPaymentResponse{Success: true, TransactionID: "gw-999"}}
		capturer := &fakeSagaHoldCapturer{result: wallet.CaptureHoldResult{Hold: captHold, Transaction: tx}}
		releaser := &fakeSagaHoldReleaser{}

		orchestrator := saga.NewPaymentSagaOrchestrator(authorizer, antiFraud, capturer, releaser, gateway)
		handler := NewPaymentHandler(orchestrator, nil)
		router := NewRouterWithSaga(nil, func(context.Context) error { return nil }, nil, nil, nil, handler)

		bodyBytes, _ := json.Marshal(baseReqBody)
		req := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(bodyBytes))
		req.Header.Set("Idempotency-Key", "idem-checkout-001")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d. body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var resp processPaymentResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "COMPLETED" {
			t.Fatalf("expected status COMPLETED, got %s", resp.Status)
		}
		if resp.GatewayTransactionID != "gw-999" {
			t.Fatalf("expected gateway tx id gw-999, got %s", resp.GatewayTransactionID)
		}
	})

	t.Run("rejected by anti-fraud returns 422 Unprocessable Entity", func(t *testing.T) {
		t.Parallel()

		authorizer := &fakeSagaHoldAuthorizer{hold: authHold}
		antiFraud := &fakeSagaAntiFraudChecker{response: saga.AntiFraudResponse{Approved: false, RiskScore: 99, RejectReason: "blacklist"}}
		gateway := &fakeSagaPaymentGateway{}
		capturer := &fakeSagaHoldCapturer{}
		releaser := &fakeSagaHoldReleaser{}

		orchestrator := saga.NewPaymentSagaOrchestrator(authorizer, antiFraud, capturer, releaser, gateway)
		handler := NewPaymentHandler(orchestrator, nil)
		router := NewRouterWithSaga(nil, func(context.Context) error { return nil }, nil, nil, nil, handler)

		bodyBytes, _ := json.Marshal(baseReqBody)
		req := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(bodyBytes))
		req.Header.Set("Idempotency-Key", "idem-checkout-002")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d, got %d. body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		}
		if releaser.callsCount != 1 {
			t.Fatalf("expected 1 compensation release call, got %d", releaser.callsCount)
		}
	})

	t.Run("rejected when missing Idempotency-Key header", func(t *testing.T) {
		t.Parallel()

		orchestrator := saga.NewPaymentSagaOrchestrator(nil, nil, nil, nil, nil)
		handler := NewPaymentHandler(orchestrator, nil)
		router := NewRouterWithSaga(nil, func(context.Context) error { return nil }, nil, nil, nil, handler)

		bodyBytes, _ := json.Marshal(baseReqBody)
		req := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}
