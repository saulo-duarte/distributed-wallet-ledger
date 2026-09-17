package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

func TestWalletHandlerCreateReturnsCreated(t *testing.T) {
	repository := &fakeWalletRepository{}
	useCase := wallet.NewCreateWalletUseCase(repository)
	handler := NewWalletHandler(useCase, nil)
	router := NewRouterWithWallet(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets",
		bytes.NewBufferString(`{
			"id": "wallet-001",
			"owner_id": "owner-001",
			"ledger_account_id": "account-001",
			"currency": "BRL"
		}`),
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response walletResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "wallet-001" {
		t.Fatalf("unexpected wallet ID: %q", response.ID)
	}
	if response.OwnerID != "owner-001" {
		t.Fatalf("unexpected owner ID: %q", response.OwnerID)
	}
	if response.LedgerAccountID != "account-001" {
		t.Fatalf("unexpected ledger account ID: %q", response.LedgerAccountID)
	}
	if response.Currency != "BRL" {
		t.Fatalf("unexpected currency: %q", response.Currency)
	}
	if response.Status != string(domain.WalletStatusOpen) {
		t.Fatalf("unexpected status: %q", response.Status)
	}

	if repository.createCalls != 1 {
		t.Fatalf("repository Create() calls = %d, want 1", repository.createCalls)
	}
}

func TestWalletHandlerCreateRejectsEmptyOwnerID(t *testing.T) {
	useCase := wallet.NewCreateWalletUseCase(&fakeWalletRepository{})
	handler := NewWalletHandler(useCase, nil)
	router := NewRouterWithWallet(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets",
		bytes.NewBufferString(`{
			"id": "wallet-001",
			"owner_id": "",
			"ledger_account_id": "account-001",
			"currency": "BRL"
		}`),
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestWalletHandlerCreateRejectsInvalidCurrency(t *testing.T) {
	repository := &fakeWalletRepository{}
	useCase := wallet.NewCreateWalletUseCase(repository)
	handler := NewWalletHandler(useCase, nil)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets",
		bytes.NewBufferString(`{
			"id": "wallet-001",
			"owner_id": "owner-001",
			"ledger_account_id": "account-001",
			"currency": "BR"
		}`),
	)
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
	if repository.createCalls != 0 {
		t.Fatal("repository should not be called for invalid currency")
	}
}

func TestWalletHandlerGetReturnsWallet(t *testing.T) {
	foundWallet := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeWalletRepository{wallet: foundWallet}
	getWallet := wallet.NewGetWalletUseCase(repository)
	listWallets := wallet.NewListWalletsByOwnerUseCase(repository)
	handler := NewWalletHandlerWithQueries(
		wallet.CreateWalletUseCase{},
		getWallet,
		listWallets,
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
		http.MethodGet,
		"/wallets/wallet-001",
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

	var response walletResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != foundWallet.ID().String() {
		t.Fatalf("unexpected wallet ID: %q", response.ID)
	}
}

func TestWalletHandlerListByOwnerReturnsWallets(t *testing.T) {
	first := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-001")
	second := newHTTPTestWallet(t, "wallet-002", "owner-001", "account-002")
	repository := &fakeWalletRepository{
		wallets: []domain.Wallet{first, second},
	}
	handler := NewWalletHandlerWithQueries(
		wallet.CreateWalletUseCase{},
		wallet.NewGetWalletUseCase(repository),
		wallet.NewListWalletsByOwnerUseCase(repository),
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
		http.MethodGet,
		"/owners/owner-001/wallets",
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

	var response []walletResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 2 {
		t.Fatalf("unexpected wallet count: got %d, want 2", len(response))
	}
}

func TestWalletHandlerGetBalanceReturnsBalance(t *testing.T) {
	foundWallet := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeWalletRepository{
		wallet: foundWallet,
		balance: wallet.LedgerBalanceSnapshot{
			TotalDebits:  2500,
			TotalCredits: 10000,
		},
	}
	handler := NewWalletHandlerWithBalance(
		wallet.CreateWalletUseCase{},
		wallet.NewGetWalletUseCase(repository),
		wallet.NewListWalletsByOwnerUseCase(repository),
		wallet.NewGetWalletBalanceUseCase(repository, repository),
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
		http.MethodGet,
		"/wallets/wallet-001/balance",
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

	var response walletBalanceResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.WalletID != foundWallet.ID().String() {
		t.Fatalf("unexpected wallet ID: %q", response.WalletID)
	}
	if response.LedgerBalanceMinorUnits != 7500 {
		t.Fatalf("unexpected ledger balance: %d", response.LedgerBalanceMinorUnits)
	}
	if response.AvailableBalanceMinorUnits != 7500 {
		t.Fatalf("unexpected available balance: %d", response.AvailableBalanceMinorUnits)
	}
}

func TestWalletHandlerWithdrawPostsWithdrawal(t *testing.T) {
	foundWallet := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeWalletRepository{
		wallet: foundWallet,
		balance: wallet.LedgerBalanceSnapshot{
			TotalCredits: 10000,
		},
	}
	ledger := &fakeHTTPWithdrawalLedgerRepository{}
	postTransaction := transaction.NewPostTransactionUseCase(ledger)
	withdraw := wallet.NewWithdrawWalletUseCase(
		repository,
		repository,
		postTransaction,
	)
	handler := NewWalletHandlerWithWithdrawal(
		wallet.CreateWalletUseCase{},
		wallet.NewGetWalletUseCase(repository),
		wallet.NewListWalletsByOwnerUseCase(repository),
		wallet.NewGetWalletBalanceUseCase(repository, repository),
		withdraw,
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
		"/wallets/wallet-001/withdrawals",
		bytes.NewBufferString(`{
			"clearing_account_id": "clearing-account-001",
			"transaction_id": "transaction-001",
			"journal_entry_id": "journal-entry-001",
			"wallet_posting_id": "posting-wallet-001",
			"clearing_posting_id": "posting-clearing-001",
			"amount_minor_units": 3000,
			"description": "Wallet withdrawal"
		}`),
	)
	request.Header.Set(idempotencyKeyHeader, "withdrawal-key-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response transactionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != "transaction-001" {
		t.Fatalf("unexpected transaction ID: %q", response.ID)
	}
	if response.Currency != "BRL" {
		t.Fatalf("unexpected currency: %q", response.Currency)
	}
	if response.Status != "posted" {
		t.Fatalf("unexpected status: %q", response.Status)
	}

	if ledger.calls != 1 {
		t.Fatalf("ledger Post() calls = %d, want 1", ledger.calls)
	}
	postings := ledger.postedTransaction.JournalEntry().Postings()
	if postings[0].AccountID() != foundWallet.LedgerAccountID() {
		t.Fatalf("unexpected wallet account: %q", postings[0].AccountID())
	}
	if postings[0].Direction() != domain.PostingDirectionDebit {
		t.Fatalf("wallet posting direction = %q, want debit", postings[0].Direction())
	}
	if postings[1].AccountID() != domain.AccountID("clearing-account-001") {
		t.Fatalf("unexpected clearing account: %q", postings[1].AccountID())
	}
	if postings[1].Direction() != domain.PostingDirectionCredit {
		t.Fatalf("clearing posting direction = %q, want credit", postings[1].Direction())
	}
}

func TestWalletHandlerWithdrawRequiresIdempotencyKey(t *testing.T) {
	handler := NewWalletHandlerWithWithdrawal(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.WithdrawWalletUseCase{},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets/wallet-001/withdrawals",
		bytes.NewBufferString(`{}`),
	)
	recorder := httptest.NewRecorder()

	handler.Withdraw(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func TestWalletHandlerDepositPostsDeposit(t *testing.T) {
	foundWallet := newHTTPTestWallet(t, "wallet-001", "owner-001", "account-wallet")
	repository := &fakeWalletRepository{wallet: foundWallet}
	ledger := &fakeHTTPWithdrawalLedgerRepository{}
	postTransaction := transaction.NewPostTransactionUseCase(ledger)
	deposit := wallet.NewDepositWalletUseCase(repository, postTransaction)
	handler := NewWalletHandlerWithOperations(
		wallet.CreateWalletUseCase{},
		wallet.NewGetWalletUseCase(repository),
		wallet.NewListWalletsByOwnerUseCase(repository),
		wallet.NewGetWalletBalanceUseCase(repository, repository),
		deposit,
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
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
		"/wallets/wallet-001/deposits",
		bytes.NewBufferString(`{
			"clearing_account_id": "clearing-account-001",
			"transaction_id": "transaction-deposit-001",
			"journal_entry_id": "journal-deposit-001",
			"clearing_posting_id": "posting-clearing-001",
			"wallet_posting_id": "posting-wallet-001",
			"amount_minor_units": 3000,
			"description": "Wallet deposit"
		}`),
	)
	request.Header.Set(idempotencyKeyHeader, "deposit-key-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response transactionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != "transaction-deposit-001" {
		t.Fatalf("unexpected transaction ID: %q", response.ID)
	}
	if response.Currency != "BRL" {
		t.Fatalf("unexpected currency: %q", response.Currency)
	}
	if response.Status != "posted" {
		t.Fatalf("unexpected status: %q", response.Status)
	}

	if ledger.calls != 1 {
		t.Fatalf("ledger Post() calls = %d, want 1", ledger.calls)
	}
	postings := ledger.postedTransaction.JournalEntry().Postings()
	if postings[0].AccountID() != domain.AccountID("clearing-account-001") {
		t.Fatalf("unexpected clearing account: %q", postings[0].AccountID())
	}
	if postings[0].Direction() != domain.PostingDirectionDebit {
		t.Fatalf("clearing direction = %q, want debit", postings[0].Direction())
	}
	if postings[1].AccountID() != foundWallet.LedgerAccountID() {
		t.Fatalf("unexpected wallet account: %q", postings[1].AccountID())
	}
	if postings[1].Direction() != domain.PostingDirectionCredit {
		t.Fatalf("wallet direction = %q, want credit", postings[1].Direction())
	}
}

func TestWalletHandlerDepositRequiresIdempotencyKey(t *testing.T) {
	handler := NewWalletHandlerWithOperations(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.DepositWalletUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets/wallet-001/deposits",
		bytes.NewBufferString(`{}`),
	)
	recorder := httptest.NewRecorder()

	handler.Deposit(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func TestWalletHandlerTransferPostsTransfer(t *testing.T) {
	sourceWallet := newHTTPTestWallet(t, "wallet-source", "owner-001", "account-source")
	destinationWallet := newHTTPTestWallet(t, "wallet-destination", "owner-002", "account-destination")
	repository := &fakeWalletRepository{
		walletsByID: map[domain.WalletID]domain.Wallet{
			sourceWallet.ID():      sourceWallet,
			destinationWallet.ID(): destinationWallet,
		},
		balancesByAccount: map[domain.AccountID]wallet.LedgerBalanceSnapshot{
			sourceWallet.LedgerAccountID(): {
				TotalCredits: 10000,
			},
		},
	}
	ledger := &fakeHTTPWithdrawalLedgerRepository{}
	postTransaction := transaction.NewPostTransactionUseCase(ledger)
	transfer := wallet.NewTransferWalletUseCase(
		repository,
		repository,
		postTransaction,
	)
	handler := NewWalletHandlerWithTransfer(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.WithdrawWalletUseCase{},
		transfer,
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
		"/wallets/wallet-source/transfers",
		bytes.NewBufferString(`{
			"destination_wallet_id": "wallet-destination",
			"transaction_id": "transaction-transfer-001",
			"journal_entry_id": "journal-transfer-001",
			"source_posting_id": "posting-source-001",
			"destination_posting_id": "posting-destination-001",
			"amount_minor_units": 3000,
			"description": "Wallet transfer"
		}`),
	)
	request.Header.Set(idempotencyKeyHeader, "transfer-key-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response transactionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != "transaction-transfer-001" {
		t.Fatalf("unexpected transaction ID: %q", response.ID)
	}
	if response.Currency != "BRL" {
		t.Fatalf("unexpected currency: %q", response.Currency)
	}
	if response.Status != "posted" {
		t.Fatalf("unexpected status: %q", response.Status)
	}

	postings := ledger.postedTransaction.JournalEntry().Postings()
	if postings[0].AccountID() != sourceWallet.LedgerAccountID() {
		t.Fatalf("unexpected source account: %q", postings[0].AccountID())
	}
	if postings[0].Direction() != domain.PostingDirectionDebit {
		t.Fatalf("source direction = %q, want debit", postings[0].Direction())
	}
	if postings[1].AccountID() != destinationWallet.LedgerAccountID() {
		t.Fatalf("unexpected destination account: %q", postings[1].AccountID())
	}
	if postings[1].Direction() != domain.PostingDirectionCredit {
		t.Fatalf("destination direction = %q, want credit", postings[1].Direction())
	}
}

func TestWalletHandlerTransferRequiresIdempotencyKey(t *testing.T) {
	handler := NewWalletHandlerWithTransfer(
		wallet.CreateWalletUseCase{},
		wallet.GetWalletUseCase{},
		wallet.ListWalletsByOwnerUseCase{},
		wallet.GetWalletBalanceUseCase{},
		wallet.WithdrawWalletUseCase{},
		wallet.TransferWalletUseCase{},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/wallets/wallet-source/transfers",
		bytes.NewBufferString(`{}`),
	)
	recorder := httptest.NewRecorder()

	handler.Transfer(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func newHTTPTestWallet(
	t *testing.T,
	id string,
	ownerID string,
	ledgerAccountID string,
) domain.Wallet {
	t.Helper()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	foundWallet, err := domain.NewWallet(
		domain.WalletID(id),
		domain.OwnerID(ownerID),
		domain.AccountID(ledgerAccountID),
		currency,
	)
	if err != nil {
		t.Fatal(err)
	}

	return foundWallet
}

type fakeWalletRepository struct {
	wallet            domain.Wallet
	wallets           []domain.Wallet
	walletsByID       map[domain.WalletID]domain.Wallet
	balancesByAccount map[domain.AccountID]wallet.LedgerBalanceSnapshot
	balance           wallet.LedgerBalanceSnapshot
	createCalls       int
	err               error
}

func (f *fakeWalletRepository) Create(
	_ context.Context,
	wallet domain.Wallet,
) error {
	f.createCalls++
	if f.err != nil {
		return f.err
	}

	f.wallet = wallet
	return nil
}

func (f *fakeWalletRepository) GetByID(
	_ context.Context,
	walletID domain.WalletID,
) (domain.Wallet, error) {
	if f.err != nil {
		return domain.Wallet{}, f.err
	}
	if f.walletsByID != nil {
		foundWallet, ok := f.walletsByID[walletID]
		if !ok {
			return domain.Wallet{}, wallet.ErrWalletNotFound
		}
		return foundWallet, nil
	}

	return f.wallet, nil
}

func (f *fakeWalletRepository) ListByOwnerID(
	_ context.Context,
	_ domain.OwnerID,
) ([]domain.Wallet, error) {
	if f.err != nil {
		return nil, f.err
	}

	if f.wallets != nil {
		return f.wallets, nil
	}

	return []domain.Wallet{f.wallet}, nil
}

func (f *fakeWalletRepository) GetLedgerBalance(
	_ context.Context,
	accountID domain.AccountID,
) (wallet.LedgerBalanceSnapshot, error) {
	if f.err != nil {
		return wallet.LedgerBalanceSnapshot{}, f.err
	}
	if f.balancesByAccount != nil {
		return f.balancesByAccount[accountID], nil
	}

	return f.balance, nil
}

type fakeHTTPWithdrawalLedgerRepository struct {
	postedTransaction domain.Transaction
	calls             int
}

func (f *fakeHTTPWithdrawalLedgerRepository) Post(
	_ context.Context,
	postedTransaction domain.Transaction,
	_ string,
	_ string,
) error {
	f.calls++
	f.postedTransaction = postedTransaction
	return nil
}
