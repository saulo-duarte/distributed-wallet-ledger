# Roadmap

The roadmap describes learning goals, not irreversible technology commitments. Each phase should begin with a concrete problem statement and end with documented evidence that the added complexity was useful.

| Phase | Objective | Problem we expect to solve | Concepts studied | Probable technologies |
| --- | --- | --- | --- | --- |
| 1 — Ledger Core | Build a correct accounting core | We need durable, balanced, immutable financial facts | Double-entry, ACID, idempotency, concurrency, testing | Go, PostgreSQL, SQLC, Docker Compose |
| 2 — Wallet | Build the first product on the ledger | Users need deposits, withdrawals, transfers, balances, and holds | Product boundaries, available vs ledger balance, concurrency | Go HTTP API, PostgreSQL |
| 3 — CQRS | Separate write and read concerns | Read workloads or projections become complex enough to justify separation | CQRS, projections, eventual consistency, rebuilds | DynamoDB as a possible read model |
| 4 — Event Driven | Integrate asynchronous consumers | Persisting a change and publishing it independently creates a failure window | Outbox, at-least-once delivery, consumers, DLQ | SNS/SQS or Pub/Sub |
| 5 — Resilience | Contain dependency and consumer failures | Retries and slow/unavailable dependencies threaten availability | Timeout, retry, backoff, jitter, circuit breaker, idempotent consumers | Go libraries and failure tests |
| 6 — Distributed Workflows | Coordinate multi-step financial flows | Holds, authorization, capture, and settlement can span independent steps | Saga, compensation, workflow state | Messaging/workflow tooling, if justified |
| 7 — Platform | Operate the system consistently | Multiple deployable components need repeatable environments and policies | Kubernetes, Helm, IaC, config, scaling, disruption budgets | Kubernetes, Helm, Terraform, Local AWS emulator |
| 8 — Reliability | Make behavior observable and test failure | Distributed behavior cannot be understood from logs alone | Tracing, metrics, logs, chaos, load and failure testing | OpenTelemetry, load/chaos tools |
| Future — Investments | Explore another product on the ledger | Investment products introduce new instruments and settlement concerns | Brokerage and investment-domain modeling | To be decided from concrete needs |

Only Phase 1 is current. Nothing in later phases is implemented by this repository setup.
