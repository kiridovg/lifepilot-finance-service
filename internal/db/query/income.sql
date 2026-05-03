-- name: CreateIncome :one
INSERT INTO incomes (user_id, date, amount, currency, charged_amount, charged_currency,
                     account_id, category_id, description, base_amount, base_currency)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListIncomes :many
SELECT * FROM incomes ORDER BY date DESC;

-- name: ListIncomesByAccount :many
SELECT * FROM incomes
WHERE account_id = sqlc.arg(account_id)
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR date >= sqlc.narg(date_from)::timestamptz)
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR date < sqlc.narg(date_to)::timestamptz)
ORDER BY date DESC;

-- name: ListIncomesByUser :many
SELECT * FROM incomes WHERE user_id = $1 ORDER BY date DESC;

-- name: ListIncomesByDateRange :many
SELECT * FROM incomes
WHERE date >= $1 AND date < $2
ORDER BY date DESC;

-- name: UpdateIncome :one
UPDATE incomes
SET amount           = COALESCE(sqlc.narg(amount), amount),
    currency         = COALESCE(sqlc.narg(currency), currency),
    charged_amount   = COALESCE(sqlc.narg(charged_amount), charged_amount),
    charged_currency = COALESCE(sqlc.narg(charged_currency), charged_currency),
    description      = COALESCE(sqlc.narg(description), description),
    category_id      = COALESCE(sqlc.narg(category_id), category_id),
    date             = COALESCE(sqlc.narg(date), date),
    updated_at       = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteIncome :exec
DELETE FROM incomes WHERE id = $1;
