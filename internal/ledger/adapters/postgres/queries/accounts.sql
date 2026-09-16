-- name: CreateAccount :one
INSERT INTO accounts (id, code, name, currency)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAccount :one
SELECT *
FROM accounts
WHERE id = $1;

-- name: GetAccountByCode :one
SELECT *
FROM accounts
WHERE code = $1;
