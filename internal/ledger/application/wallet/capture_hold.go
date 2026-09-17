package wallet

import (
	"context"
	"strings"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type CaptureHoldCommand struct {
	HoldID domain.HoldID

	SettlementAccountID domain.AccountID

	TransactionID  domain.TransactionID
	JournalEntryID domain.JournalEntryID

	WalletPostingID     domain.PostingID
	SettlementPostingID domain.PostingID

	Description string

	IdempotencyKey string
	RequestHash    string
}

type CaptureHoldResult struct {
	Hold        domain.Hold
	Transaction domain.Transaction
}

type CaptureHoldUseCase struct {
	wallets           WalletReader
	holds             HoldRepository
	captureRepository HoldCaptureRepository
}

func NewCaptureHoldUseCase(
	wallets WalletReader,
	holds HoldRepository,
	captureRepository HoldCaptureRepository,
) CaptureHoldUseCase {
	return CaptureHoldUseCase{
		wallets:           wallets,
		holds:             holds,
		captureRepository: captureRepository,
	}
}

func (uc CaptureHoldUseCase) Execute(
	ctx context.Context,
	command CaptureHoldCommand,
) (CaptureHoldResult, error) {
	if command.HoldID.IsZero() ||
		command.SettlementAccountID.IsZero() {
		return CaptureHoldResult{}, domain.ErrInvalidID
	}

	if command.TransactionID.IsZero() ||
		command.JournalEntryID.IsZero() ||
		command.WalletPostingID.IsZero() ||
		command.SettlementPostingID.IsZero() {
		return CaptureHoldResult{}, domain.ErrInvalidID
	}

	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return CaptureHoldResult{}, ErrEmptyIdempotencyKey
	}

	if strings.TrimSpace(command.RequestHash) == "" {
		return CaptureHoldResult{}, ErrEmptyRequestHash
	}

	hold, err := uc.holds.GetByID(ctx, command.HoldID)
	if err != nil {
		return CaptureHoldResult{}, err
	}

	foundWallet, err := uc.wallets.GetByID(ctx, hold.WalletID())
	if err != nil {
		return CaptureHoldResult{}, err
	}

	if err := foundWallet.CanOperate(); err != nil {
		return CaptureHoldResult{}, err
	}

	if foundWallet.LedgerAccountID() == command.SettlementAccountID {
		return CaptureHoldResult{}, ErrCaptureSameAccount
	}

	if hold.IsAuthorized() {
		if err := hold.Capture(time.Now().UTC()); err != nil {
			return CaptureHoldResult{}, err
		}
	} else if !hold.IsCaptured() {
		return CaptureHoldResult{}, domain.ErrHoldNotAuthorized
	}

	walletPosting, err := domain.NewPosting(
		command.WalletPostingID,
		foundWallet.LedgerAccountID(),
		domain.PostingDirectionDebit,
		hold.Amount(),
	)
	if err != nil {
		return CaptureHoldResult{}, err
	}

	settlementPosting, err := domain.NewPosting(
		command.SettlementPostingID,
		command.SettlementAccountID,
		domain.PostingDirectionCredit,
		hold.Amount(),
	)
	if err != nil {
		return CaptureHoldResult{}, err
	}
	journalEntry, err := domain.NewJournalEntry(
		command.JournalEntryID,
		command.TransactionID,
		hold.Amount().Currency(),
		[]domain.Posting{
			walletPosting,
			settlementPosting,
		},
	)
	if err != nil {
		return CaptureHoldResult{}, err
	}

	postedTransaction, err := domain.NewTransaction(
		command.TransactionID,
		command.Description,
		journalEntry,
	)
	if err != nil {
		return CaptureHoldResult{}, err
	}

	if err := uc.captureRepository.Capture(
		ctx,
		hold.ID(),
		postedTransaction,
		strings.TrimSpace(command.IdempotencyKey),
		strings.TrimSpace(command.RequestHash),
	); err != nil {
		return CaptureHoldResult{}, err
	}

	return CaptureHoldResult{
		Hold:        hold,
		Transaction: postedTransaction,
	}, nil
}
