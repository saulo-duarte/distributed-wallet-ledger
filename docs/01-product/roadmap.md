# Roadmap

The roadmap records the learning path that produced the current repository. It is historical context for the implementation, not a list of features that the README promises for the future.

| Phase | Objective | Problem we expect to solve | Concepts studied | Probable technologies |
| --- | --- | --- | --- | --- |
| 1 — Ledger Core | Build a correct accounting core | We need durable, balanced, immutable financial facts | Double-entry, ACID, idempotency, concurrency, testing | Go, PostgreSQL, SQLC, Docker Compose |
| 2 — Wallet | Build Wallet capabilities on the ledger | Users need deposits, withdrawals, transfers, balances, and holds | Product boundaries, available vs ledger balance, concurrency | Go HTTP API, PostgreSQL |
| 3 — CQRS | Separate write and read concerns | Read workloads or projections become complex enough to justify separation | CQRS, projections, eventual consistency, rebuilds | DynamoDB wallet-balance projection |
| 4 — Event Driven | Integrate asynchronous consumers | Persisting a change and publishing it independently creates a failure window | Outbox, at-least-once delivery, consumers, DLQ | SNS/SQS or Pub/Sub |
| 5 — Resilience | Contain dependency and consumer failures | Retries and slow/unavailable dependencies threaten availability | Timeout, retry, backoff, jitter, circuit breaker, idempotent consumers | Go libraries and failure tests |
| 6 — Distributed Workflows | Coordinate multi-step payment flows | Holds, authorization and capture span multiple operational steps | Saga, compensation, workflow state | Go orchestrator, mock anti-fraud and mock gateway |
| 7 — Platform | Operate the system consistently | Multiple deployable components need repeatable environments and policies | Kubernetes, Helm, IaC, config, scaling, disruption budgets | Kubernetes, Helm, Terraform, Local AWS emulator |
| 8 — Reliability | Make behavior observable and test failure | Distributed behavior cannot be understood from logs alone | Tracing, metrics, logs, chaos, load and failure testing | OpenTelemetry, load/chaos tools |
| Future — Investments | Explore another product on the ledger | Investment products introduce new instruments and settlement concerns | Brokerage and investment-domain modeling | To be decided from concrete needs |

Phases 1 through 8 describe the implemented path from the ledger core to wallet operations, derived projections, event delivery, resilience, the local checkout Saga, platform manifests and observability. Real external payments, settlement and additional financial products remain outside the current scope.
