-- name: ListExpenses :many
SELECT id, description, amount, currency, charged_amount, charged_currency,
       payment_method, category, transfer_id, date, created_at
FROM expenses
ORDER BY date DESC;

-- name: CreateExpense :one
INSERT INTO expenses (description, amount, currency, charged_amount, charged_currency,
                      payment_method, category, transfer_id, date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, description, amount, currency, charged_amount, charged_currency,
          payment_method, category, transfer_id, date, created_at;

-- name: UpdateExpense :one
UPDATE expenses
SET description      = COALESCE(sqlc.narg(description), description),
    amount           = COALESCE(sqlc.narg(amount), amount),
    currency         = COALESCE(sqlc.narg(currency), currency),
    charged_amount   = COALESCE(sqlc.narg(charged_amount), charged_amount),
    charged_currency = COALESCE(sqlc.narg(charged_currency), charged_currency),
    payment_method   = COALESCE(sqlc.narg(payment_method), payment_method),
    category         = COALESCE(sqlc.narg(category), category),
    date             = COALESCE(sqlc.narg(date), date)
WHERE id = sqlc.arg(id)
RETURNING id, description, amount, currency, charged_amount, charged_currency,
          payment_method, category, transfer_id, date, created_at;

-- name: DeleteExpense :exec
DELETE FROM expenses WHERE id = $1;
