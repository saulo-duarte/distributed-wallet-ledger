// Code generated manually as a temporary fallback because sqlc's Windows
// WebAssembly parser cannot initialize in the current development environment.
// The source of truth remains the SQL files under ../queries.
package db

import (
	"context"
)

const getIdempotencyKey = `
SELECT scope, idempotency_key, request_hash, transaction_id, created_at
FROM idempotency_keys
WHERE scope = $1 AND idempotency_key = $2
`

type GetIdempotencyKeyParams struct {
	Scope          string `json:"scope"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (q *Queries) GetIdempotencyKey(
	ctx context.Context,
	arg GetIdempotencyKeyParams,
) (IdempotencyKey, error) {
	row := q.db.QueryRow(
		ctx,
		getIdempotencyKey,
		arg.Scope,
		arg.IdempotencyKey,
	)

	var item IdempotencyKey
	err := row.Scan(
		&item.Scope,
		&item.IdempotencyKey,
		&item.RequestHash,
		&item.TransactionID,
		&item.CreatedAt,
	)

	return item, err
}
