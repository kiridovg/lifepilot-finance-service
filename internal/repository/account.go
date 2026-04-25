package repository

import (
	"context"
	"fmt"
	"time"
)

type Account struct {
	ID             int32
	Name           string
	Currency       string
	PaymentMethod  *string
	InitialBalance string
	InitialDate    string
	IsActive       bool
	CreatedAt      time.Time
	// computed
	Balance       float64
	TotalExpenses float64
	TransfersOut  float64
	TransfersIn   float64
}

func (r *Repository) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, currency, payment_method_code, initial_balance, initial_date,
		       is_active, created_at
		FROM accounts
		WHERE is_active = true
		ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Currency, &a.PaymentMethod,
			&a.InitialBalance, &a.InitialDate, &a.IsActive, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := r.computeBalance(ctx, &a); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *Repository) computeBalance(ctx context.Context, a *Account) error {
	if a.PaymentMethod == nil {
		return nil
	}

	// Total expenses for this payment method since initialDate
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			CASE
				WHEN charged_currency = $1 AND charged_amount IS NOT NULL THEN charged_amount::numeric
				WHEN currency = $1 THEN amount::numeric
				ELSE 0
			END
		), 0)
		FROM expenses
		WHERE payment_method = $2 AND date >= $3::timestamptz
	`, a.Currency, *a.PaymentMethod, a.InitialDate).Scan(&a.TotalExpenses)
	if err != nil {
		return err
	}

	// Transfers out (fromAmount in account's currency)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(from_amount::numeric), 0)
		FROM transfers
		WHERE from_payment_method = $1
		  AND from_currency = $2
		  AND date >= $3::timestamptz
	`, *a.PaymentMethod, a.Currency, a.InitialDate).Scan(&a.TransfersOut)
	if err != nil {
		return err
	}

	// Transfers in (toAmount in account's currency)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(to_amount::numeric), 0)
		FROM transfers
		WHERE to_payment_method = $1
		  AND to_currency = $2
		  AND date >= $3::timestamptz
	`, *a.PaymentMethod, a.Currency, a.InitialDate).Scan(&a.TransfersIn)
	if err != nil {
		return err
	}

	initialBalance := 0.0
	if _, err := fmt.Sscanf(a.InitialBalance, "%f", &initialBalance); err != nil {
		return err
	}
	a.Balance = initialBalance - a.TotalExpenses - a.TransfersOut + a.TransfersIn
	return nil
}

func (r *Repository) CreateAccount(ctx context.Context, name, currency string, pm *string, initialBalance float64, initialDate string) (Account, error) {
	var a Account
	err := r.db.QueryRow(ctx, `
		INSERT INTO accounts (name, currency, payment_method_code, initial_balance, initial_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, currency, payment_method_code, initial_balance, initial_date, is_active, created_at
	`, name, currency, pm, fmt.Sprintf("%f", initialBalance), initialDate,
	).Scan(&a.ID, &a.Name, &a.Currency, &a.PaymentMethod, &a.InitialBalance, &a.InitialDate, &a.IsActive, &a.CreatedAt)
	return a, err
}

func (r *Repository) DeleteAccount(ctx context.Context, id int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id)
	return err
}
