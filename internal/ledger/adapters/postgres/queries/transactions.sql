-- name: CreateTransaction :exec
INSERT INTO transactions (id, description)
VALUES ($1, $2);

-- name: GetTransaction :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: GetTransactionByIdempotencyKey :one
SELECT t.*
FROM transactions AS t
JOIN idempotency_keys AS i ON i.transaction_id = t.id
WHERE i.scope = $1 AND i.idempotency_key = $2;

-- name: GetTransactionDetails :many
SELECT
    t.id AS transaction_id,
    t.description,
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
ORDER BY p.id;
