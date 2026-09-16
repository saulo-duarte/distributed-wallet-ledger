-- name: CreateIdempotencyKey :exec
INSERT INTO idempotency_keys (
    scope,
    idempotency_key,
    request_hash,
    transaction_id
)
VALUES ($1, $2, $3, $4);

-- name: GetIdempotencyKey :one
SELECT scope, idempotency_key, request_hash, transaction_id, created_at
FROM idempotency_keys
WHERE scope = $1 AND idempotency_key = $2;
