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
      )::BIGINT AS total_credits
  FROM postings
  WHERE account_id = $1;