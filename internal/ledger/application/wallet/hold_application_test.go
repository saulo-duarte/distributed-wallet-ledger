package wallet

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/domain"
)

func TestCreateHoldUseCaseAuthorizesHold(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeApplicationHoldRepository{}
	useCase := NewCreateHoldUseCase(
		&fakeWalletReader{wallet: foundWallet},
		repository,
	)

	expiresAt := time.Now().UTC().Add(time.Hour)
	hold, err := useCase.Execute(
		context.Background(),
		CreateHoldCommand{
			ID:               domain.HoldID("hold-001"),
			WalletID:         foundWallet.ID(),
			AmountMinorUnits: 3000,
			ExpiresAt:        expiresAt,
			IdempotencyKey:   "hold-key-001",
			RequestHash:      "hold-hash-001",
		},
	)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if repository.authorizeCalls != 1 {
		t.Fatalf("Authorize() calls = %d, want 1", repository.authorizeCalls)
	}
	if hold.ID() != domain.HoldID("hold-001") {
		t.Fatalf("hold ID = %q, want hold-001", hold.ID())
	}
	if hold.Amount().AmountMinorUnits() != 3000 {
		t.Fatalf("hold amount = %d, want 3000", hold.Amount().AmountMinorUnits())
	}
	if !hold.IsAuthorized() {
		t.Fatal("new hold should be authorized")
	}
}

func TestCreateHoldUseCaseRejectsInvalidCommand(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeApplicationHoldRepository{}
	useCase := NewCreateHoldUseCase(
		&fakeWalletReader{wallet: foundWallet},
		repository,
	)

	baseCommand := CreateHoldCommand{
		ID:               domain.HoldID("hold-001"),
		WalletID:         foundWallet.ID(),
		AmountMinorUnits: 3000,
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
		IdempotencyKey:   "hold-key-001",
		RequestHash:      "hold-hash-001",
	}

	tests := []struct {
		name    string
		command CreateHoldCommand
		wantErr error
	}{
		{
			name:    "zero amount",
			command: commandWithAmount(baseCommand, 0),
			wantErr: domain.ErrAmountMustBePositive,
		},
		{
			name:    "negative amount",
			command: commandWithAmount(baseCommand, -1),
			wantErr: domain.ErrAmountMustNotBeNegative,
		},
		{
			name:    "empty idempotency key",
			command: commandWithIdempotencyKey(baseCommand, ""),
			wantErr: ErrEmptyIdempotencyKey,
		},
		{
			name:    "empty request hash",
			command: commandWithRequestHash(baseCommand, ""),
			wantErr: ErrEmptyRequestHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := useCase.Execute(context.Background(), tt.command)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	if repository.authorizeCalls != 0 {
		t.Fatal("invalid commands must not authorize holds")
	}
}

func TestCreateHoldUseCasePropagatesWalletError(t *testing.T) {
	expectedErr := ErrWalletNotFound
	repository := &fakeApplicationHoldRepository{}
	useCase := NewCreateHoldUseCase(
		&fakeWalletReader{err: expectedErr},
		repository,
	)

	_, err := useCase.Execute(
		context.Background(),
		CreateHoldCommand{
			ID:               domain.HoldID("hold-001"),
			WalletID:         domain.WalletID("wallet-001"),
			AmountMinorUnits: 3000,
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
			IdempotencyKey:   "hold-key-001",
			RequestHash:      "hold-hash-001",
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Execute() error = %v, want %v", err, expectedErr)
	}
	if repository.authorizeCalls != 0 {
		t.Fatal("hold must not be authorized when wallet lookup fails")
	}
}

func TestCreateHoldUseCasePropagatesInsufficientFunds(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeApplicationHoldRepository{authorizeErr: ErrInsufficientFunds}
	useCase := NewCreateHoldUseCase(
		&fakeWalletReader{wallet: foundWallet},
		repository,
	)

	_, err := useCase.Execute(
		context.Background(),
		CreateHoldCommand{
			ID:               domain.HoldID("hold-001"),
			WalletID:         foundWallet.ID(),
			AmountMinorUnits: 3000,
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
			IdempotencyKey:   "hold-key-001",
			RequestHash:      "hold-hash-001",
		},
	)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrInsufficientFunds)
	}
}

func TestReleaseHoldUseCaseUpdatesStatus(t *testing.T) {
	hold := newApplicationHold(t, domain.HoldStatusAuthorized, time.Now().UTC().Add(time.Hour))
	repository := &fakeApplicationHoldRepository{hold: hold}
	useCase := NewReleaseHoldUseCase(repository)

	released, err := useCase.Execute(context.Background(), hold.ID())
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}
	if !released.IsReleased() {
		t.Fatal("hold should be released")
	}
	if repository.updatedStatus != domain.HoldStatusReleased {
		t.Fatalf("updated status = %q, want released", repository.updatedStatus)
	}
}

