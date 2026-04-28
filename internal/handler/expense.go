package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type ExpenseHandler struct {
	q *db.Queries
}

func NewExpenseHandler(q *db.Queries) *ExpenseHandler {
	return &ExpenseHandler{q: q}
}

func (h *ExpenseHandler) ListExpenses(ctx context.Context, req *connect.Request[financev1.ListExpensesRequest]) (*connect.Response[financev1.ListExpensesResponse], error) {
	var rows []db.Expense
	var err error

	m := req.Msg
	if m.UserId != nil {
		rows, err = h.q.ListExpensesByUser(ctx, uuidFromString(*m.UserId))
	} else if m.DateFrom != nil && m.DateTo != nil {
		rows, err = h.q.ListExpensesByDateRange(ctx, db.ListExpensesByDateRangeParams{
			Date:   pgtype.Timestamptz{Time: m.DateFrom.AsTime(), Valid: true},
			Date_2: pgtype.Timestamptz{Time: m.DateTo.AsTime(), Valid: true},
		})
	} else {
		rows, err = h.q.ListExpenses(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	expenses := make([]*financev1.Expense, 0, len(rows))
	for _, r := range rows {
		expenses = append(expenses, expenseToProto(r))
	}
	return connect.NewResponse(&financev1.ListExpensesResponse{Expenses: expenses}), nil
}

func (h *ExpenseHandler) CreateExpense(ctx context.Context, req *connect.Request[financev1.CreateExpenseRequest]) (*connect.Response[financev1.CreateExpenseResponse], error) {
	m := req.Msg
	r, err := h.q.CreateExpense(ctx, db.CreateExpenseParams{
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
	return connect.NewResponse(&financev1.CreateExpenseResponse{Expense: expenseToProto(r)}), nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *connect.Request[financev1.UpdateExpenseRequest]) (*connect.Response[financev1.UpdateExpenseResponse], error) {
	m := req.Msg
	dateTs := pgtype.Timestamptz{}
	if m.Date != nil {
		dateTs = pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true}
	}
	r, err := h.q.UpdateExpense(ctx, db.UpdateExpenseParams{
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
	return connect.NewResponse(&financev1.UpdateExpenseResponse{Expense: expenseToProto(r)}), nil
}

func (h *ExpenseHandler) DeleteExpense(ctx context.Context, req *connect.Request[financev1.DeleteExpenseRequest]) (*connect.Response[financev1.DeleteExpenseResponse], error) {
	if err := h.q.DeleteExpense(ctx, uuidFromString(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteExpenseResponse{}), nil
}

func expenseToProto(r db.Expense) *financev1.Expense {
	return &financev1.Expense{
		Id:              uuidToString(r.ID),
		UserId:          uuidToString(r.UserID),
		AccountId:       uuidToString(r.AccountID),
		Amount:          numericToString(r.Amount),
		Currency:        r.Currency,
		ChargedAmount:   nullNumericToPtr(r.ChargedAmount),
		ChargedCurrency: nullTextToPtr(r.ChargedCurrency),
		CategoryId:      nullUUIDToPtr(r.CategoryID),
		Description:     nullTextToPtr(r.Description),
		TransferId:      nullUUIDToPtr(r.TransferID),
		Date:            timestamppb.New(r.Date.Time),
		CreatedAt:       timestamppb.New(r.CreatedAt.Time),
	}
}
