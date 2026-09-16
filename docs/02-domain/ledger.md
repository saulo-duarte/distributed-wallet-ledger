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

## Money representation

The domain should introduce a small `Money` value object when implementation begins. It should keep a positive integer amount in minor units together with a currency and expose operations that preserve currency and overflow rules. PostgreSQL stores the amount as `BIGINT` and the currency on the journal entry; adapters map between database rows and domain values. This keeps financial code safer than passing a naked `int64` everywhere while avoiding decimal or floating-point arithmetic.
