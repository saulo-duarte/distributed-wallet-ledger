-- name: CreateJournalEntry :exec
INSERT INTO journal_entries (id, transaction_id, currency)
VALUES ($1, $2, $3);
