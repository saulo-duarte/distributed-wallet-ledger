package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HoldRepository struct {
	pool         *pgxpool.Pool
	queries      *db.Queries
	transactions *TransactionRepository
}

func NewHoldRepository(
	pool *pgxpool.Pool,
	queries *db.Queries,
	transactions *TransactionRepository,
) *HoldRepository {
	return &HoldRepository{
		pool:         pool,
		queries:      queries,
		transactions: transactions,
	}
}

var _ wallet.HoldRepository = (*HoldRepository)(nil)
var _ wallet.HoldCaptureRepository = (*HoldRepository)(nil)

func (r *HoldRepository) Authorize(
	ctx context.Context,
	hold domain.Hold,
	idempotencyKey string,
	requestHash string,
) (domain.Hold, error) {
	if r == nil || r.pool == nil || r.queries == nil {
		return domain.Hold{}, fmt.Errorf("hold repository is not configured")
	}

	holdID, err := holdIDToUUID(hold.ID())
	if err != nil {
		return domain.Hold{}, err
	}
	walletID, err := walletIDToUUID(hold.WalletID())
	if err != nil {
		return domain.Hold{}, err
	}

	databaseTx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Hold{}, fmt.Errorf("begin hold authorization: %w", err)
	}
	defer func() {
		_ = databaseTx.Rollback(ctx)
	}()

	queries := r.queries.WithTx(databaseTx)
	lockedWallet, err := queries.LockWalletByID(ctx, walletID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Hold{}, wallet.ErrWalletNotFound
		}
		return domain.Hold{}, fmt.Errorf("lock wallet for hold: %w", err)
	}

	existing, err := queries.GetWalletHoldByIdempotencyKey(
		ctx,
		db.GetWalletHoldByIdempotencyKeyParams{
			WalletID:       walletID,
			IdempotencyKey: idempotencyKey,
		},
	)
	if err == nil {
		if existing.RequestHash != requestHash {
			return domain.Hold{}, wallet.ErrHoldIdempotencyConflict
		}

		existingHold, mapErr := mapWalletHold(existing)
		if mapErr != nil {
			return domain.Hold{}, fmt.Errorf("map existing hold: %w", mapErr)
		}

		return existingHold, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Hold{}, fmt.Errorf("get hold idempotency key: %w", err)
	}

	balance, err := queries.GetLedgerBalance(ctx, lockedWallet.LedgerAccountID)
	if err != nil {
		return domain.Hold{}, fmt.Errorf("get balance for hold: %w", err)
	}

	availableBalance := balance.TotalCredits -
		balance.TotalDebits -
		balance.ActiveHolds
	if availableBalance < hold.Amount().AmountMinorUnits() {
		return domain.Hold{}, wallet.ErrInsufficientFunds
	}

	persisted, err := queries.CreateWalletHold(
		ctx,
		db.CreateWalletHoldParams{
			ID:               holdID,
			WalletID:         walletID,
			AmountMinorUnits: hold.Amount().AmountMinorUnits(),
			Currency:         hold.Amount().Currency().String(),
			Status:           string(domain.HoldStatusAuthorized),
			IdempotencyKey:   idempotencyKey,
			RequestHash:      requestHash,
			ExpiresAt: pgtype.Timestamptz{
				Time:  hold.ExpiresAt(),
				Valid: true,
			},
		},
	)
	if err != nil {
		return domain.Hold{}, fmt.Errorf("create wallet hold: %w", err)
	}

	if err := databaseTx.Commit(ctx); err != nil {
		return domain.Hold{}, fmt.Errorf("commit hold authorization: %w", err)
	}

	createdHold, err := mapWalletHold(persisted)
	if err != nil {
		return domain.Hold{}, fmt.Errorf("map created hold: %w", err)
	}

	return createdHold, nil
}

func (r *HoldRepository) GetByID(
	ctx context.Context,
	holdID domain.HoldID,
) (domain.Hold, error) {
	if r == nil || r.queries == nil {
		return domain.Hold{}, fmt.Errorf("hold repository is not configured")
	}

	databaseID, err := holdIDToUUID(holdID)
	if err != nil {
		return domain.Hold{}, err
	}

	persisted, err := r.queries.GetWalletHoldByID(ctx, databaseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Hold{}, wallet.ErrHoldNotFound
		}

		return domain.Hold{}, fmt.Errorf("get wallet hold %q: %w", holdID, err)
	}

	foundHold, err := mapWalletHold(persisted)
	if err != nil {
		return domain.Hold{}, fmt.Errorf("map wallet hold %q: %w", holdID, err)
	}

	return foundHold, nil
}

