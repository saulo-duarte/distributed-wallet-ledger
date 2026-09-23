# ADR-0001: PostgreSQL as Ledger Source of Truth

## Status

Accepted

## Context

The ledger needs durable accounting facts with atomic multi-row writes, consistency checks, unique constraints, foreign keys and predictable transaction semantics. The application now also has derived projections and asynchronous integrations, so the system needs one unambiguous financial source of truth.

## Decision

Use PostgreSQL as the financial source of truth for the ledger and wallet. Persist transactions, journal entries, postings, accounts, wallets, holds, outbox events and idempotency records in PostgreSQL. Keep SQL explicit and access it through SQLC-generated code in the PostgreSQL adapter. DynamoDB projections and SNS/SQS messages are derived from committed PostgreSQL state.

## Alternatives Considered

### DynamoDB

DynamoDB is used for the wallet-balance read projection, but not as the source of truth. Choosing it for the accounting write model would make relational integrity and atomic accounting across related rows less direct to express.

### Event store

An event store could support event sourcing and replay. Full event sourcing is a larger modeling and operational decision than Phase 1 requires, and it would introduce a second major concept before the basic ledger invariants are proven.

### PostgreSQL

PostgreSQL provides ACID transactions, foreign keys, unique constraints, check constraints, mature operational tooling and clear SQL for the relational shape of a double-entry ledger. It remains a strong fit for the current consistency boundary while supporting the outbox and derived projections.

## Consequences

### Positive

- A journal entry and its postings can be committed atomically.
- Relational integrity is explicit and close to the data.
- SQL remains inspectable and SQLC avoids ORM behavior being hidden.
- Local development and integration testing are straightforward with Docker.
- DynamoDB, SNS and SQS can be rebuilt or replayed from PostgreSQL-backed events without becoming financial authorities.

### Negative

- The initial design has a single primary persistence dependency.
- Horizontal read scaling and asynchronous projections are not solved yet.
- Some cross-row accounting invariants still require domain/application validation in addition to database constraints.
- Future migration to another source of truth, if ever needed, will require deliberate data and consistency work.
