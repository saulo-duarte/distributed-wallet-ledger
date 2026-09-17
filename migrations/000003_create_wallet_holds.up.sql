CREATE TABLE wallet_holds (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets (id),
    amount_minor_units BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'authorized',
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT wallet_holds_amount_positive
        CHECK (amount_minor_units > 0),
    CONSTRAINT wallet_holds_currency_uppercase
        CHECK (currency = UPPER(currency)),
    CONSTRAINT wallet_holds_status_valid
        CHECK (status IN ('authorized', 'captured', 'released', 'expired')),
    CONSTRAINT wallet_holds_idempotency_key_not_empty
        CHECK (length(trim(idempotency_key)) > 0),
    CONSTRAINT wallet_holds_request_hash_not_empty
        CHECK (length(trim(request_hash)) > 0)
);

CREATE UNIQUE INDEX wallet_holds_wallet_id_idempotency_key_idx
    ON wallet_holds (wallet_id, idempotency_key);

CREATE INDEX wallet_holds_wallet_id_status_expires_at_idx
    ON wallet_holds (wallet_id, status, expires_at);
