package handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type TransferHandler struct {
	pool *pgxpool.Pool
}

func NewTransferHandler(pool *pgxpool.Pool) *TransferHandler {
	return &TransferHandler{pool: pool}
}

func (h *TransferHandler) ListTransfers(ctx context.Context, req *connect.Request[financev1.ListTransfersRequest]) (*connect.Response[financev1.ListTransfersResponse], error) {
	rows, err := db.New(h.pool).ListTransfers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	transfers := make([]*financev1.Transfer, 0, len(rows))
	for _, r := range rows {
		transfers = append(transfers, transferRowToProto(r))
	}
	return connect.NewResponse(&financev1.ListTransfersResponse{Transfers: transfers}), nil
}

func (h *TransferHandler) CreateTransfer(ctx context.Context, req *connect.Request[financev1.CreateTransferRequest]) (*connect.Response[financev1.CreateTransferResponse], error) {
	m := req.Msg

	var transfer *financev1.Transfer
	err := pgx.BeginFunc(ctx, h.pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		r, err := q.CreateTransfer(ctx, db.CreateTransferParams{
			FromAmount:         numericFromString(m.FromAmount),
			FromCurrency:       m.FromCurrency,
			ToAmount:           numericFromString(m.ToAmount),
			ToCurrency:         m.ToCurrency,
			Commission:         numericFromString(m.Commission),
			CommissionCurrency: nullText(m.CommissionCurrency),
			FromPaymentMethod:  nullText(m.FromPaymentMethod),
			ToPaymentMethod:    nullText(m.ToPaymentMethod),
			Note:               nullText(m.Note),
			Date:               pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
		})
		if err != nil {
			return err
		}

		// Commission is a real expense — create it linked to this transfer
		if m.Commission != "" && m.Commission != "0" && m.CommissionCurrency != nil {
			label := "Transfer fee"
			if m.Note != nil && *m.Note != "" {
				label = fmt.Sprintf("Transfer fee (%s)", *m.Note)
			}
			pm := "cash"
			if m.FromPaymentMethod != nil {
				pm = *m.FromPaymentMethod
			}
			_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
				Description:   label,
				Amount:        numericFromString(m.Commission),
				Currency:      *m.CommissionCurrency,
				PaymentMethod: pm,
				Category:      pgtype.Text{String: "bank-fees", Valid: true},
				TransferID:    pgtype.Int4{Int32: r.ID, Valid: true},
				Date:          pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
			})
			if err != nil {
				return err
			}
		}

		transfer = transferRowToProto(db.ListTransfersRow(r))
		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateTransferResponse{Transfer: transfer}), nil
}

func (h *TransferHandler) DeleteTransfer(ctx context.Context, req *connect.Request[financev1.DeleteTransferRequest]) (*connect.Response[financev1.DeleteTransferResponse], error) {
	// Linked commission expense is deleted via ON DELETE CASCADE
	if err := db.New(h.pool).DeleteTransfer(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteTransferResponse{}), nil
}

func transferRowToProto(r db.ListTransfersRow) *financev1.Transfer {
	return &financev1.Transfer{
		Id:                 r.ID,
		FromAmount:         numericToString(r.FromAmount),
		FromCurrency:       r.FromCurrency,
		ToAmount:           numericToString(r.ToAmount),
		ToCurrency:         r.ToCurrency,
		Commission:         numericToString(r.Commission),
		CommissionCurrency: nullTextToPtr(r.CommissionCurrency),
		FromPaymentMethod:  nullTextToPtr(r.FromPaymentMethod),
		ToPaymentMethod:    nullTextToPtr(r.ToPaymentMethod),
		Note:               nullTextToPtr(r.Note),
		Date:               timestamppb.New(r.Date.Time),
		CreatedAt:          timestamppb.New(r.CreatedAt.Time),
	}
}
