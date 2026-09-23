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

The implemented product boundary built on top of the ledger. Wallet behavior includes deposits, withdrawals, transfers, balances and holds; the ledger records the resulting accounting facts.

## Idempotency Key

A client- or application-provided key used to ensure that retrying the same operation does not create duplicate financial postings.

## Reversal

A new compensating transaction that offsets an earlier posted transaction. Financial history is not deleted or edited.

## Source of Truth

The authoritative system from which financial state can be reconstructed and against which other representations are compared.

## External Account Reference

Information used to identify an account or beneficiary at another financial institution. It is not a local Ledger `Account` and does not give our system authority over the external institution's balance.

## Clearing Account

A local Ledger account used to represent funds in transit, receivables, payables, or obligations related to an external operation. It helps keep local accounting balanced while an external operation is pending.

## Payment Instruction

An operational request sent to a payment rail, provider, or settlement operator. It contains the external destination and the amount to process, but it is not itself a Journal Entry.

## External Operational Status

The lifecycle of an external payment instruction, such as `pending`, `submitted`, `settled`, `failed`, or `rejected`. This status is separate from the immutable financial facts recorded in the Ledger.

Authorization, hold and capture are implemented wallet concepts. Settlement, clearing, reconciliation and chargeback remain external-payment concepts outside the current implementation.
