# Accounting Rules

## Double-entry bookkeeping

Every journal entry has at least one debit and one credit, and the totals must balance.

Example: transfer of R$100:

| Account | Direction | Amount |
| --- | --- | ---: |
| Account A | Debit | R$100.00 |
| Account B | Credit | R$100.00 |

Total debit: R$100.00  
Total credit: R$100.00  
Balanced: `true`

The amount is stored as `10000` BRL minor units. A debit is not a negative number; direction and amount are separate values.

## Account meaning

Debit and credit meaning depends on the account's accounting role. The initial domain keeps that interpretation explicit and avoids hiding it behind a generic signed balance. Future technical accounts may represent clearing, fees, reserve, or settlement.

## Reversal

A reversal creates a new transaction and journal entry with the original postings' directions exchanged, preserving the original entry and its audit trail. A transaction must not be deleted to undo its effect.

## Currency

An entry uses one currency in Phase 1. The model uses a three-letter currency code and starts with BRL. Cross-currency transfers are outside the current scope and must not be smuggled in through conversion fields without an explicit domain decision.
