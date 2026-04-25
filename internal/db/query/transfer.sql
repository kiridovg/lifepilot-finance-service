-- name: ListTransfers :many
SELECT id, from_amount, from_currency, to_amount, to_currency,
       commission, commission_currency, from_payment_method, to_payment_method,
       note, date, created_at
FROM transfers
ORDER BY date DESC;

-- name: CreateTransfer :one
INSERT INTO transfers (from_amount, from_currency, to_amount, to_currency,
                       commission, commission_currency, from_payment_method,
                       to_payment_method, note, date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, from_amount, from_currency, to_amount, to_currency,
          commission, commission_currency, from_payment_method, to_payment_method,
          note, date, created_at;

-- name: DeleteTransfer :exec
DELETE FROM transfers WHERE id = $1;
