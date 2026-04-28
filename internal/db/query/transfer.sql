-- name: CreateTransfer :one
INSERT INTO transfers (date, from_account_id, from_amount, from_currency,
                       to_account_id, to_amount, to_currency,
                       commission, commission_currency,
                       commission2, commission2_currency,
                       description, linked_transfer_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: ListTransfers :many
SELECT * FROM transfers ORDER BY date DESC;

-- name: GetTransfer :one
SELECT * FROM transfers WHERE id = $1;

-- name: DeleteTransfer :exec
DELETE FROM transfers WHERE id = $1;
