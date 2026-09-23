package postgres

import (
	"context"
	"errors"
	"fmt"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// WalletRepository is the PostgreSQL adapter for wallet persistence.
//
// The adapter only translates the domain object into SQLC parameters and
// executes the persistence operation. Wallet business rules remain in the
// domain and application layers.
type WalletRepository struct {
	queries *db.Queries
}

func NewWalletRepository(queries *db.Queries) *WalletRepository {
	return &WalletRepository{
		queries: queries,
	}
}

var _ wallet.WalletRepository = (*WalletRepository)(nil)
var _ wallet.WalletReader = (*WalletRepository)(nil)

func (r *WalletRepository) Create(
	ctx context.Context,
	w domain.Wallet,
) error {
	if r == nil || r.queries == nil {
		return fmt.Errorf("wallet repository is not configured")
	}

	walletID, err := walletIDToUUID(w.ID())
	if err != nil {
		return err
	}

	ledgerAccountID, err := accountIDToUUID(w.LedgerAccountID())
	if err != nil {
		return err
	}

	_, err = r.queries.CreateWallet(ctx, db.CreateWalletParams{
		ID:              walletID,
		OwnerID:         w.OwnerID().String(),
		LedgerAccountID: ledgerAccountID,
		Currency:        w.Currency().String(),
		Status:          string(w.Status()),
	})
	if err != nil {
		return fmt.Errorf("create wallet %q: %w", w.ID(), err)
	}

	return nil
}

func (r *WalletRepository) GetByID(
	ctx context.Context,
	walletID domain.WalletID,
) (domain.Wallet, error) {
	if r == nil || r.queries == nil {
		return domain.Wallet{}, fmt.Errorf("wallet repository is not configured")
	}

	databaseWalletID, err := walletIDToUUID(walletID)
	if err != nil {
		return domain.Wallet{}, err
	}

	persisted, err := r.queries.GetWalletByID(ctx, databaseWalletID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Wallet{}, wallet.ErrWalletNotFound
		}

		return domain.Wallet{}, fmt.Errorf("get wallet %q: %w", walletID, err)
	}

	reconstituted, err := mapWallet(persisted)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("map wallet %q: %w", walletID, err)
	}

	return reconstituted, nil
}

func (r *WalletRepository) ListByOwnerID(
	ctx context.Context,
	ownerID domain.OwnerID,
) ([]domain.Wallet, error) {
	if r == nil || r.queries == nil {
		return nil, fmt.Errorf("wallet repository is not configured")
	}

	rows, err := r.queries.ListWalletsByOwnerID(ctx, ownerID.String())
	if err != nil {
		return nil, fmt.Errorf("list wallets by owner %q: %w", ownerID, err)
	}

	wallets := make([]domain.Wallet, 0, len(rows))
	for _, row := range rows {
		reconstituted, err := mapWallet(row)
		if err != nil {
			return nil, fmt.Errorf("map wallet for owner %q: %w", ownerID, err)
		}

		wallets = append(wallets, reconstituted)
	}

	return wallets, nil
}

func mapWallet(row db.Wallet) (domain.Wallet, error) {
	if !row.ID.Valid {
		return domain.Wallet{}, fmt.Errorf("wallet ID is null")
	}
	if !row.LedgerAccountID.Valid {
		return domain.Wallet{}, fmt.Errorf("wallet ledger account ID is null")
	}

	walletID, err := domain.NewWalletID(row.ID.String())
	if err != nil {
		return domain.Wallet{}, err
	}

	ownerID, err := domain.NewOwnerID(row.OwnerID)
	if err != nil {
		return domain.Wallet{}, err
	}

	ledgerAccountID, err := domain.NewAccountID(row.LedgerAccountID.String())
	if err != nil {
		return domain.Wallet{}, err
	}

	currency, err := domain.NewCurrency(row.Currency)
	if err != nil {
		return domain.Wallet{}, err
	}

	reconstituted, err := domain.ReconstituteWallet(
		walletID,
		ownerID,
		ledgerAccountID,
		currency,
		domain.WalletStatus(row.Status),
	)
	if err != nil {
		return domain.Wallet{}, err
	}

	return reconstituted, nil
}

func walletIDToUUID(id domain.WalletID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, invalidUUIDError("wallet ID", id.String(), err)
	}

	return uuid, nil
}

func (r *WalletRepository) GetLedgerBalance(
	ctx context.Context,
	accountID domain.AccountID,
) (wallet.LedgerBalanceSnapshot, error) {
	if r == nil || r.queries == nil {
		return wallet.LedgerBalanceSnapshot{}, fmt.Errorf(
			"wallet repository is not configured",
		)
	}

	databaseAccountID, err := accountIDToUUID(accountID)
	if err != nil {
		return wallet.LedgerBalanceSnapshot{}, err
	}

	balance, err := r.queries.GetLedgerBalance(
		ctx,
		databaseAccountID,
	)
	if err != nil {
		return wallet.LedgerBalanceSnapshot{}, fmt.Errorf(
			"get ledger balance: %w",
			err,
		)
	}

	return wallet.LedgerBalanceSnapshot{
		TotalDebits:           balance.TotalDebits,
		TotalCredits:          balance.TotalCredits,
		ActiveHoldsMinorUnits: balance.ActiveHolds,
	}, nil
}

var _ wallet.WalletBalanceReader = (*WalletRepository)(nil)
