package handler

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/repository"
)

type ExpenseHandler struct {
	repo *repository.Repository
}

func NewExpenseHandler(repo *repository.Repository) *ExpenseHandler {
	return &ExpenseHandler{repo: repo}
}

func (h *ExpenseHandler) ListExpenses(ctx context.Context, req *connect.Request[financev1.ListExpensesRequest]) (*connect.Response[financev1.ListExpensesResponse], error) {
	expenses, err := h.repo.ListExpenses(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var proto []*financev1.Expense
	for _, e := range expenses {
		proto = append(proto, expenseToProto(e))
	}
	return connect.NewResponse(&financev1.ListExpensesResponse{Expenses: proto}), nil
}

func (h *ExpenseHandler) CreateExpense(ctx context.Context, req *connect.Request[financev1.CreateExpenseRequest]) (*connect.Response[financev1.CreateExpenseResponse], error) {
	p := repository.CreateExpenseParams{
		Description:     req.Msg.Description,
		Amount:          req.Msg.Amount,
		Currency:        req.Msg.Currency,
		ChargedAmount:   req.Msg.ChargedAmount,
		ChargedCurrency: req.Msg.ChargedCurrency,
		PaymentMethod:   req.Msg.PaymentMethod,
		Category:        req.Msg.Category,
		Date:            req.Msg.Date.AsTime(),
	}
	e, err := h.repo.CreateExpense(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateExpenseResponse{Expense: expenseToProto(e)}), nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *connect.Request[financev1.UpdateExpenseRequest]) (*connect.Response[financev1.UpdateExpenseResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (h *ExpenseHandler) DeleteExpense(ctx context.Context, req *connect.Request[financev1.DeleteExpenseRequest]) (*connect.Response[financev1.DeleteExpenseResponse], error) {
	if err := h.repo.DeleteExpense(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteExpenseResponse{}), nil
}

func expenseToProto(e repository.Expense) *financev1.Expense {
	return &financev1.Expense{
		Id:              e.ID,
		Description:     e.Description,
		Amount:          e.Amount,
		Currency:        e.Currency,
		ChargedAmount:   e.ChargedAmount,
		ChargedCurrency: e.ChargedCurrency,
		PaymentMethod:   e.PaymentMethod,
		Category:        e.Category,
		TransferId:      e.TransferID,
		Date:            timestamppb.New(e.Date),
		CreatedAt:       timestamppb.New(e.CreatedAt),
	}
}