func TestExpireHoldUseCaseUpdatesStatus(t *testing.T) {
	hold := newApplicationHold(t, domain.HoldStatusAuthorized, time.Now().UTC().Add(-time.Hour))
	repository := &fakeApplicationHoldRepository{hold: hold}
	useCase := NewExpireHoldUseCase(repository)

	expired, err := useCase.Execute(context.Background(), hold.ID())
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}
	if !expired.IsExpired() {
		t.Fatal("hold should be expired")
	}
	if repository.updatedStatus != domain.HoldStatusExpired {
		t.Fatalf("updated status = %q, want expired", repository.updatedStatus)
	}
}

func TestCaptureHoldUseCaseCreatesSettlementTransaction(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	hold := newApplicationHold(
		t,
		domain.HoldStatusAuthorized,
		time.Now().UTC().Add(time.Hour),
	)
	repository := &fakeApplicationHoldRepository{hold: hold}
	useCase := NewCaptureHoldUseCase(
		&fakeWalletReader{wallet: foundWallet},
		repository,
		repository,
	)

	result, err := useCase.Execute(
		context.Background(),
		CaptureHoldCommand{
			HoldID:              hold.ID(),
			SettlementAccountID: domain.AccountID("settlement-account-001"),
			TransactionID:       domain.TransactionID("transaction-capture-001"),
			JournalEntryID:      domain.JournalEntryID("journal-capture-001"),
			WalletPostingID:     domain.PostingID("posting-wallet-001"),
			SettlementPostingID: domain.PostingID("posting-settlement-001"),
			Description:         "Capture wallet hold",
			IdempotencyKey:      "capture-key-001",
			RequestHash:         "capture-hash-001",
		},
	)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if repository.captureCalls != 1 {
		t.Fatalf("Capture() calls = %d, want 1", repository.captureCalls)
	}
	if !result.Hold.IsCaptured() {
		t.Fatal("hold should be captured")
	}

	postings := result.Transaction.JournalEntry().Postings()
	if postings[0].AccountID() != foundWallet.LedgerAccountID() {
		t.Fatalf("wallet account = %q, want %q", postings[0].AccountID(), foundWallet.LedgerAccountID())
	}
	if postings[0].Direction() != domain.PostingDirectionDebit {
		t.Fatalf("wallet direction = %q, want debit", postings[0].Direction())
	}
	if postings[1].Direction() != domain.PostingDirectionCredit {
		t.Fatalf("settlement direction = %q, want credit", postings[1].Direction())
	}
}

func TestCaptureHoldUseCaseRejectsExpiredHold(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	hold := newApplicationHold(
		t,
		domain.HoldStatusAuthorized,
		time.Now().UTC().Add(-time.Hour),
	)
	repository := &fakeApplicationHoldRepository{hold: hold}
	useCase := NewCaptureHoldUseCase(
		&fakeWalletReader{wallet: foundWallet},
		repository,
		repository,
	)

	_, err := useCase.Execute(
		context.Background(),
		newCaptureHoldCommand(hold),
	)
	if !errors.Is(err, domain.ErrHoldExpired) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrHoldExpired)
	}
	if repository.captureCalls != 0 {
		t.Fatal("expired hold must not be captured")
	}
}

