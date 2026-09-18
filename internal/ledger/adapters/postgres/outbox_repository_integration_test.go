//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/domain"
)

func TestOutboxRepository_Integration(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewOutboxRepository(queries)
	ctx := context.Background()

	eventID, err := domain.NewEventID("a0000000-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatalf("unexpected event ID error: %v", err)
	}

	event, err := domain.NewOutboxEvent(
		eventID,
		"wallet",
		"w-001",
		"WalletBalanceUpdated",
		[]byte(`{"wallet_id":"w-001","balance":1000}`),
	)
	if err != nil {
		t.Fatalf("unexpected new outbox event error: %v", err)
	}

	t.Cleanup(func() {
		uuid, _ := eventIDToUUID(eventID)
		_, _ = pool.Exec(ctx, "DELETE FROM outbox_events WHERE id = $1", uuid)
	})

	if err := repository.Save(ctx, event); err != nil {
		t.Fatalf("failed to save outbox event: %v", err)
	}

	pending, err := repository.FetchPending(ctx, 3, 100)
	if err != nil {
		t.Fatalf("failed to fetch pending events: %v", err)
	}

	var found *domain.OutboxEvent
	for i := range pending {
		if pending[i].ID() == eventID {
			found = &pending[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected to find saved event %q in pending list", eventID)
	}

	if found.AggregateType() != "wallet" || found.AggregateID() != "w-001" {
		t.Errorf("unexpected aggregate data: got %s:%s", found.AggregateType(), found.AggregateID())
	}

	now := time.Now().UTC()
	if err := repository.MarkPublished(ctx, eventID, now); err != nil {
		t.Fatalf("failed to mark event published: %v", err)
	}

	pendingAfterPublish, err := repository.FetchPending(ctx, 3, 10)
	if err != nil {
		t.Fatalf("failed to fetch pending after publish: %v", err)
	}

	for _, e := range pendingAfterPublish {
		if e.ID() == eventID {
			t.Fatalf("event %q should not be pending after being published", eventID)
		}
	}
}
