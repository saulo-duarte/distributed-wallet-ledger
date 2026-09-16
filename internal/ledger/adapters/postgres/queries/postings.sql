-- name: CreatePosting :exec
INSERT INTO postings (
    id,
    journal_entry_id,
    account_id,
    direction,
    amount_minor_units
)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAccountPostings :many
SELECT
    p.id,
    p.journal_entry_id,
    p.account_id,
    p.direction,
    p.amount_minor_units,
    p.created_at
FROM postings AS p
WHERE p.account_id = $1
ORDER BY p.created_at, p.id;

-- name: ListJournalEntryPostings :many
SELECT
    p.id,
    p.journal_entry_id,
    p.account_id,
    p.direction,
    p.amount_minor_units,
    p.created_at
FROM postings AS p
WHERE p.journal_entry_id = $1
ORDER BY p.id;
