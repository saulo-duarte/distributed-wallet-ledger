# Current Scope

## Phase 1 includes

- Ledger accounts.
- Business transactions.
- Journal entries.
- Debit and credit postings.
- Double-entry validation.
- PostgreSQL as the financial source of truth.
- SQLC-generated database access.
- ACID transaction boundaries.
- Domain unit tests.
- Integration tests against real PostgreSQL.
- Reversal through compensating ledger entries.
- Initial, database-backed idempotency.

## Phase 1 does not include

- DynamoDB or a distributed CQRS read model.
- SNS, SQS, Pub/Sub, or distributed event delivery.
- Kubernetes, Helm, or AWS Terraform.
- Sagas or distributed workflows.
- Circuit breakers or cross-service resilience policies.
- Risk, notification, settlement, or other distributed services.
- Full event sourcing.

Future technologies may appear in the roadmap as planned study topics. Their inclusion is not a current implementation decision.
