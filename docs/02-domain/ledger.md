# Ledger

## What it is

The Ledger is an append-oriented record of financial facts. It contains accounts, posted journal entries, and the debit/credit postings applied to those accounts.

## Why it is the source of truth

Balances and product views can be derived from postings. The ledger is therefore the authoritative record for what was posted, when it was posted, and which accounts participated. A balance field or future read model may be a projection, but it must not silently replace the accounting history.

## Why it is immutable

Changing a posted posting would destroy the audit trail and make historical results depend on mutable data. If a posting is wrong, the correction is another balanced journal entry that reverses or compensates for the original.

## Ledger versus Wallet

The ledger is a reusable accounting capability. A wallet is a product that defines user-facing behavior such as deposit, withdrawal, transfer, available balance, and holds. The wallet should request ledger changes; it should not maintain a competing financial truth.

## Ledger versus Event Sourcing

An immutable ledger is not automatically event sourcing. The ledger is modeled around accounting facts and double-entry invariants. Event sourcing is a broader persistence and reconstruction strategy in which domain events are the primary state history. Phase 1 does not adopt full event sourcing.

## How postings work

For a R$100 transfer, the application creates a journal entry with positive amounts expressed as 10,000 minor units. The posting direction identifies debit or credit. The entry is accepted only when the total debits equal the total credits and all participating accounts/currencies are valid.

## Internal accounts and external payment destinations

The Ledger is the source of truth for financial facts controlled by our institution. An `Account` in this domain represents an account inside our Ledger, such as a customer wallet, clearing account, fees account, reserve, or settlement account.

An account held at another institution MUST NOT be modeled as a local `Account` merely because it is the destination of a payment. We do not own the external institution's balance or internal state. A future payment boundary may store an external account reference and create a payment instruction, but the local accounting postings still use local clearing or settlement accounts.

For example, an outbound external payment may first be represented locally as:

```text
Debit  CUSTOMER_WALLET    BRL 100.00
Credit EXTERNAL_CLEARING  BRL 100.00
```

The external destination is associated with a separate payment instruction. A payment rail, provider, or authorized settlement operator may later accept, reject, or settle that instruction. This integration result is operational state; it is not a replacement for, or mutation of, the original journal entry.

If the external operation fails after the local posting, the correction is another balanced compensating entry. If the business must wait for authorization before moving funds, a future wallet or workflow model may introduce holds and capture instead of posting the final movement immediately.

This external payment flow is planned for later phases. Phase 1 models only local Ledger accounts and does not implement providers, queues, settlement, or reconciliation.

## Money representation

The domain should introduce a small `Money` value object when implementation begins. It should keep a positive integer amount in minor units together with a currency and expose operations that preserve currency and overflow rules. PostgreSQL stores the amount as `BIGINT` and the currency on the journal entry; adapters map between database rows and domain values. This keeps financial code safer than passing a naked `int64` everywhere while avoiding decimal or floating-point arithmetic.
