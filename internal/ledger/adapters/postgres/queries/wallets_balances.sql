-- name: GetLedgerBalance :one
  SELECT
      COALESCE(
          SUM(
              CASE
                  WHEN direction = 'debit'
                  THEN amount_minor_units
                  ELSE 0
              END
          ),
          0
      )::BIGINT AS total_debits,

      COALESCE(
          SUM(
              CASE
                  WHEN direction = 'credit'
                  THEN amount_minor_units
                  ELSE 0
              END
          ),
          0
      )::BIGINT AS total_credits,

      COALESCE(
          (
              SELECT SUM(wh.amount_minor_units)
              FROM wallet_holds wh
              JOIN wallets w ON w.id = wh.wallet_id
              WHERE w.ledger_account_id = $1
                AND wh.status = 'authorized'
                AND wh.expires_at > NOW()
          ),
          0
      )::BIGINT AS active_holds
  FROM postings
  WHERE account_id = $1;
