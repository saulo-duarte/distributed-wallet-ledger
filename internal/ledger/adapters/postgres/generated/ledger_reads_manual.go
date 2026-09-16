// Code generated manually as a temporary fallback because sqlc's Windows
// WebAssembly parser cannot initialize in the current development environment.
// The source of truth remains the SQL files under ../queries.
package db

import (
	"context"

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
