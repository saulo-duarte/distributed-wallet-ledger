package postgres

import (
	"context"
	"fmt"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/outbox"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgtype"
)

type OutboxRepository struct {
	queries *db.Queries
}

func NewOutboxRepository(queries *db.Queries) *OutboxRepository {
	return &OutboxRepository{
		queries: queries,
	}
}

var _ outbox.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Save(
	ctx context.Context,
	event domain.OutboxEvent,
) error {
	id, err := eventIDToUUID(event.ID())
	if err != nil {
		return err
	}

	_, err = r.queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		ID:            id,
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID(),
		EventType:     event.EventType(),
		Payload:       event.Payload(),
		Status:        string(event.Status()),
		RetryCount:    int32(event.RetryCount()),
		CreatedAt:     pgtype.Timestamptz{Time: event.CreatedAt(), Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Time: event.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create outbox event %q: %w", event.ID(), err)
	}

	return nil
}

func (r *OutboxRepository) FetchPending(
	ctx context.Context,
	maxRetries int,
	limit int,
) ([]domain.OutboxEvent, error) {
	rows, err := r.queries.FetchPendingOutboxEvents(ctx, db.FetchPendingOutboxEventsParams{
		RetryCount: int32(maxRetries),
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox events: %w", err)
	}

	events := make([]domain.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		eventID, err := uuidToEventID(row.ID)
		if err != nil {
			return nil, err
		}

		var processedAt *time.Time
		if row.ProcessedAt.Valid {
			t := row.ProcessedAt.Time
			processedAt = &t
		}

		var lastError string
		if row.LastError.Valid {
			lastError = row.LastError.String
		}

		event, err := domain.ReconstituteOutboxEvent(
			eventID,
			row.AggregateType,
			row.AggregateID,
			row.EventType,
			row.Payload,
			domain.EventStatus(row.Status),
			int(row.RetryCount),
			lastError,
			row.CreatedAt.Time,
			processedAt,
			row.UpdatedAt.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("reconstitute outbox event %q: %w", eventID, err)
		}

		events = append(events, event)
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(
	ctx context.Context,
	eventID domain.EventID,
	processedAt time.Time,
) error {
	id, err := eventIDToUUID(eventID)
	if err != nil {
		return err
	}

	err = r.queries.MarkOutboxEventPublished(ctx, db.MarkOutboxEventPublishedParams{
		ID:          id,
		ProcessedAt: pgtype.Timestamptz{Time: processedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("mark outbox event %q published: %w", eventID, err)
	}

	return nil
}

func (r *OutboxRepository) MarkFailed(
	ctx context.Context,
	eventID domain.EventID,
	lastError string,
	updatedAt time.Time,
) error {
	id, err := eventIDToUUID(eventID)
	if err != nil {
		return err
	}

	err = r.queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID:        id,
		LastError: pgtype.Text{String: lastError, Valid: lastError != ""},
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("mark outbox event %q failed: %w", eventID, err)
	}

	return nil
}

func eventIDToUUID(id domain.EventID) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf("convert event ID %q to PostgreSQL UUID: %w", id.String(), err)
	}
	return uuid, nil
}

func uuidToEventID(id pgtype.UUID) (domain.EventID, error) {
	if !id.Valid {
		return "", fmt.Errorf("event ID is null")
	}
	return domain.NewEventID(id.String())
}
