-- name: CreateIdempotencyKey :exec
INSERT INTO idempotency_keys (
    scope,
    idempotency_key,
    request_hash,
    transaction_id
)
VALUES ($1, $2, $3, $4);
