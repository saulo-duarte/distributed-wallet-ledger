package postgres

import (
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

func TestInvalidUUIDErrorIsAnInvalidDomainID(t *testing.T) {
	err := invalidUUIDError("wallet ID", "wallet-001", errors.New("invalid UUID"))

	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected invalid UUID error to wrap domain.ErrInvalidID: %v", err)
	}
}
