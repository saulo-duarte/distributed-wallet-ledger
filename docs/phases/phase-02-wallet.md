# Phase 2 — Wallet

## Goal

Build the Wallet product on top of the Ledger while keeping the Ledger as the financial source of truth.

## Implemented scope

- Wallet creation and queries.
- Wallet balance with ledger balance and available balance.
- Deposits from a local clearing account.
- Withdrawals to a local clearing account.
- Wallet-to-wallet transfers.
- Initial idempotency at the PostgreSQL boundary.
- Holds with authorization, release, expiration, and available-balance reservation.
- Hold capture with atomic Ledger posting and status transition.
- PostgreSQL locking for wallet funds and hold authorization.

## Hold semantics

Creating a hold does not create Ledger postings. It reserves funds for a wallet operation:

```text
Available Balance = Ledger Balance - Active Holds
```

An active hold is an `authorized` hold whose expiration time has not been reached. Releasing or expiring a hold makes the amount available again without changing the Ledger.

## Current API

```text
POST /wallets/{walletID}/holds
POST /holds/{holdID}/release
POST /holds/{holdID}/expire
POST /holds/{holdID}/capture
```

## Capture semantics

Capture coordinates the final Ledger posting and the hold state transition in the same PostgreSQL transaction. It uses a deterministic lock order, validates the authorized and non-expired state, and reuses the Ledger idempotency record so a retry does not create duplicate postings.

The repository now includes a separate local checkout Saga that uses these hold semantics. It coordinates mock anti-fraud, a mock payment gateway, capture and compensation; real provider authorization, settlement callbacks and reconciliation remain outside the current scope.

## Testing

The phase includes unit tests for domain and application rules, HTTP handler tests, and PostgreSQL integration tests for:

- active holds affecting available balance;
- idempotent hold authorization;
- idempotency conflicts;
- concurrent hold authorization and insufficient funds;
- release and expiration state transitions.
- atomic capture with Ledger postings and idempotent replay.

## Definition of Done

Phase 2 is complete when:

- a wallet can be created and queried by ID or owner;
- deposits, withdrawals, and wallet-to-wallet transfers use the Ledger as the source of truth;
- wallet balances expose both Ledger Balance and Available Balance;
- concurrent operations cannot overspend the same wallet;
- financial operations are protected by PostgreSQL transactions and initial idempotency;
- holds can be authorized, released, expired, and captured;
- capture atomically creates the Ledger postings and marks the hold as captured;
- invalid, expired, duplicated, and conflicting operations return explicit domain/application errors;
- unit, HTTP, and PostgreSQL integration tests cover the implemented wallet flows, including concurrency and rollback scenarios;
- wallet behavior is documented separately from the local checkout Saga and from derived messaging infrastructure.
