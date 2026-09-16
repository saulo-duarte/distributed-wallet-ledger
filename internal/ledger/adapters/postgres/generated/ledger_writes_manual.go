// Code generated manually as a temporary fallback because sqlc's Windows
// WebAssembly parser cannot initialize in the current development environment.
// The source of truth remains the SQL files under ../queries.
package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createTransaction = `
INSERT INTO transactions (id, description)
VALUES ($1, $2)
`

type CreateTransactionParams struct {
	ID          pgtype.UUID `json:"id"`
	Description string      `json:"description"`
}

func (q *Queries) CreateTransaction(
	ctx context.Context,
	arg CreateTransactionParams,
) error {
	_, err := q.db.Exec(ctx, createTransaction, arg.ID, arg.Description)
	return err
}

const createJournalEntry = `
INSERT INTO journal_entries (id, transaction_id, currency)
VALUES ($1, $2, $3)
`

type CreateJournalEntryParams struct {
	ID            pgtype.UUID `json:"id"`
	TransactionID pgtype.UUID `json:"transaction_id"`
	Currency      string      `json:"currency"`
}

func (q *Queries) CreateJournalEntry(
	ctx context.Context,
	arg CreateJournalEntryParams,
) error {
	_, err := q.db.Exec(
		ctx,
		createJournalEntry,
		arg.ID,
		arg.TransactionID,
		arg.Currency,
	)
	return err
}

const createPosting = `
INSERT INTO postings (
    id,
    journal_entry_id,
    account_id,
    direction,
    amount_minor_units
)
VALUES ($1, $2, $3, $4, $5)
`

type CreatePostingParams struct {
	ID               pgtype.UUID `json:"id"`
	JournalEntryID   pgtype.UUID `json:"journal_entry_id"`
	AccountID        pgtype.UUID `json:"account_id"`
	Direction        string      `json:"direction"`
	AmountMinorUnits int64       `json:"amount_minor_units"`
}

func (q *Queries) CreatePosting(
	ctx context.Context,
	arg CreatePostingParams,
) error {
	_, err := q.db.Exec(
		ctx,
		createPosting,
		arg.ID,
		arg.JournalEntryID,
		arg.AccountID,
		arg.Direction,
		arg.AmountMinorUnits,
	)
	return err
}

const createIdempotencyKey = `
INSERT INTO idempotency_keys (
    scope,
    idempotency_key,
    request_hash,
    transaction_id
)
VALUES ($1, $2, $3, $4)
`

type CreateIdempotencyKeyParams struct {
	Scope          string      `json:"scope"`
	IdempotencyKey string      `json:"idempotency_key"`
	RequestHash    string      `json:"request_hash"`
	TransactionID  pgtype.UUID `json:"transaction_id"`
}

func (q *Queries) CreateIdempotencyKey(
	ctx context.Context,
	arg CreateIdempotencyKeyParams,
) error {
	_, err := q.db.Exec(
		ctx,
		createIdempotencyKey,
		arg.Scope,
		arg.IdempotencyKey,
		arg.RequestHash,
		arg.TransactionID,
	)
	return err
}
