package postgres

import (
	"context"
	"errors"
	"fmt"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5"
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
var _ transaction.TransactionReader = (*TransactionRepository)(nil)
var _ transaction.TransactionAggregateReader = (*TransactionRepository)(nil)

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

	var reversesTransactionID pgtype.UUID
	if originalTransactionID := postedTransaction.ReversesTransactionID(); originalTransactionID != nil {
		reversesTransactionID, err = transactionIDToUUID(*originalTransactionID)
		if err != nil {
			return err
		}
	}

	if err := queries.CreateTransaction(
		ctx,
		db.CreateTransactionParams{
			ID:                    transactionID,
			Description:           postedTransaction.Description(),
			ReversesTransactionID: reversesTransactionID,
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

func (r *TransactionRepository) Get(
	ctx context.Context,
	id domain.TransactionID,
) (transaction.TransactionDetails, error) {
	if r == nil || r.queries == nil {
		return transaction.TransactionDetails{}, fmt.Errorf(
			"transaction repository is not configured",
		)
	}

	databaseID, err := transactionIDToUUID(id)
	if err != nil {
		return transaction.TransactionDetails{}, err
	}

	rows, err := r.queries.GetTransactionDetails(ctx, databaseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return transaction.TransactionDetails{}, transaction.ErrTransactionNotFound
		}
		return transaction.TransactionDetails{}, fmt.Errorf(
			"get transaction details: %w",
			err,
		)
	}
	if len(rows) == 0 {
		return transaction.TransactionDetails{}, transaction.ErrTransactionNotFound
	}

	first := rows[0]
	if !first.TransactionCreatedAt.Valid || !first.PostedAt.Valid {
		return transaction.TransactionDetails{}, fmt.Errorf(
			"transaction timestamps are invalid",
		)
	}

	details := transaction.TransactionDetails{
		ID:          domain.TransactionID(first.TransactionID.String()),
		Description: first.Description,
		CreatedAt:   first.TransactionCreatedAt.Time.UTC(),
		JournalEntry: transaction.JournalEntryDetails{
			ID:       domain.JournalEntryID(first.JournalEntryID.String()),
			Currency: first.Currency,
			PostedAt: first.PostedAt.Time.UTC(),
			Postings: make([]transaction.PostingDetails, 0, len(rows)),
		},
	}

	for _, row := range rows {
		details.JournalEntry.Postings = append(
			details.JournalEntry.Postings,
			transaction.PostingDetails{
				ID:               domain.PostingID(row.PostingID.String()),
				AccountID:        domain.AccountID(row.AccountID.String()),
				Direction:        domain.PostingDirection(row.Direction),
				AmountMinorUnits: row.AmountMinorUnits,
			},
		)
	}

	return details, nil
}

func (r *TransactionRepository) GetAggregate(
	ctx context.Context,
	id domain.TransactionID,
) (domain.Transaction, error) {
	if r == nil || r.queries == nil {
		return domain.Transaction{}, fmt.Errorf(
			"transaction repository is not configured",
		)
	}

	databaseID, err := transactionIDToUUID(id)
	if err != nil {
		return domain.Transaction{}, err
	}

	rows, err := r.queries.GetTransactionDetails(ctx, databaseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Transaction{}, transaction.ErrTransactionNotFound
		}

		return domain.Transaction{}, fmt.Errorf(
			"get transaction aggregate: %w",
			err,
		)
	}
	if len(rows) == 0 {
		return domain.Transaction{}, transaction.ErrTransactionNotFound
	}

	first := rows[0]

	transactionID, err := domain.NewTransactionID(
		first.TransactionID.String(),
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	journalEntryID, err := domain.NewJournalEntryID(
		first.JournalEntryID.String(),
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	currency, err := domain.NewCurrency(first.Currency)
	if err != nil {
		return domain.Transaction{}, err
	}

	postings := make([]domain.Posting, 0, len(rows))

	for _, row := range rows {
		postingID, err := domain.NewPostingID(
			row.PostingID.String(),
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		accountID, err := domain.NewAccountID(
			row.AccountID.String(),
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		amount, err := domain.NewMoney(
			currency,
			row.AmountMinorUnits,
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		posting, err := domain.NewPosting(
			postingID,
			accountID,
			domain.PostingDirection(row.Direction),
			amount,
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		postings = append(postings, posting)
	}

	journalEntry, err := domain.NewJournalEntry(
		journalEntryID,
		transactionID,
		currency,
		postings,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	var reversesTransactionID *domain.TransactionID

	if first.ReversesTransactionID.Valid {
		reversalID, err := domain.NewTransactionID(
			first.ReversesTransactionID.String(),
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		reversesTransactionID = &reversalID
	}

	reconstructedTransaction, err := domain.ReconstituteTransaction(
		transactionID,
		first.Description,
		journalEntry,
		reversesTransactionID,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	return reconstructedTransaction, nil
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
