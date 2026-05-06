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
	m := req.Msg
	q := db.New(h.pool)
	var rows []db.Transfer
	var err error
	if m.AccountId != nil {
		rows, err = q.ListTransfersByAccount(ctx, db.ListTransfersByAccountParams{
			AccountID: uuidFromString(*m.AccountId),
			DateFrom:  nullTimestamptz(m.DateFrom),
			DateTo:    nullTimestamptz(m.DateTo),
		})
	} else {
		rows, err = q.ListTransfers(ctx)
	}
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

		rate := numericToFloat(nullNumericFromPtr(m.Rate))

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
			Rate:                nullNumericFromPtr(m.Rate),
		})
		if err != nil {
			return err
		}

		// Commission1: fee from the source account, recorded as a separate expense.
		if m.Commission != nil && *m.Commission != "" && *m.Commission != "0" && m.FromAccountId != nil {
			fromAcc, err := q.GetAccount(ctx, uuidFromString(*m.FromAccountId))
			if err != nil {
				return err
			}
			comm1RateToBase := 0.0
			if strDeref(m.CommissionCurrency) == BaseCurrency {
				comm1RateToBase = 1.0
			}
			_, err = createExpenseWithBase(ctx, q, db.CreateExpenseParams{
				UserID:      fromAcc.UserID,
				Date:        pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
				Amount:      numericFromString(*m.Commission),
				Currency:    strDeref(m.CommissionCurrency),
				AccountID:   uuidFromString(*m.FromAccountId),
				CategoryID:  systemCategoryUUID("bank-fees"),
				TransferID:  r.ID,
				Description: commissionDesc(m.Description),
			}, comm1RateToBase)
			if err != nil {
				return err
			}
		}

		// Create FIFO lot for the destination account when receiving foreign currency.
		// rate is stored as: base_currency (EUR) per 1 unit of to_currency.
		// Lot is created BEFORE commission2 so that commission2 can immediately consume from it.
		if m.ToAccountId != nil && m.ToCurrency != BaseCurrency && rate > 0 {
			toAmount := numericToFloat(numericFromString(m.ToAmount))
			_, err = q.CreateLot(ctx, db.CreateLotParams{
				AccountID:      uuidFromString(*m.ToAccountId),
				TransferID:     r.ID,
				OriginalAmount: numericFromFloat64(toAmount),
				RateToBase:     numericFromFloat64(rate),
				Remaining:      numericFromFloat64(toAmount),
				BaseCurrency:   BaseCurrency,
				Date:           pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
			})
			if err != nil {
				return err
			}
		}

		// Commission2: fee taken from the destination account (e.g. ATM KZT fee).
		// Uses computeBaseAmount so it consumes from the FIFO lot created above.
		if m.Commission2 != nil && *m.Commission2 != "" && *m.Commission2 != "0" && m.ToAccountId != nil {
			toAcc, err := q.GetAccount(ctx, uuidFromString(*m.ToAccountId))
			if err != nil {
				return err
			}
			baseAmt, baseCur := computeBaseAmount(ctx, q, toAcc, nil, nil,
				*m.Commission2, strDeref(m.Commission2Currency))
			_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
				UserID:       toAcc.UserID,
				Date:         pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
				Amount:       numericFromString(*m.Commission2),
				Currency:     strDeref(m.Commission2Currency),
				AccountID:    uuidFromString(*m.ToAccountId),
				CategoryID:   systemCategoryUUID("bank-fees"),
				TransferID:   r.ID,
				Description:  commissionDesc(m.Description),
				BaseAmount:   baseAmt,
				BaseCurrency: baseCur,
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
		Rate:                nullNumericToPtr(r.Rate),
		Date:                timestamppb.New(r.Date.Time),
		CreatedAt:           timestamppb.New(r.CreatedAt.Time),
	}
}