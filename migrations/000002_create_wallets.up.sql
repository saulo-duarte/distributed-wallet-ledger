CREATE TABLE wallets (
    id UUID PRIMARY KEY,
    owner_id TEXT NOT NULL,
    ledger_account_id UUID NOT NULL UNIQUE
        REFERENCES accounts (id),

    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT wallets_currency_uppercase
        CHECK (currency = UPPER(currency)),

    CONSTRAINT wallets_status_valid
        CHECK (status IN ('open', 'suspended', 'closed'))
);

CREATE INDEX wallets_owner_id_idx
    ON wallets (owner_id);
