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
