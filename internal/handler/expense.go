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
	rows, err := h.q.ListExpenses(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	expenses := make([]*financev1.Expense, 0, len(rows))
	for _, r := range rows {
		var transferID *int32
		if r.TransferID.Valid {
			v := r.TransferID.Int32
			transferID = &v
		}
		expenses = append(expenses, &financev1.Expense{
			Id:              r.ID,
			Description:     r.Description,
			Amount:          numericToString(r.Amount),
			Currency:        r.Currency,
			ChargedAmount:   nullNumericToPtr(r.ChargedAmount),
			ChargedCurrency: nullTextToPtr(r.ChargedCurrency),
			PaymentMethod:   r.PaymentMethod,
			Category:        nullTextToPtr(r.Category),
			TransferId:      transferID,
			Date:            timestamppb.New(r.Date.Time),
			CreatedAt:       timestamppb.New(r.CreatedAt.Time),
		})
	}
	return connect.NewResponse(&financev1.ListExpensesResponse{Expenses: expenses}), nil
}

func (h *ExpenseHandler) CreateExpense(ctx context.Context, req *connect.Request[financev1.CreateExpenseRequest]) (*connect.Response[financev1.CreateExpenseResponse], error) {
	m := req.Msg
	r, err := h.q.CreateExpense(ctx, db.CreateExpenseParams{
		Description:     m.Description,
		Amount:          numericFromString(m.Amount),
		Currency:        m.Currency,
		ChargedAmount:   nullNumericFromPtr(m.ChargedAmount),
		ChargedCurrency: nullText(m.ChargedCurrency),
		PaymentMethod:   m.PaymentMethod,
		Category:        nullText(m.Category),
		Date:            pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateExpenseResponse{
		Expense: &financev1.Expense{
			Id:              r.ID,
			Description:     r.Description,
			Amount:          numericToString(r.Amount),
			Currency:        r.Currency,
			ChargedAmount:   nullNumericToPtr(r.ChargedAmount),
			ChargedCurrency: nullTextToPtr(r.ChargedCurrency),
			PaymentMethod:   r.PaymentMethod,
			Category:        nullTextToPtr(r.Category),
			Date:            timestamppb.New(r.Date.Time),
			CreatedAt:       timestamppb.New(r.CreatedAt.Time),
		},
	}), nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *connect.Request[financev1.UpdateExpenseRequest]) (*connect.Response[financev1.UpdateExpenseResponse], error) {
	m := req.Msg
	dateTs := pgtype.Timestamptz{}
	if m.Date != nil {
		dateTs = pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true}
	}
	r, err := h.q.UpdateExpense(ctx, db.UpdateExpenseParams{
		ID:              m.Id,
		Description:     nullText(m.Description),
		Amount:          nullNumericFromPtr(m.Amount),
		Currency:        nullText(m.Currency),
		ChargedAmount:   nullNumericFromPtr(m.ChargedAmount),
		ChargedCurrency: nullText(m.ChargedCurrency),
		PaymentMethod:   nullText(m.PaymentMethod),
		Category:        nullText(m.Category),
		Date:            dateTs,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.UpdateExpenseResponse{
		Expense: &financev1.Expense{
			Id:              r.ID,
			Description:     r.Description,
			Amount:          numericToString(r.Amount),
			Currency:        r.Currency,
			ChargedAmount:   nullNumericToPtr(r.ChargedAmount),
			ChargedCurrency: nullTextToPtr(r.ChargedCurrency),
			PaymentMethod:   r.PaymentMethod,
			Category:        nullTextToPtr(r.Category),
			Date:            timestamppb.New(r.Date.Time),
			CreatedAt:       timestamppb.New(r.CreatedAt.Time),
		},
	}), nil
}

func (h *ExpenseHandler) DeleteExpense(ctx context.Context, req *connect.Request[financev1.DeleteExpenseRequest]) (*connect.Response[financev1.DeleteExpenseResponse], error) {
	if err := h.q.DeleteExpense(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteExpenseResponse{}), nil
}
