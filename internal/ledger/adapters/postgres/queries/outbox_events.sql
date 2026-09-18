-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    retry_count,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    retry_count,
    last_error,
    created_at,
    processed_at,
    updated_at;

-- name: FetchPendingOutboxEvents :many
SELECT
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    retry_count,
    last_error,
    created_at,
    processed_at,
    updated_at
FROM outbox_events
WHERE status IN ('pending', 'failed')
  AND retry_count < $1
ORDER BY created_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET
    status = 'published',
    processed_at = $2,
    updated_at = $2
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET
    status = 'failed',
    retry_count = retry_count + 1,
    last_error = $2,
    updated_at = $3
WHERE id = $1;
