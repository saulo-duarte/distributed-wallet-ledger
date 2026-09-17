-- name: CreateWallet :one
INSERT INTO wallets (
    id,
    owner_id,
    ledger_account_id,
    currency,
    status
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING
    id,
    owner_id,
    ledger_account_id,
    currency,
    status,
    created_at;

-- name: GetWalletByID :one
SELECT
    id,
    owner_id,
    ledger_account_id,
    currency,
    status,
    created_at
FROM wallets
WHERE id = $1;

-- name: LockWalletByLedgerAccountID :one
SELECT
    id,
    owner_id,
    ledger_account_id,
    currency,
    status,
    created_at
FROM wallets
WHERE ledger_account_id = $1
FOR UPDATE;

-- name: ListWalletsByOwnerID :many
SELECT
    id,
    owner_id,
    ledger_account_id,
    currency,
    status,
    created_at
FROM wallets
WHERE owner_id = $1
ORDER BY created_at DESC, id DESC;
