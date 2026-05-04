-- name: CreateLot :one
INSERT INTO account_lots (account_id, transfer_id, original_amount, rate_to_base, remaining, base_currency, date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListAvailableLots :many
SELECT * FROM account_lots
WHERE account_id = $1 AND remaining > 0
ORDER BY date ASC, created_at ASC
FOR UPDATE;

-- name: UpdateLotRemaining :one
UPDATE account_lots
SET remaining = $2
WHERE id = $1
RETURNING *;