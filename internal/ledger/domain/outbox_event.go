package domain

import (
	"strings"
	"time"
)

type EventStatus string

const (
	EventStatusPending    EventStatus = "pending"
	EventStatusProcessing EventStatus = "processing"
	EventStatusPublished  EventStatus = "published"
	EventStatusFailed     EventStatus = "failed"
)

func (s EventStatus) IsValid() bool {
	switch s {
	case EventStatusPending, EventStatusProcessing, EventStatusPublished, EventStatusFailed:
		return true
	default:
		return false
	}
}

type OutboxEvent struct {
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
}

func NewOutboxEvent(
	id EventID,
	aggregateType string,
	aggregateID string,
	eventType string,
	payload []byte,
) (OutboxEvent, error) {
	if id.IsZero() {
		return OutboxEvent{}, ErrInvalidID
	}
	cleanAggregateType := strings.TrimSpace(aggregateType)
	if cleanAggregateType == "" {
		return OutboxEvent{}, ErrEmptyAggregateType
	}
	cleanAggregateID := strings.TrimSpace(aggregateID)
	if cleanAggregateID == "" {
		return OutboxEvent{}, ErrEmptyAggregateID
	}
	cleanEventType := strings.TrimSpace(eventType)
	if cleanEventType == "" {
		return OutboxEvent{}, ErrEmptyEventType
	}
	if len(payload) == 0 {
		return OutboxEvent{}, ErrEmptyEventPayload
	}
	now := time.Now().UTC()
	return OutboxEvent{
		id:            id,
		aggregateType: cleanAggregateType,
		aggregateID:   cleanAggregateID,
		eventType:     cleanEventType,
		payload:       payload,
		status:        EventStatusPending,
		retryCount:    0,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}
func ReconstituteOutboxEvent(
	id EventID,
	aggregateType string,
	aggregateID string,
	eventType string,
	payload []byte,
	status EventStatus,
	retryCount int,
	lastError string,
	createdAt time.Time,
	processedAt *time.Time,
	updatedAt time.Time,
) (OutboxEvent, error) {
	if id.IsZero() {
		return OutboxEvent{}, ErrInvalidID
	}
	if strings.TrimSpace(aggregateType) == "" {
		return OutboxEvent{}, ErrEmptyAggregateType
	}
	if strings.TrimSpace(aggregateID) == "" {
		return OutboxEvent{}, ErrEmptyAggregateID
	}
	if strings.TrimSpace(eventType) == "" {
		return OutboxEvent{}, ErrEmptyEventType
	}
	if len(payload) == 0 {
		return OutboxEvent{}, ErrEmptyEventPayload
	}
	if !status.IsValid() {
		return OutboxEvent{}, ErrInvalidEventStatus
	}
	return OutboxEvent{
		id:            id,
		aggregateType: aggregateType,
		aggregateID:   aggregateID,
		eventType:     eventType,
		payload:       payload,
		status:        status,
		retryCount:    retryCount,
		lastError:     lastError,
		createdAt:     createdAt,
		processedAt:   processedAt,
		updatedAt:     updatedAt,
	}, nil
}
func (e OutboxEvent) ID() EventID {
	return e.id
}
func (e OutboxEvent) AggregateType() string {
	return e.aggregateType
}
func (e OutboxEvent) AggregateID() string {
	return e.aggregateID
}
func (e OutboxEvent) EventType() string {
	return e.eventType
}
func (e OutboxEvent) Payload() []byte {
	return e.payload
}
func (e OutboxEvent) Status() EventStatus {
	return e.status
}
func (e OutboxEvent) RetryCount() int {
	return e.retryCount
}
func (e OutboxEvent) LastError() string {
	return e.lastError
}
func (e OutboxEvent) CreatedAt() time.Time {
	return e.createdAt
}
func (e OutboxEvent) ProcessedAt() *time.Time {
	return e.processedAt
}
func (e OutboxEvent) UpdatedAt() time.Time {
	return e.updatedAt
}
func (e *OutboxEvent) MarkPublished(now time.Time) error {
	if e.status == EventStatusPublished {
		return ErrEventAlreadyProcessed
	}
	e.status = EventStatusPublished
	utc := now.UTC()
	e.processedAt = &utc
	e.updatedAt = utc
	return nil
}
func (e *OutboxEvent) MarkFailed(errDescription string, now time.Time) {
	e.status = EventStatusFailed
	e.lastError = errDescription
	e.retryCount++
	e.updatedAt = now.UTC()
}
