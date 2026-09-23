package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

func TestWalletHandlerCreateHoldReturnsCreated(t *testing.T) {
	foundWallet := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-001")
	walletRepository := &fakeWalletRepository{wallet: foundWallet}
	holdRepository := &fakeHTTPHoldRepository{}
	handler := NewWalletHandlerWithHolds(
		wallet.CreateWalletUseCase{},
		wallet.NewGetWalletUseCase(walletRepository),
		wallet.NewListWalletsByOwnerUseCase(walletRepository),
		wallet.NewGetWalletBalanceUseCase(walletRepository, walletRepository),
		wallet.DepositWalletUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		wallet.NewCreateHoldUseCase(walletRepository, holdRepository),
		wallet.NewReleaseHoldUseCase(holdRepository),
		wallet.NewExpireHoldUseCase(holdRepository),
		wallet.CaptureHoldUseCase{},
		nil,
	)
	router := NewRouterWithWallet(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	body, err := json.Marshal(createHoldRequest{
		ID:               "hold-001",
		AmountMinorUnits: 3000,
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets/wallet-001/holds",
		bytes.NewReader(body),
	)
	request.Header.Set(idempotencyKeyHeader, "hold-key-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response holdResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != "hold-001" {
		t.Fatalf("hold ID = %q, want hold-001", response.ID)
	}
	if response.WalletID != "wallet-001" {
		t.Fatalf("wallet ID = %q, want wallet-001", response.WalletID)
	}
	if response.AmountMinorUnits != 3000 {
		t.Fatalf("hold amount = %d, want 3000", response.AmountMinorUnits)
	}
	if response.Status != string(domain.HoldStatusAuthorized) {
		t.Fatalf("hold status = %q, want authorized", response.Status)
	}
	if holdRepository.authorizeCalls != 1 {
		t.Fatalf("Authorize() calls = %d, want 1", holdRepository.authorizeCalls)
	}
}

func TestWalletHandlerCreateHoldRequiresIdempotencyKey(t *testing.T) {
	handler := NewWalletHandlerWithHolds(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.DepositWalletUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		wallet.CreateHoldUseCase{},
		wallet.ReleaseHoldUseCase{},
		wallet.ExpireHoldUseCase{},
		wallet.CaptureHoldUseCase{},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets/wallet-001/holds",
		bytes.NewBufferString(`{"id":"hold-001"}`),
	)
	recorder := httptest.NewRecorder()

	handler.CreateHold(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func TestWalletHandlerReleaseHoldReturnsUpdatedHold(t *testing.T) {
	hold := newHTTPHold(t, domain.HoldStatusAuthorized, time.Now().UTC().Add(time.Hour))
	repository := &fakeHTTPHoldRepository{hold: hold}
	handler := NewWalletHandlerWithHolds(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.DepositWalletUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		wallet.CreateHoldUseCase{},
		wallet.NewReleaseHoldUseCase(repository),
		wallet.NewExpireHoldUseCase(repository),
		wallet.CaptureHoldUseCase{},
		nil,
	)
	router := NewRouterWithWallet(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/holds/hold-001/release",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response holdResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != string(domain.HoldStatusReleased) {
		t.Fatalf("hold status = %q, want released", response.Status)
	}
}

func TestWalletHandlerExpireHoldReturnsUpdatedHold(t *testing.T) {
	hold := newHTTPHold(t, domain.HoldStatusAuthorized, time.Now().UTC().Add(-time.Hour))
	repository := &fakeHTTPHoldRepository{hold: hold}
	handler := NewWalletHandlerWithHolds(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.DepositWalletUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		wallet.CreateHoldUseCase{},
		wallet.NewReleaseHoldUseCase(repository),
		wallet.NewExpireHoldUseCase(repository),
		wallet.CaptureHoldUseCase{},
		nil,
	)
	router := NewRouterWithWallet(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/holds/hold-001/expire",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response holdResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != string(domain.HoldStatusExpired) {
		t.Fatalf("hold status = %q, want expired", response.Status)
	}
}

func TestWalletHandlerCaptureHoldReturnsCapturedTransaction(t *testing.T) {
	foundWallet := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-wallet")
	hold := newHTTPHold(t, domain.HoldStatusAuthorized, time.Now().UTC().Add(time.Hour))
	walletRepository := &fakeWalletRepository{wallet: foundWallet}
	holdRepository := &fakeHTTPHoldRepository{hold: hold}
	capture := wallet.NewCaptureHoldUseCase(
		walletRepository,
		holdRepository,
		holdRepository,
	)
	handler := NewWalletHandlerWithHolds(
		wallet.CreateWalletUseCase{},
		wallet.NewGetWalletUseCase(walletRepository),
		wallet.NewListWalletsByOwnerUseCase(walletRepository),
		wallet.NewGetWalletBalanceUseCase(walletRepository, walletRepository),
		wallet.DepositWalletUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		wallet.CreateHoldUseCase{},
		wallet.ReleaseHoldUseCase{},
		wallet.ExpireHoldUseCase{},
		capture,
		nil,
	)
	router := NewRouterWithWallet(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/holds/hold-001/capture",
		bytes.NewBufferString(`{
			"settlement_account_id": "settlement-account-001",
			"transaction_id": "transaction-capture-001",
			"journal_entry_id": "journal-capture-001",
			"wallet_posting_id": "posting-wallet-001",
			"settlement_posting_id": "posting-settlement-001",
			"description": "Capture wallet hold"
		}`),
	)
	request.Header.Set(idempotencyKeyHeader, "capture-key-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response captureHoldResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Hold.Status != string(domain.HoldStatusCaptured) {
		t.Fatalf("hold status = %q, want captured", response.Hold.Status)
	}
	if response.Transaction.ID != "transaction-capture-001" {
		t.Fatalf(
			"transaction ID = %q, want transaction-capture-001",
			response.Transaction.ID,
		)
	}
	if holdRepository.captureCalls != 1 {
		t.Fatalf("Capture() calls = %d, want 1", holdRepository.captureCalls)
	}
}

func newHTTPHold(
	t *testing.T,
	status domain.HoldStatus,
	expiresAt time.Time,
) domain.Hold {
	t.Helper()

	amount, err := domain.NewMoney(newHTTPTestCurrency(t), 3000)
	if err != nil {
		t.Fatal(err)
	}

	hold, err := domain.ReconstituteHold(
		domain.HoldID("hold-001"),
		domain.WalletID("wallet-001"),
		amount,
		status,
		expiresAt.Add(-2*time.Hour),
		expiresAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	return hold
}

func newHTTPTestCurrency(t *testing.T) domain.Currency {
	t.Helper()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	return currency
}

type fakeHTTPHoldRepository struct {
	hold           domain.Hold
	authorizeCalls int
	captureCalls   int
	captureErr     error
	capturedTx     domain.Transaction
	getErr         error
	updateErr      error
}

func (f *fakeHTTPHoldRepository) Authorize(
	_ context.Context,
	hold domain.Hold,
	_ string,
	_ string,
) (domain.Hold, error) {
	f.authorizeCalls++
	f.hold = hold
	return hold, nil
}

func (f *fakeHTTPHoldRepository) GetByID(
	_ context.Context,
	_ domain.HoldID,
) (domain.Hold, error) {
	if f.getErr != nil {
		return domain.Hold{}, f.getErr
	}
	return f.hold, nil
}

func (f *fakeHTTPHoldRepository) UpdateStatus(
	_ context.Context,
	_ domain.HoldID,
	status domain.HoldStatus,
) error {
	if f.updateErr != nil {
		return f.updateErr
	}

	updated, err := domain.ReconstituteHold(
		f.hold.ID(),
		f.hold.WalletID(),
		f.hold.Amount(),
		status,
		f.hold.CreatedAt(),
		f.hold.ExpiresAt(),
	)
	if err != nil {
		return err
	}
	f.hold = updated
	return nil
}

func (f *fakeHTTPHoldRepository) ListExpiredHolds(
	_ context.Context,
	_ int32,
) ([]domain.HoldID, error) {
	return nil, nil
}

func (f *fakeHTTPHoldRepository) Capture(
	_ context.Context,
	_ domain.HoldID,
	postedTransaction domain.Transaction,
	_ string,
	_ string,
) error {
	f.captureCalls++
	if f.captureErr != nil {
		return f.captureErr
	}

	updated, err := domain.ReconstituteHold(
		f.hold.ID(),
		f.hold.WalletID(),
		f.hold.Amount(),
		domain.HoldStatusCaptured,
		f.hold.CreatedAt(),
		f.hold.ExpiresAt(),
	)
	if err != nil {
		return err
	}
	f.hold = updated
	f.capturedTx = postedTransaction
	return nil
}
