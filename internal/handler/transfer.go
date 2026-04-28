package handler

import (
	"context"

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
		transfers = append(transfers, transferToProto(r))
	}
	return connect.NewResponse(&financev1.ListTransfersResponse{Transfers: transfers}), nil
}

func (h *TransferHandler) CreateTransfer(ctx context.Context, req *connect.Request[financev1.CreateTransferRequest]) (*connect.Response[financev1.CreateTransferResponse], error) {
	m := req.Msg

	var transfer *financev1.Transfer
	err := pgx.BeginFunc(ctx, h.pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		r, err := q.CreateTransfer(ctx, db.CreateTransferParams{
			Date:                pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
			FromAccountID:       nullUUIDFromPtr(m.FromAccountId),
			FromAmount:          nullNumericFromPtr(m.FromAmount),
			FromCurrency:        nullTextFromPtr(m.FromCurrency),
			ToAccountID:         nullUUIDFromPtr(m.ToAccountId),
			ToAmount:            numericFromString(m.ToAmount),
			ToCurrency:          m.ToCurrency,
			Commission:          nullNumericFromPtr(m.Commission),
			CommissionCurrency:  nullTextFromPtr(m.CommissionCurrency),
			Commission2:         nullNumericFromPtr(m.Commission2),
			Commission2Currency: nullTextFromPtr(m.Commission2Currency),
			Description:         nullTextFromPtr(m.Description),
			LinkedTransferID:    nullUUIDFromPtr(m.LinkedTransferId),
		})
		if err != nil {
			return err
		}

		// Commission expense linked to this transfer (excluded from balance, visible in bank-fees stats)
		if m.Commission != nil && *m.Commission != "" && *m.Commission != "0" && m.FromAccountId != nil {
			fromAcc, err := q.GetAccount(ctx, uuidFromString(*m.FromAccountId))
			if err != nil {
				return err
			}
			_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
				UserID:     fromAcc.UserID,
				Date:       pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
				Amount:     numericFromString(*m.Commission),
				Currency:   strDeref(m.CommissionCurrency),
				AccountID:  uuidFromString(*m.FromAccountId),
				CategoryID: systemCategoryUUID("bank-fees"),
				TransferID: r.ID,
			})
			if err != nil {
				return err
			}
		}

		// Commission2 expense (e.g. ATM KZT fee) — linked to transfer, from to_account
		if m.Commission2 != nil && *m.Commission2 != "" && *m.Commission2 != "0" && m.ToAccountId != nil {
			toAcc, err := q.GetAccount(ctx, uuidFromString(*m.ToAccountId))
			if err != nil {
				return err
			}
			_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
				UserID:     toAcc.UserID,
				Date:       pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
				Amount:     numericFromString(*m.Commission2),
				Currency:   strDeref(m.Commission2Currency),
				AccountID:  uuidFromString(*m.ToAccountId),
				CategoryID: systemCategoryUUID("bank-fees"),
				TransferID: r.ID,
			})
			if err != nil {
				return err
			}
		}

		transfer = transferToProto(r)
		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateTransferResponse{Transfer: transfer}), nil
}

func (h *TransferHandler) DeleteTransfer(ctx context.Context, req *connect.Request[financev1.DeleteTransferRequest]) (*connect.Response[financev1.DeleteTransferResponse], error) {
	if err := db.New(h.pool).DeleteTransfer(ctx, uuidFromString(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteTransferResponse{}), nil
}

func transferToProto(r db.Transfer) *financev1.Transfer {
	return &financev1.Transfer{
		Id:                  uuidToString(r.ID),
		FromAccountId:       nullUUIDToPtr(r.FromAccountID),
		FromAmount:          nullNumericToPtr(r.FromAmount),
		FromCurrency:        nullTextToPtr(r.FromCurrency),
		ToAccountId:         nullUUIDToPtr(r.ToAccountID),
		ToAmount:            numericToString(r.ToAmount),
		ToCurrency:          r.ToCurrency,
		Commission:          nullNumericToPtr(r.Commission),
		CommissionCurrency:  nullTextToPtr(r.CommissionCurrency),
		Commission2:         nullNumericToPtr(r.Commission2),
		Commission2Currency: nullTextToPtr(r.Commission2Currency),
		Description:         nullTextToPtr(r.Description),
		LinkedTransferId:    nullUUIDToPtr(r.LinkedTransferID),
		Date:                timestamppb.New(r.Date.Time),
		CreatedAt:           timestamppb.New(r.CreatedAt.Time),
	}
}
