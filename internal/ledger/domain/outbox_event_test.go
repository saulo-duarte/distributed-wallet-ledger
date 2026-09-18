package domain

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestEventStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status EventStatus
		want   bool
	}{
		{
			name:   "pending is valid",
			status: EventStatusPending,
			want:   true,
		},
		{
			name:   "processing is valid",
			status: EventStatusProcessing,
			want:   true,
		},
		{
			name:   "published is valid",
			status: EventStatusPublished,
			want:   true,
		},
		{
			name:   "failed is valid",
			status: EventStatusFailed,
			want:   true,
		},
		{
			name:   "unknown status is invalid",
			status: EventStatus("unknown"),
			want:   false,
		},
		{
			name:   "empty status is invalid",
			status: EventStatus(""),
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.status.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOutboxEvent(t *testing.T) {
	t.Parallel()

	id, err := NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected error creating event id: %v", err)
	}

	tests := []struct {
		name          string
		id            EventID
		aggregateType string
		aggregateID   string
		eventType     string
		payload       []byte
		wantErr       error
	}{
		{
			name:          "creates valid outbox event",
			id:            id,
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			wantErr:       nil,
		},
		{
			name:          "rejects zero event id",
			id:            EventID(""),
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			wantErr:       ErrInvalidID,
		},
		{
			name:          "rejects empty aggregate type",
			id:            id,
			aggregateType: "   ",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			wantErr:       ErrEmptyAggregateType,
		},
		{
			name:          "rejects empty aggregate id",
			id:            id,
			aggregateType: "wallet",
			aggregateID:   "   ",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			wantErr:       ErrEmptyAggregateID,
		},
		{
			name:          "rejects empty event type",
			id:            id,
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "   ",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			wantErr:       ErrEmptyEventType,
		},
		{
			name:          "rejects empty payload",
			id:            id,
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       nil,
			wantErr:       ErrEmptyEventPayload,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event, err := NewOutboxEvent(
				tt.id,
				tt.aggregateType,
				tt.aggregateID,
				tt.eventType,
				tt.payload,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewOutboxEvent() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if event.ID() != tt.id {
					t.Errorf("ID() = %v, want %v", event.ID(), tt.id)
				}
				if event.AggregateType() != tt.aggregateType {
					t.Errorf("AggregateType() = %v, want %v", event.AggregateType(), tt.aggregateType)
				}
				if event.AggregateID() != tt.aggregateID {
					t.Errorf("AggregateID() = %v, want %v", event.AggregateID(), tt.aggregateID)
				}
				if event.EventType() != tt.eventType {
					t.Errorf("EventType() = %v, want %v", event.EventType(), tt.eventType)
				}
				if !bytes.Equal(event.Payload(), tt.payload) {
					t.Errorf("Payload() = %s, want %s", string(event.Payload()), string(tt.payload))
				}
				if event.Status() != EventStatusPending {
					t.Errorf("Status() = %v, want %v", event.Status(), EventStatusPending)
				}
				if event.RetryCount() != 0 {
					t.Errorf("RetryCount() = %v, want 0", event.RetryCount())
				}
				if event.LastError() != "" {
					t.Errorf("LastError() = %v, want empty", event.LastError())
				}
				if event.ProcessedAt() != nil {
					t.Errorf("ProcessedAt() = %v, want nil", event.ProcessedAt())
				}
				if event.CreatedAt().IsZero() {
					t.Error("CreatedAt() should not be zero")
				}
				if event.UpdatedAt().IsZero() {
					t.Error("UpdatedAt() should not be zero")
				}
			}
		})
	}
}

