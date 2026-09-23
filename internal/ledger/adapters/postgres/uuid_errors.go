package postgres

import (
	"fmt"

	"financial-ledger/internal/ledger/domain"
)

func invalidUUIDError(kind, value string, cause error) error {
	return fmt.Errorf(
		"%w: convert %s %q to PostgreSQL UUID: %w",
		domain.ErrInvalidID,
		kind,
		value,
		cause,
	)
}
