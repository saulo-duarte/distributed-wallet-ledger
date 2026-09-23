package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/observability"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RelayConfig struct {
	PollInterval      time.Duration
	BatchSize         int
	MaxRetries        int
	InitialInterval   time.Duration
	MaxInterval       time.Duration
	BackoffMultiplier float64
}

type Relay struct {
	repo      OutboxRepository
	publisher EventPublisher
	config    RelayConfig
	backoff   BackoffStrategy
	logger    *slog.Logger
	metrics   *observability.Metrics
}

func NewRelay(
	repo OutboxRepository,
	publisher EventPublisher,
	cfg RelayConfig,
	logger *slog.Logger,
	metrics ...*observability.Metrics,
) *Relay {
	var collector *observability.Metrics
	if len(metrics) > 0 {
		collector = metrics[0]
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 100 * time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	backoff := NewExponentialBackoff(
		cfg.InitialInterval,
		cfg.MaxInterval,
		cfg.BackoffMultiplier,
	)
	return &Relay{
		repo:      repo,
		publisher: publisher,
		config:    cfg,
		backoff:   backoff,
		logger:    logger,
		metrics:   collector,
	}
}

func (r *Relay) ProcessBatch(ctx context.Context) (int, error) {
	events, err := r.repo.FetchPending(ctx, r.config.MaxRetries, r.config.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("fetch pending outbox events: %w", err)
	}
	if len(events) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	processedCount := 0
	for _, event := range events {
		if !r.isEligibleForProcessing(event, now) {
			// FetchPending claims rows in the database so another relay cannot
			// process them concurrently. A failed event can still be inside its
			// backoff window, so release that claim without changing its retry
			// timestamp or count.
			if err := r.repo.ReleaseClaim(ctx, event.ID(), event.UpdatedAt()); err != nil {
				if r.logger != nil {
					r.logger.ErrorContext(
						ctx,
						"outbox_event_claim_release_failed",
						slog.String("event_id", event.ID().String()),
						slog.Any("error", err),
					)
				}
			}
			continue
		}

		if err := r.processEvent(ctx, event, now); err != nil {
			if r.logger != nil {
				r.logger.ErrorContext(
					ctx,
					"outbox_event_processing_failed",
					slog.String("event_id", event.ID().String()),
					slog.String("event_type", event.EventType()),
					slog.Any("error", err),
				)
			}
			continue
		}
		processedCount++
	}

	return processedCount, nil
}

func (r *Relay) isEligibleForProcessing(event domain.OutboxEvent, now time.Time) bool {
	if event.Status() == domain.EventStatusPending {
		return true
	}
	if event.RetryCount() == 0 {
		return true
	}

	delay := r.backoff.NextDelay(event.RetryCount())
	nextRetryAt := event.UpdatedAt().Add(delay)
	return !now.Before(nextRetryAt)
}

func (r *Relay) processEvent(ctx context.Context, event domain.OutboxEvent, now time.Time) error {
	ctx, span := observability.StartSpan(ctx, "outbox.publish_event",
		trace.WithAttributes(
			attribute.String("event.id", event.ID().String()),
			attribute.String("event.type", event.EventType()),
			attribute.String("aggregate.id", event.AggregateID()),
		),
	)
	defer span.End()

	if err := r.publisher.Publish(ctx, event); err != nil {
		if r.metrics != nil {
			r.metrics.RecordOutboxPublish(event.EventType(), "failed")
		}
		markErr := r.repo.MarkFailed(ctx, event.ID(), err.Error(), now)
		if markErr != nil {
			return fmt.Errorf("publish failed (%w) and mark failed (%w)", err, markErr)
		}
		return err
	}

	if err := r.repo.MarkPublished(ctx, event.ID(), now); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	if r.metrics != nil {
		r.metrics.RecordOutboxPublish(event.EventType(), "published")
	}
	if r.logger != nil {
		r.logger.DebugContext(
			ctx,
			"outbox_event_published",
			slog.String("event_id", event.ID().String()),
			slog.String("event_type", event.EventType()),
			slog.String("aggregate_id", event.AggregateID()),
		)
	}

	return nil
}

func (r *Relay) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, _ = r.ProcessBatch(ctx)
		}
	}
}
