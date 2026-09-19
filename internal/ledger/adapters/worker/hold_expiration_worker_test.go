package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type fakeExpiredHoldLister struct {
	holdsToReturn []domain.HoldID
	err           error
	callsCount    int
}

func (f *fakeExpiredHoldLister) ListExpiredHolds(
	_ context.Context,
	_ int32,
) ([]domain.HoldID, error) {
	f.callsCount++
	if f.err != nil {
		return nil, f.err
	}
	return f.holdsToReturn, nil
}

type fakeExpireHoldExecutor struct {
	executedIDs []domain.HoldID
	err         error
}

func (f *fakeExpireHoldExecutor) Execute(
	_ context.Context,
	holdID domain.HoldID,
) (domain.Hold, error) {
	if f.err != nil {
		return domain.Hold{}, f.err
	}
	f.executedIDs = append(f.executedIDs, holdID)
	return domain.Hold{}, nil
}

func TestHoldExpirationWorker_ProcessExpiredHolds(t *testing.T) {
	t.Parallel()

	h1 := domain.HoldID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	h2 := domain.HoldID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")

	t.Run("successfully processes all expired holds", func(t *testing.T) {
		t.Parallel()

		lister := &fakeExpiredHoldLister{holdsToReturn: []domain.HoldID{h1, h2}}
		executor := &fakeExpireHoldExecutor{}

		worker := NewHoldExpirationWorker(lister, executor, HoldExpirationConfig{
			BatchSize: 10,
		}, nil)

		err := worker.ProcessExpiredHolds(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lister.callsCount != 1 {
			t.Fatalf("expected 1 lister call, got %d", lister.callsCount)
		}
		if len(executor.executedIDs) != 2 {
			t.Fatalf("expected 2 executed holds, got %d", len(executor.executedIDs))
		}
	})

	t.Run("propagates lister error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("database connection failed")
		lister := &fakeExpiredHoldLister{err: expectedErr}
		executor := &fakeExpireHoldExecutor{}

		worker := NewHoldExpirationWorker(lister, executor, HoldExpirationConfig{}, nil)

		err := worker.ProcessExpiredHolds(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("continues processing remaining holds when one fails", func(t *testing.T) {
		t.Parallel()

		lister := &fakeExpiredHoldLister{holdsToReturn: []domain.HoldID{h1, h2}}
		executor := &fakeExpireHoldExecutor{err: errors.New("hold already released")}

		worker := NewHoldExpirationWorker(lister, executor, HoldExpirationConfig{}, nil)

		err := worker.ProcessExpiredHolds(context.Background())
		if err != nil {
			t.Fatalf("expected nil error on single hold failure, got %v", err)
		}
	})

	t.Run("stops promptly on context cancellation in Start", func(t *testing.T) {
		t.Parallel()

		lister := &fakeExpiredHoldLister{}
		executor := &fakeExpireHoldExecutor{}

		worker := NewHoldExpirationWorker(lister, executor, HoldExpirationConfig{
			Interval: 100 * time.Millisecond,
		}, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := worker.Start(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
