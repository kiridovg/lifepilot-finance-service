-- name: CreateAccount :one
INSERT INTO accounts (user_id, name, payment_method, currency, initial_balance, initial_date, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = $1;

-- name: ListActiveAccounts :many
SELECT * FROM accounts WHERE is_active = true ORDER BY user_id, created_at;

-- name: ListActiveAccountsByUser :many
SELECT * FROM accounts WHERE is_active = true AND user_id = $1 ORDER BY created_at;

-- name: UpdateAccount :one
UPDATE accounts
SET name            = COALESCE(sqlc.narg(name), name),
    initial_balance = COALESCE(sqlc.narg(initial_balance), initial_balance),
    initial_date    = COALESCE(sqlc.narg(initial_date), initial_date),
    notes           = COALESCE(sqlc.narg(notes), notes),
    updated_at      = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeactivateAccount :exec
UPDATE accounts SET is_active = false, updated_at = NOW() WHERE id = $1;

-- name: GetAccountExpenses :one
-- Sums expenses for an account since initialDate.
-- Excludes transfer fees (transfer_id IS NOT NULL) — already included in fromAmount.
-- Uses chargedAmount when currency differs from account currency.
SELECT COALESCE(SUM(
    CASE
        WHEN charged_currency = $2 AND charged_amount IS NOT NULL THEN charged_amount
        ELSE amount
    END
), 0)::numeric AS total
FROM expenses
WHERE account_id   = $1
  AND date         >= $3
  AND transfer_id  IS NULL
  AND (currency = $2 OR charged_currency = $2);

-- name: GetAccountTransfersOut :one
SELECT COALESCE(SUM(from_amount), 0)::numeric AS total
FROM transfers
WHERE from_account_id = $1
  AND date >= $2;

-- name: GetAccountTransfersIn :one
SELECT COALESCE(SUM(to_amount), 0)::numeric AS total
FROM transfers
WHERE to_account_id = $1
  AND date >= $2;
