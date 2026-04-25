package handler

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/repository"
)

type AccountHandler struct {
	repo *repository.Repository
}

func NewAccountHandler(repo *repository.Repository) *AccountHandler {
	return &AccountHandler{repo: repo}
}

func (h *AccountHandler) ListAccounts(ctx context.Context, req *connect.Request[financev1.ListAccountsRequest]) (*connect.Response[financev1.ListAccountsResponse], error) {
	accounts, err := h.repo.ListAccounts(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var proto []*financev1.Account
	for _, a := range accounts {
		proto = append(proto, accountToProto(a))
	}
	return connect.NewResponse(&financev1.ListAccountsResponse{Accounts: proto}), nil
}

func (h *AccountHandler) CreateAccount(ctx context.Context, req *connect.Request[financev1.CreateAccountRequest]) (*connect.Response[financev1.CreateAccountResponse], error) {
	a, err := h.repo.CreateAccount(ctx,
		req.Msg.Name, req.Msg.Currency, req.Msg.PaymentMethodCode,
		req.Msg.InitialBalance, req.Msg.InitialDate,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateAccountResponse{Account: accountToProto(a)}), nil
}

func (h *AccountHandler) UpdateAccount(ctx context.Context, req *connect.Request[financev1.UpdateAccountRequest]) (*connect.Response[financev1.UpdateAccountResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (h *AccountHandler) DeleteAccount(ctx context.Context, req *connect.Request[financev1.DeleteAccountRequest]) (*connect.Response[financev1.DeleteAccountResponse], error) {
	if err := h.repo.DeleteAccount(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteAccountResponse{}), nil
}

func accountToProto(a repository.Account) *financev1.Account {
	return &financev1.Account{
		Id:              a.ID,
		Name:            a.Name,
		Currency:        a.Currency,
		PaymentMethodCode: a.PaymentMethod,
		InitialBalance:  a.InitialBalance,
		InitialDate:     a.InitialDate,
		IsActive:        a.IsActive,
		CreatedAt:       timestamppb.New(a.CreatedAt),
		Balance:         a.Balance,
		TotalExpenses:   a.TotalExpenses,
		TransfersOut:    a.TransfersOut,
		TransfersIn:     a.TransfersIn,
	}
}
