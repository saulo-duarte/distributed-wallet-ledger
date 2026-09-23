# Current Scope

This document describes what is implemented in the repository today. The system is a Go modular monolith with PostgreSQL as the financial source of truth and optional local AWS-compatible services for projections and messaging.

## Ledger

- Ledger accounts, business transactions, journal entries and debit/credit postings.
- Double-entry validation, positive integer minor units and currency checks.
- Atomic PostgreSQL transactions with SQLC-generated access code.
- Immutable posted facts and reversal through new compensating entries.
- Database-backed idempotency and conflict detection.
- Account posting queries and transaction details.

## Wallet

- Wallet creation and owner queries.
- Deposits, withdrawals and wallet-to-wallet transfers.
- Ledger balance and available balance.
- Authorization holds with release, expiration and capture.
- PostgreSQL locking to protect funds and prevent concurrent overspending.

## Checkout Saga

- Hold authorization as the first step.
- Local mock anti-fraud evaluation.
- Local mock payment-gateway processing protected by a circuit breaker.
- Hold capture after successful gateway processing.
- Hold release as compensation after rejection or gateway failure.
- Deterministic server-generated IDs for safe retries with the same idempotency key.

## Event-driven read model

- Transactional outbox records events in the same PostgreSQL transaction as the financial write.
- Relay with retry backoff and claim protection for concurrent workers.
- SNS publication and SQS projection consumption.
- DynamoDB wallet-balance projection with conditional ordering protection.
- PostgreSQL fallback when a projection is unavailable or stale.

## Platform and reliability

- Structured request logging with request and trace IDs.
- Prometheus metrics and OpenTelemetry tracing with Jaeger support.
- Circuit-breaker state and Saga/outbox/SQS metrics.
- Docker Compose and Terraform for local dependencies.
- Kubernetes and Helm deployment manifests with HPA, PDB and observability resources.
- A PowerShell Kubernetes lab and a Go load generator with stored reports.

## Explicitly outside the current scope

- Real external payment providers, payment rails, callbacks and settlement.
- External account ownership, reconciliation and chargeback operations.
- User authentication, authorization and tenancy isolation.
- Multi-currency conversion and foreign-exchange accounting.
- Full event sourcing and independently deployable microservices.
