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

-- name: ListAccountEntriesFirst :many
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
LIMIT $2;

-- name: ListAccountEntriesAfter :many
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
LIMIT $4;

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
