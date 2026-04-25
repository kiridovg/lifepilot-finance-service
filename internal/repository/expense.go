package repository

import (
	"context"
	"time"
)

type Expense struct {
	ID              int32
	Description     string
	Amount          string
	Currency        string
	ChargedAmount   *string
	ChargedCurrency *string
	PaymentMethod   string
	Category        *string
	TransferID      *int32
	Date            time.Time
	CreatedAt       time.Time
}

type CreateExpenseParams struct {
	Description     string
	Amount          string
	Currency        string
	ChargedAmount   *string
	ChargedCurrency *string
	PaymentMethod   string
	Category        *string
	TransferID      *int32
	Date            time.Time
}

func (r *Repository) ListExpenses(ctx context.Context) ([]Expense, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, description, amount, currency, charged_amount, charged_currency,
		       payment_method, category, transfer_id, date, created_at
		FROM expenses
		ORDER BY date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(
			&e.ID, &e.Description, &e.Amount, &e.Currency,
			&e.ChargedAmount, &e.ChargedCurrency, &e.PaymentMethod,
			&e.Category, &e.TransferID, &e.Date, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

func (r *Repository) CreateExpense(ctx context.Context, p CreateExpenseParams) (Expense, error) {
	var e Expense
	err := r.db.QueryRow(ctx, `
		INSERT INTO expenses (description, amount, currency, charged_amount, charged_currency,
		                      payment_method, category, transfer_id, date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, description, amount, currency, charged_amount, charged_currency,
		          payment_method, category, transfer_id, date, created_at
	`, p.Description, p.Amount, p.Currency, p.ChargedAmount, p.ChargedCurrency,
		p.PaymentMethod, p.Category, p.TransferID, p.Date,
	).Scan(
		&e.ID, &e.Description, &e.Amount, &e.Currency,
		&e.ChargedAmount, &e.ChargedCurrency, &e.PaymentMethod,
		&e.Category, &e.TransferID, &e.Date, &e.CreatedAt,
	)
	return e, err
}

func (r *Repository) DeleteExpense(ctx context.Context, id int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id)
	return err
}
