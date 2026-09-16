// Code generated manually as a temporary fallback because sqlc's Windows
// WebAssembly parser cannot initialize in the current development environment.
// The source of truth remains the SQL files under ../queries.
package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AccountEntry struct {
	PostingID        pgtype.UUID        `json:"posting_id"`
	JournalEntryID   pgtype.UUID        `json:"journal_entry_id"`
	TransactionID    pgtype.UUID        `json:"transaction_id"`
	Description      string             `json:"description"`
	Currency         string             `json:"currency"`
	Direction        string             `json:"direction"`
	AmountMinorUnits int64              `json:"amount_minor_units"`
	CreatedAt        pgtype.Timestamptz `json:"created_at"`
}

const listAccountEntriesFirst = `
SELECT
    p.id AS posting_id,
    p.journal_entry_id,
    j.transaction_id,
    t.description,
    j.currency,
    p.direction,
    p.amount_minor_units,
    p.created_at
FROM postings AS p
JOIN journal_entries AS j ON j.id = p.journal_entry_id
JOIN transactions AS t ON t.id = j.transaction_id
WHERE p.account_id = $1
ORDER BY p.created_at DESC, p.id DESC
LIMIT $2
`

type ListAccountEntriesFirstParams struct {
	AccountID pgtype.UUID `json:"account_id"`
	Limit     int32       `json:"limit"`
}

func (q *Queries) ListAccountEntriesFirst(
	ctx context.Context,
	arg ListAccountEntriesFirstParams,
) ([]AccountEntry, error) {
	rows, err := q.db.Query(
		ctx,
		listAccountEntriesFirst,
		arg.AccountID,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]AccountEntry, 0)
	for rows.Next() {
		var entry AccountEntry
		if err := rows.Scan(
			&entry.PostingID,
			&entry.JournalEntryID,
			&entry.TransactionID,
			&entry.Description,
			&entry.Currency,
			&entry.Direction,
			&entry.AmountMinorUnits,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

const listAccountEntriesAfter = `
SELECT
    p.id AS posting_id,
    p.journal_entry_id,
    j.transaction_id,
    t.description,
    j.currency,
    p.direction,
    p.amount_minor_units,
    p.created_at
FROM postings AS p
JOIN journal_entries AS j ON j.id = p.journal_entry_id
JOIN transactions AS t ON t.id = j.transaction_id
WHERE p.account_id = $1
  AND (
      p.created_at < $2
      OR (p.created_at = $2 AND p.id < $3)
  )
ORDER BY p.created_at DESC, p.id DESC
LIMIT $4
`

type ListAccountEntriesAfterParams struct {
	AccountID pgtype.UUID        `json:"account_id"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	PostingID pgtype.UUID        `json:"posting_id"`
	Limit     int32              `json:"limit"`
}

func (q *Queries) ListAccountEntriesAfter(
	ctx context.Context,
	arg ListAccountEntriesAfterParams,
) ([]AccountEntry, error) {
	rows, err := q.db.Query(
		ctx,
		listAccountEntriesAfter,
		arg.AccountID,
		arg.CreatedAt,
		arg.PostingID,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]AccountEntry, 0)
	for rows.Next() {
		var entry AccountEntry
		if err := rows.Scan(
			&entry.PostingID,
			&entry.JournalEntryID,
			&entry.TransactionID,
			&entry.Description,
			&entry.Currency,
			&entry.Direction,
			&entry.AmountMinorUnits,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

type TransactionDetailsRow struct {
	TransactionID         pgtype.UUID
	Description           string
	ReversesTransactionID pgtype.UUID
	TransactionCreatedAt  pgtype.Timestamptz
	JournalEntryID        pgtype.UUID
	Currency              string
	PostedAt              pgtype.Timestamptz
	PostingID             pgtype.UUID
	AccountID             pgtype.UUID
	Direction             string
	AmountMinorUnits      int64
}

const getTransactionDetails = `
SELECT
    t.id AS transaction_id,
    t.description,
    t.reverses_transaction_id,
    t.created_at AS transaction_created_at,
    j.id AS journal_entry_id,
    j.currency,
    j.posted_at,
    p.id AS posting_id,
    p.account_id,
    p.direction,
    p.amount_minor_units
FROM transactions AS t
JOIN journal_entries AS j ON j.transaction_id = t.id
JOIN postings AS p ON p.journal_entry_id = j.id
WHERE t.id = $1
ORDER BY p.id
`

func (q *Queries) GetTransactionDetails(
	ctx context.Context,
	id pgtype.UUID,
) ([]TransactionDetailsRow, error) {
	rows, err := q.db.Query(ctx, getTransactionDetails, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TransactionDetailsRow, 0)
	for rows.Next() {
		var item TransactionDetailsRow
		if err := rows.Scan(
			&item.TransactionID,
			&item.Description,
			&item.ReversesTransactionID,
			&item.TransactionCreatedAt,
			&item.JournalEntryID,
			&item.Currency,
			&item.PostedAt,
			&item.PostingID,
			&item.AccountID,
			&item.Direction,
			&item.AmountMinorUnits,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, pgx.ErrNoRows
	}

	return items, nil
}
