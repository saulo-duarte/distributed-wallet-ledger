# Architecture Decision Records

An Architecture Decision Record (ADR) captures an important choice, its context, considered alternatives, and consequences. ADRs describe why the system is shaped a certain way; they are not a list of technologies to adopt automatically.

## Format

Each ADR should contain:

```text
# ADR-XXXX: Title

## Status
## Context
## Decision
## Alternatives Considered
## Consequences
### Positive
### Negative
```

Use `Proposed` while a decision is being evaluated, `Accepted` once it guides implementation, `Superseded` when a later ADR replaces it, and `Rejected` when it is explicitly not adopted.

## Index

- [0001 — PostgreSQL as Ledger Source of Truth](0001-postgresql-as-ledger-source-of-truth.md)
- [0002 — External Accounts Are Not Local Ledger Accounts](0002-external-accounts-are-not-local-ledger-accounts.md)
