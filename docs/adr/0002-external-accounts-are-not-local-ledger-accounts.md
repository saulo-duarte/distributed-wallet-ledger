# ADR-0002: External Accounts Are Not Local Ledger Accounts

## Status

Accepted

## Context

The Ledger records financial facts controlled by our institution. A transfer may have a destination at another bank or financial institution, but our system does not own that institution's accounts, balances or internal state. The current checkout flow uses a recipient value and mock gateway only; it does not model a real external account or payment rail.

Treating every external destination as a local `Account` would incorrectly suggest that our Ledger can validate or mutate an account it does not control. It would also mix accounting facts with payment-network integration data.

## Decision

Keep `AccountID` in a `Posting` limited to accounts owned by the local Ledger. Represent an external destination through a future external account reference and payment instruction. Use local clearing or settlement accounts to represent funds in transit, obligations, or receivables related to the external operation.

The external payment instruction has its own operational lifecycle, such as `pending`, `submitted`, `settled`, `failed`, or `rejected`. Its status is separate from the immutable financial facts in the Ledger.

If an external failure requires a financial correction after a local posting, create a new balanced compensating entry. Do not update or delete the original postings. If funds must not move before external authorization, use the implemented wallet hold and authorization/capture semantics while keeping the external payment instruction separate.

## Alternatives Considered

### Store external accounts as local accounts

Rejected because the local institution cannot treat another institution's account as part of its own authoritative accounting state.

### Put external references directly on postings

Rejected for the initial design because a posting is an accounting line against a local Ledger account. Mixing network identifiers into it would couple the accounting core to payment-rail protocols.

### Keep payment instructions separate from the Ledger

Accepted. This preserves the boundary between local financial facts and external operational integration while allowing both to be correlated by a transaction or payment identifier.

## Consequences

### Positive

- The Ledger has a clear ownership boundary.
- External account data cannot be mistaken for local balance state.
- The mock checkout can be replaced by payment-provider integration without changing accounting rules.
- Asynchronous responses can update operational state without mutating posted financial facts.
- Reconciliation and compensating entries have explicit places in the future model.

### Negative

- External transfers require more than one concept: local postings, external references, payment instructions, and later reconciliation.
- A local posting may exist while the external operation is still pending.
- The project will need explicit failure and compensation rules when external payments are introduced.
