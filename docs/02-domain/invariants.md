# Domain Invariants

The following rules are normative for Phase 1. `MUST` is mandatory, `MUST NOT` is prohibited, and `SHOULD` is a strong default that requires an explicit reason to deviate from.

## Balanced entry

Every posted journal entry MUST satisfy:

```text
Σ(debit amounts) = Σ(credit amounts)
```

An unbalanced entry MUST be rejected before it becomes visible as posted financial state.

## Positive amount

Every posting amount MUST be greater than zero and represented in integer minor units. Direction MUST be stored separately as debit or credit. Financial calculations MUST NOT use `float64`.

## Atomic posting

All postings belonging to a journal entry MUST be persisted in one database transaction. The operation MUST commit all of them or none of them.

## Immutable ledger

Posted transactions, journal entries, and postings MUST NOT be edited or deleted as a correction mechanism. Their original facts SHOULD remain queryable indefinitely within the retention policy of the experiment.

## Idempotency

Retrying a command with the same idempotency scope and key MUST NOT create a second financial transaction. Reusing a key with a materially different request MUST be rejected as a conflict.

## Currency consistency

All postings in one journal entry MUST use the entry currency, and every referenced account MUST support that currency. Phase 1 starts with BRL and does not implement foreign-exchange conversion.

## Reversal instead of mutation

Undoing a posted transaction MUST create a new balanced compensating entry. A reversal MUST NOT mutate or delete the original financial facts, and a transaction MUST NOT be reversed twice.
