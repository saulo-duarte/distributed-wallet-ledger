package postgres

import (
	"context"
	"fmt"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const transactionPostingScope = "transaction.post"

type TransactionRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewTransactionRepository(
	pool *pgxpool.Pool,
	queries *db.Queries,
) *TransactionRepository {
	return &TransactionRepository{
		pool:    pool,
		queries: queries,
	}
}

var _ transaction.LedgerRepository = (*TransactionRepository)(nil)

func (r *TransactionRepository) Post(
	ctx context.Context,
	postedTransaction domain.Transaction,
	idempotencyKey string,
	requestHash string,
) error {
	if r == nil || r.pool == nil || r.queries == nil {
		return fmt.Errorf("transaction repository is not configured")
	}

	databaseTransaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction posting: %w", err)
	}
	defer func() {
		_ = databaseTransaction.Rollback(ctx)
	}()

	queries := r.queries.WithTx(databaseTransaction)

	transactionID, err := transactionIDToUUID(postedTransaction.ID())
	if err != nil {
		return err
	}

	if err := queries.CreateTransaction(
		ctx,
		db.CreateTransactionParams{
			ID:          transactionID,
			Description: postedTransaction.Description(),
		},
	); err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	journalEntry := postedTransaction.JournalEntry()
	journalEntryID, err := journalEntryIDToUUID(journalEntry.ID())
	if err != nil {
		return err
	}

	if err := queries.CreateJournalEntry(
		ctx,
		db.CreateJournalEntryParams{
			ID:            journalEntryID,
			TransactionID: transactionID,
			Currency:      journalEntry.Currency().String(),
		},
	); err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}

	for _, posting := range journalEntry.Postings() {
		postingID, err := postingIDToUUID(posting.ID())
		if err != nil {
			return err
		}

		accountID, err := accountIDToUUID(posting.AccountID())
		if err != nil {
			return err
		}

		if err := queries.CreatePosting(
			ctx,
			db.CreatePostingParams{
				ID:               postingID,
				JournalEntryID:   journalEntryID,
				AccountID:        accountID,
				Direction:        string(posting.Direction()),
				AmountMinorUnits: posting.Amount().AmountMinorUnits(),
			},
		); err != nil {
			return fmt.Errorf("create posting: %w", err)
		}
	}

	if err := queries.CreateIdempotencyKey(
		ctx,
		db.CreateIdempotencyKeyParams{
			Scope:          transactionPostingScope,
			IdempotencyKey: idempotencyKey,
			RequestHash:    requestHash,
			TransactionID:  transactionID,
		},
	); err != nil {
		return fmt.Errorf("create idempotency key: %w", err)
	}

	if err := databaseTransaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction posting: %w", err)
	}

	return nil
}

func transactionIDToUUID(id domain.TransactionID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf(
			"convert transaction ID %q to PostgreSQL UUID: %w",
			id.String(),
			err,
		)
	}

	return uuid, nil
}

func journalEntryIDToUUID(id domain.JournalEntryID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf(
			"convert journal entry ID %q to PostgreSQL UUID: %w",
			id.String(),
			err,
		)
	}

	return uuid, nil
}

func postingIDToUUID(id domain.PostingID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf(
			"convert posting ID %q to PostgreSQL UUID: %w",
			id.String(),
			err,
		)
	}

	return uuid, nil
}