func TestReconstituteOutboxEvent(t *testing.T) {
	t.Parallel()

	id, err := NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected error creating event id: %v", err)
	}

	now := time.Now().UTC()
	processed := now.Add(time.Minute)

	tests := []struct {
		name          string
		id            EventID
		aggregateType string
		aggregateID   string
		eventType     string
		payload       []byte
		status        EventStatus
		retryCount    int
		lastError     string
		createdAt     time.Time
		processedAt   *time.Time
		updatedAt     time.Time
		wantErr       error
	}{
		{
			name:          "reconstitutes valid published event",
			id:            id,
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			status:        EventStatusPublished,
			retryCount:    1,
			lastError:     "transient error",
			createdAt:     now,
			processedAt:   &processed,
			updatedAt:     processed,
			wantErr:       nil,
		},
		{
			name:          "rejects zero id",
			id:            EventID(""),
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			status:        EventStatusPending,
			createdAt:     now,
			updatedAt:     now,
			wantErr:       ErrInvalidID,
		},
		{
			name:          "rejects invalid status",
			id:            id,
			aggregateType: "wallet",
			aggregateID:   "wallet-001",
			eventType:     "WalletBalanceUpdated",
			payload:       []byte(`{"wallet_id":"wallet-001"}`),
			status:        EventStatus("invalid"),
			createdAt:     now,
			updatedAt:     now,
			wantErr:       ErrInvalidEventStatus,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event, err := ReconstituteOutboxEvent(
				tt.id,
				tt.aggregateType,
				tt.aggregateID,
				tt.eventType,
				tt.payload,
				tt.status,
				tt.retryCount,
				tt.lastError,
				tt.createdAt,
				tt.processedAt,
				tt.updatedAt,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ReconstituteOutboxEvent() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if event.Status() != tt.status {
					t.Errorf("Status() = %v, want %v", event.Status(), tt.status)
				}
				if event.RetryCount() != tt.retryCount {
					t.Errorf("RetryCount() = %v, want %v", event.RetryCount(), tt.retryCount)
				}
				if event.LastError() != tt.lastError {
					t.Errorf("LastError() = %v, want %v", event.LastError(), tt.lastError)
				}
			}
		})
	}
}

func TestOutboxEvent_MarkPublished(t *testing.T) {
	t.Parallel()

	id, err := NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected error creating event id: %v", err)
	}

	event, err := NewOutboxEvent(
		id,
		"wallet",
		"wallet-001",
		"WalletBalanceUpdated",
		[]byte(`{"wallet_id":"wallet-001"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error creating event: %v", err)
	}

	now := time.Now().UTC()
	if err := event.MarkPublished(now); err != nil {
		t.Fatalf("unexpected error marking published: %v", err)
	}

	if event.Status() != EventStatusPublished {
		t.Errorf("Status() = %v, want %v", event.Status(), EventStatusPublished)
	}
	if event.ProcessedAt() == nil || !event.ProcessedAt().Equal(now) {
		t.Errorf("ProcessedAt() = %v, want %v", event.ProcessedAt(), now)
	}
	if !event.UpdatedAt().Equal(now) {
		t.Errorf("UpdatedAt() = %v, want %v", event.UpdatedAt(), now)
	}

	if err := event.MarkPublished(now); !errors.Is(err, ErrEventAlreadyProcessed) {
		t.Fatalf("expected ErrEventAlreadyProcessed, got: %v", err)
	}
}

func TestOutboxEvent_MarkFailed(t *testing.T) {
	t.Parallel()

	id, err := NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected error creating event id: %v", err)
	}

	event, err := NewOutboxEvent(
		id,
		"wallet",
		"wallet-001",
		"WalletBalanceUpdated",
		[]byte(`{"wallet_id":"wallet-001"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error creating event: %v", err)
	}

	now := time.Now().UTC()
	errDesc := "connection refused to dynamodb"
	event.MarkFailed(errDesc, now)

	if event.Status() != EventStatusFailed {
		t.Errorf("Status() = %v, want %v", event.Status(), EventStatusFailed)
	}
	if event.LastError() != errDesc {
		t.Errorf("LastError() = %v, want %v", event.LastError(), errDesc)
	}
	if event.RetryCount() != 1 {
		t.Errorf("RetryCount() = %v, want 1", event.RetryCount())
	}
	if !event.UpdatedAt().Equal(now) {
		t.Errorf("UpdatedAt() = %v, want %v", event.UpdatedAt(), now)
	}
}
