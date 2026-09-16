package postgres

import (
	"context"
	"fmt"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgtype"
)

// AccountRepository is the PostgreSQL adapter for account persistence.
//
// It deliberately contains no business rules. Domain validation is performed
// before this adapter is called; this type only maps the domain object to the
// SQLC-generated parameter type and executes the SQL query.
type AccountRepository struct {
	queries *db.Queries
}

func NewAccountRepository(queries *db.Queries) *AccountRepository {
	return &AccountRepository{
		queries: queries,
	}
}

var _ account.AccountRepository = (*AccountRepository)(nil)

func (r *AccountRepository) Create(
	ctx context.Context,
	acc domain.Account,
) error {
	id, err := accountIDToUUID(acc.ID())
	if err != nil {
		return err
	}

	_, err = r.queries.CreateAccount(ctx, db.CreateAccountParams{
		ID:       id,
		Code:     acc.Code(),
		Name:     acc.Name(),
		Currency: acc.Currency().String(),
	})
	if err != nil {
		return fmt.Errorf("create account %q: %w", acc.Code(), err)
	}

	return nil
}

func accountIDToUUID(id domain.AccountID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf(
			"convert account ID %q to PostgreSQL UUID: %w",
			id.String(),
			err,
		)
	}

	return uuid, nil
}
