package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type mockOutboxRepository struct {
	events          []domain.OutboxEvent
	publishedEvents []domain.EventID
	failedEvents    map[domain.EventID]string
	releasedEvents  []domain.EventID
	fetchErr        error
}

func (m *mockOutboxRepository) Save(ctx context.Context, event domain.OutboxEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockOutboxRepository) FetchPending(ctx context.Context, maxRetries int, limit int) ([]domain.OutboxEvent, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.events, nil
}

func (m *mockOutboxRepository) MarkPublished(ctx context.Context, eventID domain.EventID, processedAt time.Time) error {
	m.publishedEvents = append(m.publishedEvents, eventID)
	return nil
}

func (m *mockOutboxRepository) ReleaseClaim(ctx context.Context, eventID domain.EventID, updatedAt time.Time) error {
	m.releasedEvents = append(m.releasedEvents, eventID)
	return nil
}

func (m *mockOutboxRepository) MarkFailed(ctx context.Context, eventID domain.EventID, lastError string, updatedAt time.Time) error {
	if m.failedEvents == nil {
		m.failedEvents = make(map[domain.EventID]string)
	}
	m.failedEvents[eventID] = lastError
	return nil
}

type mockPublisher struct {
	published []domain.OutboxEvent
	err       error
}

func (m *mockPublisher) Publish(ctx context.Context, event domain.OutboxEvent) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, event)
	return nil
}

func TestRelay_ProcessBatch(t *testing.T) {
	t.Parallel()

	eventID, err := domain.NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected event id error: %v", err)
	}

	event, err := domain.NewOutboxEvent(
		eventID,
		"wallet",
		"w-001",
		"WalletBalanceUpdated",
		[]byte(`{"wallet_id":"w-001"}`),
	)
	if err != nil {
		t.Fatalf("unexpected event creation error: %v", err)
	}

	t.Run("successfully processes pending event", func(t *testing.T) {
		t.Parallel()

		repo := &mockOutboxRepository{events: []domain.OutboxEvent{event}}
		pub := &mockPublisher{}
		relay := NewRelay(repo, pub, RelayConfig{}, nil)

		count, err := relay.ProcessBatch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		if len(pub.published) != 1 {
			t.Errorf("published count = %d, want 1", len(pub.published))
		}
		if len(repo.publishedEvents) != 1 || repo.publishedEvents[0] != eventID {
			t.Errorf("expected event %v to be marked published", eventID)
		}
	})

	t.Run("marks event as failed when publisher fails", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("dynamodb error")
		repo := &mockOutboxRepository{events: []domain.OutboxEvent{event}}
		pub := &mockPublisher{err: expectedErr}
		relay := NewRelay(repo, pub, RelayConfig{}, nil)

		count, err := relay.ProcessBatch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
		if repo.failedEvents[eventID] != expectedErr.Error() {
			t.Errorf("lastError = %q, want %q", repo.failedEvents[eventID], expectedErr.Error())
		}
	})

	t.Run("skips failed event if backoff delay has not elapsed", func(t *testing.T) {
		t.Parallel()

		failedEvent, err := domain.ReconstituteOutboxEvent(
			eventID,
			"wallet",
			"w-001",
			"WalletBalanceUpdated",
			[]byte(`{"wallet_id":"w-001"}`),
			domain.EventStatusFailed,
			1,
			"previous error",
			time.Now().UTC().Add(-100*time.Millisecond),
			nil,
			time.Now().UTC(),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		repo := &mockOutboxRepository{events: []domain.OutboxEvent{failedEvent}}
		pub := &mockPublisher{}
		relay := NewRelay(repo, pub, RelayConfig{
			InitialInterval: 5 * time.Second,
		}, nil)

		count, err := relay.ProcessBatch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 0 {
			t.Errorf("count = %d, want 0 (skipped due to backoff)", count)
		}
		if len(pub.published) != 0 {
			t.Errorf("published count = %d, want 0", len(pub.published))
		}
		if len(repo.releasedEvents) != 1 || repo.releasedEvents[0] != eventID {
			t.Errorf("expected event %v claim to be released", eventID)
		}
	})

	t.Run("processes failed event when backoff delay has elapsed", func(t *testing.T) {
		t.Parallel()

		failedEvent, err := domain.ReconstituteOutboxEvent(
			eventID,
			"wallet",
			"w-001",
			"WalletBalanceUpdated",
			[]byte(`{"wallet_id":"w-001"}`),
			domain.EventStatusFailed,
			1,
			"previous error",
			time.Now().UTC().Add(-10*time.Second),
			nil,
			time.Now().UTC().Add(-5*time.Second),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		repo := &mockOutboxRepository{events: []domain.OutboxEvent{failedEvent}}
		pub := &mockPublisher{}
		relay := NewRelay(repo, pub, RelayConfig{
			InitialInterval: 1 * time.Second,
		}, nil)

		count, err := relay.ProcessBatch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		if len(pub.published) != 1 {
			t.Errorf("published count = %d, want 1", len(pub.published))
		}
	})
}
