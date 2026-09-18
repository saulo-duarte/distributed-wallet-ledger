package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

type mockBalanceProjector struct {
	projectedWalletID domain.WalletID
	calls             int
	err               error
}

func (m *mockBalanceProjector) ProjectBalance(
	ctx context.Context,
	walletID domain.WalletID,
) (WalletBalance, error) {
	m.calls++
	m.projectedWalletID = walletID
	if m.err != nil {
		return WalletBalance{}, m.err
	}
	return WalletBalance{}, nil
}

func TestWalletBalanceEventPublisher(t *testing.T) {
	t.Parallel()

	eventID, err := domain.NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected event id error: %v", err)
	}

	t.Run("successfully projects balance for valid event", func(t *testing.T) {
		t.Parallel()

		event, err := domain.NewOutboxEvent(
			eventID,
			"wallet",
			"wallet-001",
			"WalletBalanceUpdated",
			[]byte(`{"wallet_id":"wallet-001"}`),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mockProjector := &mockBalanceProjector{}
		publisher := NewWalletBalanceEventPublisher(mockProjector)

		err = publisher.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected publish error: %v", err)
		}

		if mockProjector.calls != 1 {
			t.Errorf("projector calls = %d, want 1", mockProjector.calls)
		}
		if mockProjector.projectedWalletID.String() != "wallet-001" {
			t.Errorf("projected wallet ID = %q, want %q", mockProjector.projectedWalletID, "wallet-001")
		}
	})

	t.Run("ignores unrelated event types", func(t *testing.T) {
		t.Parallel()

		event, err := domain.NewOutboxEvent(
			eventID,
			"account",
			"acc-001",
			"AccountCreated",
			[]byte(`{"account_id":"acc-001"}`),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mockProjector := &mockBalanceProjector{}
		publisher := NewWalletBalanceEventPublisher(mockProjector)

		err = publisher.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected publish error: %v", err)
		}

		if mockProjector.calls != 0 {
			t.Errorf("projector calls = %d, want 0", mockProjector.calls)
		}
	})

	t.Run("returns error on invalid payload JSON", func(t *testing.T) {
		t.Parallel()

		event, err := domain.NewOutboxEvent(
			eventID,
			"wallet",
			"wallet-001",
			"WalletBalanceUpdated",
			[]byte("invalid-json"),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mockProjector := &mockBalanceProjector{}
		publisher := NewWalletBalanceEventPublisher(mockProjector)

		err = publisher.Publish(context.Background(), event)
		if err == nil {
			t.Fatal("expected error on invalid json, got nil")
		}
	})

	t.Run("propagates projector error", func(t *testing.T) {
		t.Parallel()

		event, err := domain.NewOutboxEvent(
			eventID,
			"wallet",
			"wallet-001",
			"WalletBalanceUpdated",
			[]byte(`{"wallet_id":"wallet-001"}`),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedErr := errors.New("dynamo timeout")
		mockProjector := &mockBalanceProjector{err: expectedErr}
		publisher := NewWalletBalanceEventPublisher(mockProjector)

		err = publisher.Publish(context.Background(), event)
		if err == nil {
			t.Fatal("expected projector error, got nil")
		}
	})
}
