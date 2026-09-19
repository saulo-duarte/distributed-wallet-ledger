package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type ExpireHoldExecutor interface {
	Execute(ctx context.Context, holdID domain.HoldID) (domain.Hold, error)
}

type ExpiredHoldLister interface {
	ListExpiredHolds(ctx context.Context, limit int32) ([]domain.HoldID, error)
}

type HoldExpirationConfig struct {
	Interval  time.Duration
	BatchSize int32
}

func (c HoldExpirationConfig) withDefaults() HoldExpirationConfig {
	if c.Interval <= 0 {
		c.Interval = 10 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	return c
}

type HoldExpirationWorker struct {
	lister   ExpiredHoldLister
	executor ExpireHoldExecutor
	cfg      HoldExpirationConfig
	logger   *slog.Logger
}

func NewHoldExpirationWorker(
	lister ExpiredHoldLister,
	executor ExpireHoldExecutor,
	cfg HoldExpirationConfig,
	logger *slog.Logger,
) *HoldExpirationWorker {
	return &HoldExpirationWorker{
		lister:   lister,
		executor: executor,
		cfg:      cfg.withDefaults(),
		logger:   logger,
	}
}

func (w *HoldExpirationWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.ProcessExpiredHolds(ctx); err != nil {
				if !errors.Is(err, context.Canceled) && w.logger != nil {
					w.logger.ErrorContext(ctx, "hold_expiration_failed", slog.Any("error", err))
				}
			}
		}
	}
}

func (w *HoldExpirationWorker) ProcessExpiredHolds(ctx context.Context) error {
	if w.lister == nil {
		return nil
	}

	expiredIDs, err := w.lister.ListExpiredHolds(ctx, w.cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("list expired holds: %w", err)
	}

	for _, holdID := range expiredIDs {
		_, err := w.executor.Execute(ctx, holdID)
		if err != nil {
			if w.logger != nil {
				w.logger.WarnContext(
					ctx,
					"expire_hold_failed",
					slog.String("hold_id", holdID.String()),
					slog.Any("error", err),
				)
			}
			continue
		}
		if w.logger != nil {
			w.logger.InfoContext(
				ctx,
				"hold_expired_automatically",
				slog.String("hold_id", holdID.String()),
			)
		}
	}

	return nil
}
