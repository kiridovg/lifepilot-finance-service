package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type IncomeHandler struct {
	q *db.Queries
}

func NewIncomeHandler(q *db.Queries) *IncomeHandler {
	return &IncomeHandler{q: q}
}

func (h *IncomeHandler) ListIncomes(ctx context.Context, req *connect.Request[financev1.ListIncomesRequest]) (*connect.Response[financev1.ListIncomesResponse], error) {
	var rows []db.Income
	var err error

	m := req.Msg
	if m.AccountId != nil {
		rows, err = h.q.ListIncomesByAccount(ctx, db.ListIncomesByAccountParams{
			AccountID: uuidFromString(*m.AccountId),
			DateFrom:  nullTimestamptz(m.DateFrom),
			DateTo:    nullTimestamptz(m.DateTo),
		})
	} else if m.UserId != nil {
		rows, err = h.q.ListIncomesByUser(ctx, uuidFromString(*m.UserId))
	} else if m.DateFrom != nil && m.DateTo != nil {
		rows, err = h.q.ListIncomesByDateRange(ctx, db.ListIncomesByDateRangeParams{
			Date:   pgtype.Timestamptz{Time: m.DateFrom.AsTime(), Valid: true},
			Date_2: pgtype.Timestamptz{Time: m.DateTo.AsTime(), Valid: true},
		})
	} else {
		rows, err = h.q.ListIncomes(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	incomes := make([]*financev1.Income, 0, len(rows))
	for _, r := range rows {
		incomes = append(incomes, incomeToProto(r))
	}
	return connect.NewResponse(&financev1.ListIncomesResponse{Incomes: incomes}), nil
}

func (h *IncomeHandler) CreateIncome(ctx context.Context, req *connect.Request[financev1.CreateIncomeRequest]) (*connect.Response[financev1.CreateIncomeResponse], error) {
	m := req.Msg
	r, err := h.q.CreateIncome(ctx, db.CreateIncomeParams{
		UserID:          uuidFromString(m.UserId),
		AccountID:       uuidFromString(m.AccountId),
		Amount:          numericFromString(m.Amount),
		Currency:        m.Currency,
		ChargedAmount:   nullNumericFromPtr(m.ChargedAmount),
		ChargedCurrency: nullTextFromPtr(m.ChargedCurrency),
		CategoryID:      nullUUIDFromPtr(m.CategoryId),
		Description:     nullTextFromPtr(m.Description),
		Date:            pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateIncomeResponse{Income: incomeToProto(r)}), nil
}

func (h *IncomeHandler) UpdateIncome(ctx context.Context, req *connect.Request[financev1.UpdateIncomeRequest]) (*connect.Response[financev1.UpdateIncomeResponse], error) {
	m := req.Msg
	dateTs := pgtype.Timestamptz{}
	if m.Date != nil {
		dateTs = pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true}
	}
	r, err := h.q.UpdateIncome(ctx, db.UpdateIncomeParams{
		ID:              uuidFromString(m.Id),
		Amount:          nullNumericFromPtr(m.Amount),
		Currency:        nullTextFromPtr(m.Currency),
		ChargedAmount:   nullNumericFromPtr(m.ChargedAmount),
		ChargedCurrency: nullTextFromPtr(m.ChargedCurrency),
		CategoryID:      nullUUIDFromPtr(m.CategoryId),
		Description:     nullTextFromPtr(m.Description),
		Date:            dateTs,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.UpdateIncomeResponse{Income: incomeToProto(r)}), nil
}

func (h *IncomeHandler) DeleteIncome(ctx context.Context, req *connect.Request[financev1.DeleteIncomeRequest]) (*connect.Response[financev1.DeleteIncomeResponse], error) {
	if err := h.q.DeleteIncome(ctx, uuidFromString(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteIncomeResponse{}), nil
}

func incomeToProto(r db.Income) *financev1.Income {
	return &financev1.Income{
		Id:                 uuidToString(r.ID),
		UserId:             uuidToString(r.UserID),
		AccountId:          uuidToString(r.AccountID),
		Amount:             numericToString(r.Amount),
		Currency:           r.Currency,
		ChargedAmount:      nullNumericToPtr(r.ChargedAmount),
		ChargedCurrency:    nullTextToPtr(r.ChargedCurrency),
		CategoryId:         nullUUIDToPtr(r.CategoryID),
		Description:        nullTextToPtr(r.Description),
		IsRefund:           r.IsRefund,
		RefundedExpenseId:  nullUUIDToPtr(r.RefundedExpenseID),
		Date:               timestamppb.New(r.Date.Time),
		CreatedAt:          timestamppb.New(r.CreatedAt.Time),
	}
}
