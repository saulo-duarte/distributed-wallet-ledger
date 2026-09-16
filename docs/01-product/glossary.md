# Glossary

## Ledger

The append-oriented accounting record that is the financial source of truth.

## Account

A named accounting bucket inside the ledger. Examples include a wallet, clearing account, fees account, reserve, and settlement account.

## Transaction

A business operation that requests or explains a financial change, such as a transfer. It is not itself the debit or credit line.

## Journal Entry

The balanced accounting event created for a transaction. It groups the postings that must be persisted atomically.

## Posting

One debit or credit applied to an account as part of a journal entry.

## Debit

One direction of a posting. Its business meaning depends on the account type; it is not represented by a negative amount.

## Credit

The other direction of a posting. Its business meaning depends on the account type; it is not represented by a negative amount.

## Balance

A value derived from an account's postings. A balance is not a mutable replacement for the ledger history.

## Wallet

The first planned product built on top of the ledger. Wallet behavior is product logic; the ledger records the resulting accounting facts.

## Idempotency Key

A client- or application-provided key used to ensure that retrying the same operation does not create duplicate financial postings.

## Reversal

A new compensating transaction that offsets an earlier posted transaction. Financial history is not deleted or edited.

## Source of Truth

The authoritative system from which financial state can be reconstructed and against which other representations are compared.

Future terms such as authorization, hold, capture, settlement, clearing, reconciliation, and chargeback will be added when those concepts enter the implemented scope.
