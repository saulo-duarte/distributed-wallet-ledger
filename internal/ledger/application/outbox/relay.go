package outbox

import (
	"context"
	"financial-ledger/internal/ledger/domain"
	"fmt"
	"log/slog"
	"time"
)

type RelayConfig struct {
	PollInterval       time.Duration
	BatchSize          int
	MaxRetries         int
	InitialInterval    time.Duration
	MaxInterval        time.Duration
	BackoffMultiplier  float64
}

type Relay struct {
	repo      OutboxRepository
	publisher EventPublisher
	config    RelayConfig
	backoff   BackoffStrategy
	logger    *slog.Logger
}

func NewRelay(
	repo OutboxRepository,
	publisher EventPublisher,
	cfg RelayConfig,
	logger *slog.Logger,
) *Relay {
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
	if err := r.publisher.Publish(ctx, event); err != nil {
		markErr := r.repo.MarkFailed(ctx, event.ID(), err.Error(), now)
		if markErr != nil {
			return fmt.Errorf("publish failed (%w) and mark failed (%w)", err, markErr)
		}
		return err
	}

	if err := r.repo.MarkPublished(ctx, event.ID(), now); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
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