func (r *HoldRepository) UpdateStatus(
	ctx context.Context,
	holdID domain.HoldID,
	status domain.HoldStatus,
) error {
	if r == nil || r.queries == nil {
		return fmt.Errorf("hold repository is not configured")
	}

	if err := status.Validate(); err != nil {
		return err
	}

	databaseID, err := holdIDToUUID(holdID)
	if err != nil {
		return err
	}

	rowsAffected, err := r.queries.UpdateWalletHoldStatus(
		ctx,
		db.UpdateWalletHoldStatusParams{
			ID:     databaseID,
			Status: string(status),
		},
	)
	if err != nil {
		return fmt.Errorf("update wallet hold status: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrHoldNotAuthorized
	}

	return nil
}

func (r *HoldRepository) Capture(
	ctx context.Context,
	holdID domain.HoldID,
	postedTransaction domain.Transaction,
	idempotencyKey string,
	requestHash string,
) error {
	if r == nil || r.pool == nil || r.queries == nil || r.transactions == nil {
		return fmt.Errorf("hold capture repository is not configured")
	}

	databaseHoldID, err := holdIDToUUID(holdID)
	if err != nil {
		return err
	}

	// Read the wallet reference before opening the transaction. The wallet row
	// is locked first below so capture and normal Ledger posting use the same
	// lock order and avoid introducing a lock inversion.
	unlockedHold, err := r.queries.GetWalletHoldByID(ctx, databaseHoldID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return wallet.ErrHoldNotFound
		}
		return fmt.Errorf("get hold before capture: %w", err)
	}

	databaseTx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin hold capture: %w", err)
	}
	defer func() {
		_ = databaseTx.Rollback(ctx)
	}()

	queries := r.queries.WithTx(databaseTx)
	lockedWallet, err := queries.LockWalletByID(ctx, unlockedHold.WalletID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return wallet.ErrWalletNotFound
		}
		return fmt.Errorf("lock wallet for hold capture: %w", err)
	}

	lockedHold, err := queries.GetWalletHoldByIDForUpdate(ctx, databaseHoldID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return wallet.ErrHoldNotFound
		}
		return fmt.Errorf("lock hold for capture: %w", err)
	}

	if lockedHold.WalletID != lockedWallet.ID {
		return fmt.Errorf("hold wallet does not match locked wallet")
	}

	if lockedHold.Status == string(domain.HoldStatusCaptured) {
		replayed, err := checkIdempotencyWithQueries(
			ctx,
			queries,
			postedTransaction,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		if !replayed {
			return domain.ErrHoldNotAuthorized
		}

		if err := databaseTx.Commit(ctx); err != nil {
			return fmt.Errorf("commit replayed hold capture: %w", err)
		}
		return nil
	}

	if lockedHold.Status != string(domain.HoldStatusAuthorized) {
		return domain.ErrHoldNotAuthorized
	}

	if !lockedHold.ExpiresAt.Valid ||
		!time.Now().UTC().Before(lockedHold.ExpiresAt.Time.UTC()) {
		return domain.ErrHoldExpired
	}

	if err := r.transactions.postInTransaction(
		ctx,
		queries,
		postedTransaction,
		idempotencyKey,
		requestHash,
	); err != nil {
		return err
	}

	rowsAffected, err := queries.UpdateWalletHoldStatus(
		ctx,
		db.UpdateWalletHoldStatusParams{
			ID:     databaseHoldID,
			Status: string(domain.HoldStatusCaptured),
		},
	)
	if err != nil {
		return fmt.Errorf("mark hold as captured: %w", err)
	}
	if rowsAffected != 1 {
		return domain.ErrHoldNotAuthorized
	}

	if err := databaseTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit hold capture: %w", err)
	}

	return nil
}

func mapWalletHold(row db.WalletHold) (domain.Hold, error) {
	if !row.ID.Valid || !row.WalletID.Valid {
		return domain.Hold{}, fmt.Errorf("wallet hold IDs are null")
	}
	if !row.ExpiresAt.Valid || !row.CreatedAt.Valid {
		return domain.Hold{}, fmt.Errorf("wallet hold timestamps are invalid")
	}

	holdID, err := domain.NewHoldID(row.ID.String())
	if err != nil {
		return domain.Hold{}, err
	}
	walletID, err := domain.NewWalletID(row.WalletID.String())
	if err != nil {
		return domain.Hold{}, err
	}
	currency, err := domain.NewCurrency(row.Currency)
	if err != nil {
		return domain.Hold{}, err
	}
	amount, err := domain.NewMoney(currency, row.AmountMinorUnits)
	if err != nil {
		return domain.Hold{}, err
	}

	hold, err := domain.ReconstituteHold(
		holdID,
		walletID,
		amount,
		domain.HoldStatus(row.Status),
		row.CreatedAt.Time.UTC(),
		row.ExpiresAt.Time.UTC(),
	)
	if err != nil {
		return domain.Hold{}, err
	}

	return hold, nil
}

func holdIDToUUID(id domain.HoldID) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf(
			"convert hold ID %q to PostgreSQL UUID: %w",
			id.String(),
			err,
		)
	}

	return uuid, nil
}