func TestCaptureHoldUseCaseRejectsSameSettlementAccount(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	hold := newApplicationHold(
		t,
		domain.HoldStatusAuthorized,
		time.Now().UTC().Add(time.Hour),
	)
	repository := &fakeApplicationHoldRepository{hold: hold}
	useCase := NewCaptureHoldUseCase(
		&fakeWalletReader{wallet: foundWallet},
		repository,
		repository,
	)
	command := newCaptureHoldCommand(hold)
	command.SettlementAccountID = foundWallet.LedgerAccountID()

	_, err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, ErrCaptureSameAccount) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrCaptureSameAccount)
	}
	if repository.captureCalls != 0 {
		t.Fatal("same-account capture must not be persisted")
	}
}

func newCaptureHoldCommand(hold domain.Hold) CaptureHoldCommand {
	return CaptureHoldCommand{
		HoldID:              hold.ID(),
		SettlementAccountID: domain.AccountID("settlement-account-001"),
		TransactionID:       domain.TransactionID("transaction-capture-001"),
		JournalEntryID:      domain.JournalEntryID("journal-capture-001"),
		WalletPostingID:     domain.PostingID("posting-wallet-001"),
		SettlementPostingID: domain.PostingID("posting-settlement-001"),
		Description:         "Capture wallet hold",
		IdempotencyKey:      "capture-key-001",
		RequestHash:         "capture-hash-001",
	}
}

func commandWithAmount(command CreateHoldCommand, amount int64) CreateHoldCommand {
	command.AmountMinorUnits = amount
	return command
}

func commandWithIdempotencyKey(command CreateHoldCommand, key string) CreateHoldCommand {
	command.IdempotencyKey = key
	return command
}

func commandWithRequestHash(command CreateHoldCommand, hash string) CreateHoldCommand {
	command.RequestHash = hash
	return command
}

func newApplicationHold(
	t *testing.T,
	status domain.HoldStatus,
	expiresAt time.Time,
) domain.Hold {
	t.Helper()

	createdAt := expiresAt.Add(-2 * time.Hour)
	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}
	amount, err := domain.NewMoney(currency, 3000)
	if err != nil {
		t.Fatal(err)
	}

	hold, err := domain.ReconstituteHold(
		domain.HoldID("hold-001"),
		domain.WalletID("wallet-001"),
		amount,
		status,
		createdAt,
		expiresAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	return hold
}

type fakeApplicationHoldRepository struct {
	hold           domain.Hold
	authorizeErr   error
	getErr         error
	updateErr      error
	captureErr     error
	authorizeCalls int
	captureCalls   int
	capturedTx     domain.Transaction
	updatedStatus  domain.HoldStatus
}

func (f *fakeApplicationHoldRepository) Authorize(
	_ context.Context,
	hold domain.Hold,
	_ string,
	_ string,
) (domain.Hold, error) {
	f.authorizeCalls++
	if f.authorizeErr != nil {
		return domain.Hold{}, f.authorizeErr
	}

	f.hold = hold
	return hold, nil
}

func (f *fakeApplicationHoldRepository) GetByID(
	_ context.Context,
	_ domain.HoldID,
) (domain.Hold, error) {
	if f.getErr != nil {
		return domain.Hold{}, f.getErr
	}
	return f.hold, nil
}

func (f *fakeApplicationHoldRepository) UpdateStatus(
	_ context.Context,
	_ domain.HoldID,
	status domain.HoldStatus,
) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updatedStatus = status
	return nil
}

func (f *fakeApplicationHoldRepository) ListExpiredHolds(
	_ context.Context,
	_ int32,
) ([]domain.HoldID, error) {
	return nil, nil
}

func (f *fakeApplicationHoldRepository) Capture(
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
	f.capturedTx = postedTransaction
	return nil
}
