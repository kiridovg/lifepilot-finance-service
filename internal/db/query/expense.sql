-- name: CreateExpense :one
INSERT INTO expenses (user_id, date, amount, currency, charged_amount, charged_currency,
                      account_id, category_id, description, transfer_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListExpensesByAccount :many
SELECT * FROM expenses
WHERE account_id = $1
ORDER BY date DESC;

-- name: ListExpenses :many
SELECT * FROM expenses ORDER BY date DESC;

-- name: ListExpensesByDateRange :many
SELECT * FROM expenses
WHERE date >= $1 AND date < $2
ORDER BY date DESC;

-- name: ListExpensesByUser :many
SELECT * FROM expenses WHERE user_id = $1 ORDER BY date DESC;

-- name: UpdateExpense :one
UPDATE expenses
SET account_id       = COALESCE(sqlc.narg(account_id), account_id),
    user_id          = COALESCE(sqlc.narg(user_id), user_id),
    amount           = COALESCE(sqlc.narg(amount), amount),
    currency         = COALESCE(sqlc.narg(currency), currency),
    charged_amount   = COALESCE(sqlc.narg(charged_amount), charged_amount),
    charged_currency = COALESCE(sqlc.narg(charged_currency), charged_currency),
    description      = COALESCE(sqlc.narg(description), description),
    category_id      = COALESCE(sqlc.narg(category_id), category_id),
    date             = COALESCE(sqlc.narg(date), date),
    updated_at       = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteExpense :exec
DELETE FROM expenses WHERE id = $1;
