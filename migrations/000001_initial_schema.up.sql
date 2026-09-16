CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT accounts_currency_uppercase CHECK (currency = UPPER(currency)),
    CONSTRAINT accounts_status_valid CHECK (status IN ('open', 'closed'))
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    description TEXT NOT NULL,
    reverses_transaction_id UUID UNIQUE REFERENCES transactions (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_cannot_reverse_itself CHECK (id <> reverses_transaction_id)
);

CREATE TABLE journal_entries (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL UNIQUE REFERENCES transactions (id),
    currency CHAR(3) NOT NULL,
    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT journal_entries_currency_uppercase CHECK (currency = UPPER(currency))
);

CREATE TABLE postings (
    id UUID PRIMARY KEY,
    journal_entry_id UUID NOT NULL REFERENCES journal_entries (id),
    account_id UUID NOT NULL REFERENCES accounts (id),
    direction TEXT NOT NULL,
    amount_minor_units BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT postings_direction_valid CHECK (direction IN ('debit', 'credit')),
    CONSTRAINT postings_amount_positive CHECK (amount_minor_units > 0)
);

CREATE INDEX postings_account_id_created_at_idx
    ON postings (account_id, created_at, id);

CREATE INDEX postings_journal_entry_id_idx
    ON postings (journal_entry_id);

CREATE TABLE idempotency_keys (
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    transaction_id UUID NOT NULL UNIQUE REFERENCES transactions (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scope, idempotency_key)
);
