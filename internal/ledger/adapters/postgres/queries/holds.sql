-- name: CreateWalletHold :one
INSERT INTO wallet_holds (
    id,
    wallet_id,
    amount_minor_units,
    currency,
    status,
    idempotency_key,
    request_hash,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING
    id,
    wallet_id,
    amount_minor_units,
    currency,
    status,
    idempotency_key,
    request_hash,
    expires_at,
    created_at,
    updated_at;

-- name: GetWalletHoldByID :one
SELECT
    id,
    wallet_id,
    amount_minor_units,
    currency,
    status,
    idempotency_key,
    request_hash,
    expires_at,
    created_at,
    updated_at
FROM wallet_holds
WHERE id = $1;

-- name: GetWalletHoldByIDForUpdate :one
SELECT
    id,
    wallet_id,
    amount_minor_units,
    currency,
    status,
    idempotency_key,
    request_hash,
    expires_at,
    created_at,
    updated_at
FROM wallet_holds
WHERE id = $1
FOR UPDATE;

-- name: GetWalletHoldByIdempotencyKey :one
SELECT
    id,
    wallet_id,
    amount_minor_units,
    currency,
    status,
    idempotency_key,
    request_hash,
    expires_at,
    created_at,
    updated_at
FROM wallet_holds
WHERE wallet_id = $1
  AND idempotency_key = $2;

-- name: GetActiveWalletHoldsTotal :one
SELECT COALESCE(SUM(amount_minor_units), 0)::BIGINT AS total_amount
FROM wallet_holds
WHERE wallet_id = $1
  AND status = 'authorized'
  AND expires_at > NOW();

-- name: UpdateWalletHoldStatus :execrows
UPDATE wallet_holds
SET status = $2,
    updated_at = NOW()
WHERE id = $1
  AND status = 'authorized';
