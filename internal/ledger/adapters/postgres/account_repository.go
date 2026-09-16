package postgres

import (
	"context"
	"fmt"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgtype"
)

// AccountRepository is the PostgreSQL adapter for account persistence.
//
// It deliberately contains no business rules. Domain validation is performed
// before this adapter is called; this type only maps the domain object to the
// SQLC-generated parameter type and executes the SQL query.
type AccountRepository struct {
	queries *db.Queries
}

func NewAccountRepository(queries *db.Queries) *AccountRepository {
	return &AccountRepository{
		queries: queries,
	}
}

var _ account.AccountRepository = (*AccountRepository)(nil)
var _ account.AccountEntriesRepository = (*AccountRepository)(nil)

func (r *AccountRepository) Create(
	ctx context.Context,
	acc domain.Account,
) error {
	id, err := accountIDToUUID(acc.ID())
	if err != nil {
		return err
	}

	_, err = r.queries.CreateAccount(ctx, db.CreateAccountParams{
		ID:       id,
		Code:     acc.Code(),
		Name:     acc.Name(),
		Currency: acc.Currency().String(),
	})
	if err != nil {
		return fmt.Errorf("create account %q: %w", acc.Code(), err)
	}

	return nil
}

func (r *AccountRepository) ListEntries(
	ctx context.Context,
	accountID domain.AccountID,
	limit int,
	cursor *account.AccountEntriesCursor,
) ([]account.AccountEntry, error) {
	databaseAccountID, err := accountIDToUUID(accountID)
	if err != nil {
		return nil, err
	}

	var rows []db.AccountEntry
	if cursor == nil {
		rows, err = r.queries.ListAccountEntriesFirst(
			ctx,
			db.ListAccountEntriesFirstParams{
				AccountID: databaseAccountID,
				Limit:     int32(limit),
			},
		)
	} else {
		postingID, conversionErr := postingIDToUUID(cursor.PostingID)
		if conversionErr != nil {
			return nil, conversionErr
		}

		rows, err = r.queries.ListAccountEntriesAfter(
			ctx,
			db.ListAccountEntriesAfterParams{
				AccountID: databaseAccountID,
				CreatedAt: pgtype.Timestamptz{
					Time:  cursor.CreatedAt,
					Valid: true,
				},
				PostingID: postingID,
				Limit:     int32(limit),
			},
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list account entries: %w", err)
	}

	entries := make([]account.AccountEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := mapAccountEntry(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func mapAccountEntry(row db.AccountEntry) (account.AccountEntry, error) {
	postingID, err := uuidToPostingID(row.PostingID)
	if err != nil {
		return account.AccountEntry{}, err
	}

	journalEntryID, err := uuidToJournalEntryID(row.JournalEntryID)
	if err != nil {
		return account.AccountEntry{}, err
	}

	transactionID, err := uuidToTransactionID(row.TransactionID)
	if err != nil {
		return account.AccountEntry{}, err
	}

	currency, err := domain.NewCurrency(row.Currency)
	if err != nil {
		return account.AccountEntry{}, err
	}

	if !row.CreatedAt.Valid {
		return account.AccountEntry{}, fmt.Errorf("account entry created_at is null")
	}

	return account.AccountEntry{
		PostingID:      postingID,
		JournalEntryID: journalEntryID,
		TransactionID:  transactionID,
		Description:    row.Description,
		Currency:       currency,
		Direction:      domain.PostingDirection(row.Direction),
		AmountMinor:    row.AmountMinorUnits,
		CreatedAt:      row.CreatedAt.Time.UTC(),
	}, nil
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

func uuidToPostingID(id pgtype.UUID) (domain.PostingID, error) {
	if !id.Valid {
		return "", fmt.Errorf("posting ID is null")
	}
	return domain.NewPostingID(id.String())
}

func uuidToJournalEntryID(id pgtype.UUID) (domain.JournalEntryID, error) {
	if !id.Valid {
		return "", fmt.Errorf("journal entry ID is null")
	}
	return domain.NewJournalEntryID(id.String())
}

func uuidToTransactionID(id pgtype.UUID) (domain.TransactionID, error) {
	if !id.Valid {
		return "", fmt.Errorf("transaction ID is null")
	}
	return domain.NewTransactionID(id.String())
}

func accountIDToUUID(id domain.AccountID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf(
			"convert account ID %q to PostgreSQL UUID: %w",
			id.String(),
			err,
		)
	}

	return uuid, nil
}
