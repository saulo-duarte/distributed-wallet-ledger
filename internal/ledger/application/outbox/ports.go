package outbox

import (
	"context"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type OutboxRepository interface {
	Save(ctx context.Context, event domain.OutboxEvent) error
	FetchPending(ctx context.Context, maxRetries int, limit int) ([]domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID domain.EventID, processedAt time.Time) error
	MarkFailed(ctx context.Context, eventID domain.EventID, lastError string, updatedAt time.Time) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event domain.OutboxEvent) error
}
