CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT outbox_events_status_valid
        CHECK (status IN ('pending', 'processing', 'published', 'failed')),
    CONSTRAINT outbox_events_aggregate_type_not_empty
        CHECK (length(trim(aggregate_type)) > 0),
    CONSTRAINT outbox_events_aggregate_id_not_empty
        CHECK (length(trim(aggregate_id)) > 0),
    CONSTRAINT outbox_events_event_type_not_empty
        CHECK (length(trim(event_type)) > 0)
);

CREATE INDEX outbox_events_status_created_at_idx
    ON outbox_events (status, created_at)
    WHERE status IN ('pending', 'processing');