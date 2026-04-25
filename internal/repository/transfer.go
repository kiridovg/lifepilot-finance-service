package repository

import (
	"context"
	"time"
)

type Transfer struct {
	ID                 int32
	FromAmount         string
	FromCurrency       string
	ToAmount           string
	ToCurrency         string
	Commission         string
	CommissionCurrency *string
	FromPaymentMethod  *string
	ToPaymentMethod    *string
	Note               *string
	Date               time.Time
	CreatedAt          time.Time
}

type CreateTransferParams struct {
	FromAmount         string
	FromCurrency       string
	ToAmount           string
	ToCurrency         string
	Commission         string
	CommissionCurrency *string
	FromPaymentMethod  *string
	ToPaymentMethod    *string
	Note               *string
	Date               time.Time
}

func (r *Repository) ListTransfers(ctx context.Context) ([]Transfer, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, from_amount, from_currency, to_amount, to_currency,
		       commission, commission_currency, from_payment_method, to_payment_method,
		       note, date, created_at
		FROM transfers
		ORDER BY date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []Transfer
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(
			&t.ID, &t.FromAmount, &t.FromCurrency, &t.ToAmount, &t.ToCurrency,
			&t.Commission, &t.CommissionCurrency, &t.FromPaymentMethod, &t.ToPaymentMethod,
			&t.Note, &t.Date, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}

func (r *Repository) CreateTransfer(ctx context.Context, p CreateTransferParams) (Transfer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Transfer{}, err
	}
	defer tx.Rollback(ctx)

	var t Transfer
	err = tx.QueryRow(ctx, `
		INSERT INTO transfers (from_amount, from_currency, to_amount, to_currency,
		                       commission, commission_currency, from_payment_method,
		                       to_payment_method, note, date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, from_amount, from_currency, to_amount, to_currency,
		          commission, commission_currency, from_payment_method, to_payment_method,
		          note, date, created_at
	`, p.FromAmount, p.FromCurrency, p.ToAmount, p.ToCurrency,
		p.Commission, p.CommissionCurrency, p.FromPaymentMethod,
		p.ToPaymentMethod, p.Note, p.Date,
	).Scan(
		&t.ID, &t.FromAmount, &t.FromCurrency, &t.ToAmount, &t.ToCurrency,
		&t.Commission, &t.CommissionCurrency, &t.FromPaymentMethod, &t.ToPaymentMethod,
		&t.Note, &t.Date, &t.CreatedAt,
	)
	if err != nil {
		return Transfer{}, err
	}

	// Commission is a real expense — create it linked to this transfer
	if p.Commission != "" && p.Commission != "0" && p.CommissionCurrency != nil {
		label := "Transfer fee"
		if p.Note != nil && *p.Note != "" {
			label = "Transfer fee (" + *p.Note + ")"
		}
		pm := "cash"
		if p.FromPaymentMethod != nil {
			pm = *p.FromPaymentMethod
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO expenses (description, amount, currency, payment_method, category, transfer_id, date)
			VALUES ($1, $2, $3, $4, 'bank-fees', $5, $6)
		`, label, p.Commission, *p.CommissionCurrency, pm, t.ID, p.Date)
		if err != nil {
			return Transfer{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Transfer{}, err
	}
	return t, nil
}

func (r *Repository) DeleteTransfer(ctx context.Context, id int32) error {
	// Linked commission expense is deleted via ON DELETE CASCADE
	_, err := r.db.Exec(ctx, `DELETE FROM transfers WHERE id = $1`, id)
	return err
}
