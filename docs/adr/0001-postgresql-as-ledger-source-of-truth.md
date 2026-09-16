# ADR-0001: PostgreSQL as Ledger Source of Truth

## Status

Accepted

## Context

The Phase 1 ledger needs durable accounting facts with atomic multi-row writes, consistency checks, unique constraints, foreign keys, and predictable transaction semantics. The project is also intentionally starting as a modular monolith so that domain and persistence behavior can be understood before distributed infrastructure is introduced.

## Decision

Use PostgreSQL as the financial source of truth for the Phase 1 Ledger. Persist transactions, journal entries, postings, accounts, and initial idempotency records in PostgreSQL. Keep SQL explicit and access it through SQLC-generated code in the PostgreSQL adapter.

## Alternatives Considered

### DynamoDB

DynamoDB can be an excellent choice for high-scale access patterns and is a likely candidate for a future CQRS read model. At this stage, the ledger's primary challenge is relational integrity and atomic accounting across related rows, not independent key-value read scaling. Choosing it as the source of truth now would make the intended consistency and relational invariants less direct to express.

### Event store

An event store could support event sourcing and replay. Full event sourcing is a larger modeling and operational decision than Phase 1 requires, and it would introduce a second major concept before the basic ledger invariants are proven.

### PostgreSQL

PostgreSQL provides ACID transactions, foreign keys, unique constraints, check constraints, mature operational tooling, and clear SQL for the relational shape of a double-entry ledger. It is a strong fit for learning and for the current consistency boundary without preventing future projections or integrations.

## Consequences

### Positive

- A journal entry and its postings can be committed atomically.
- Relational integrity is explicit and close to the data.
- SQL remains inspectable and SQLC avoids ORM behavior being hidden.
- Local development and integration testing are straightforward with Docker.
- A later read model can consume or rebuild from PostgreSQL-backed ledger data.

### Negative

- The initial design has a single primary persistence dependency.
- Horizontal read scaling and asynchronous projections are not solved yet.
- Some cross-row accounting invariants still require domain/application validation in addition to database constraints.
- Future migration to another source of truth, if ever needed, will require deliberate data and consistency work.
