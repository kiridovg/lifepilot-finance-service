-- name: ListActiveAccounts :many
SELECT id, name, currency, payment_method_code, initial_balance, initial_date,
       is_active, created_at
FROM accounts
WHERE is_active = true
ORDER BY created_at;

-- name: CreateAccount :one
INSERT INTO accounts (name, currency, payment_method_code, initial_balance, initial_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, currency, payment_method_code, initial_balance, initial_date, is_active, created_at;

-- name: UpdateAccount :one
UPDATE accounts
SET name            = COALESCE(sqlc.narg(name), name),
    initial_balance = COALESCE(sqlc.narg(initial_balance), initial_balance),
    initial_date    = COALESCE(sqlc.narg(initial_date), initial_date)
WHERE id = sqlc.arg(id)
RETURNING id, name, currency, payment_method_code, initial_balance, initial_date, is_active, created_at;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = $1;

-- name: GetAccountExpenses :one
SELECT COALESCE(SUM(
    CASE
        WHEN charged_currency = $1 AND charged_amount IS NOT NULL THEN charged_amount
        WHEN currency = $1 THEN amount
        ELSE 0
    END
), 0)::numeric AS total
FROM expenses
WHERE payment_method = $2 AND date >= $3::timestamptz;

-- name: GetAccountTransfersOut :one
SELECT COALESCE(SUM(from_amount), 0)::numeric AS total
FROM transfers
WHERE from_payment_method = $1
  AND from_currency = $2
  AND date >= $3::timestamptz;

-- name: GetAccountTransfersIn :one
SELECT COALESCE(SUM(to_amount), 0)::numeric AS total
FROM transfers
WHERE to_payment_method = $1
  AND to_currency = $2
  AND date >= $3::timestamptz;
