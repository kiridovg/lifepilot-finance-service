package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type AccountHandler struct {
	pool *pgxpool.Pool
}

func NewAccountHandler(pool *pgxpool.Pool) *AccountHandler {
	return &AccountHandler{pool: pool}
}

func (h *AccountHandler) ListAccounts(ctx context.Context, req *connect.Request[financev1.ListAccountsRequest]) (*connect.Response[financev1.ListAccountsResponse], error) {
	q := db.New(h.pool)

	var rows []db.Account
	var err error
	if req.Msg.UserId != nil {
		rows, err = q.ListActiveAccountsByUser(ctx, uuidFromString(*req.Msg.UserId))
	} else {
		rows, err = q.ListActiveAccounts(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accounts := make([]*financev1.Account, 0, len(rows))
	for _, r := range rows {
		a, err := h.withBalance(ctx, q, r)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		accounts = append(accounts, a)
	}
	return connect.NewResponse(&financev1.ListAccountsResponse{Accounts: accounts}), nil
}

func (h *AccountHandler) CreateAccount(ctx context.Context, req *connect.Request[financev1.CreateAccountRequest]) (*connect.Response[financev1.CreateAccountResponse], error) {
	m := req.Msg
	r, err := db.New(h.pool).CreateAccount(ctx, db.CreateAccountParams{
		UserID:         uuidFromString(m.UserId),
		Name:           m.Name,
		Currency:       m.Currency,
		PaymentMethod:  nullTextFromPtr(m.PaymentMethod),
		InitialBalance: numericFromString(m.InitialBalance),
		InitialDate:    dateFromString(m.InitialDate),
		Notes:          nullTextFromPtr(m.Notes),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateAccountResponse{
		Account: accountToProto(r, 0, 0, 0),
	}), nil
}

func (h *AccountHandler) UpdateAccount(ctx context.Context, req *connect.Request[financev1.UpdateAccountRequest]) (*connect.Response[financev1.UpdateAccountResponse], error) {
	m := req.Msg
	q := db.New(h.pool)

	r, err := q.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:             uuidFromString(m.Id),
		Name:           nullTextFromPtr(m.Name),
		InitialBalance: nullNumericFromPtr(m.InitialBalance),
		InitialDate:    nullDateFromPtr(m.InitialDate),
		Notes:          nullTextFromPtr(m.Notes),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	a, err := h.withBalance(ctx, q, r)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.UpdateAccountResponse{Account: a}), nil
}

func (h *AccountHandler) DeactivateAccount(ctx context.Context, req *connect.Request[financev1.DeactivateAccountRequest]) (*connect.Response[financev1.DeactivateAccountResponse], error) {
	if err := db.New(h.pool).DeactivateAccount(ctx, uuidFromString(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeactivateAccountResponse{}), nil
}

func (h *AccountHandler) withBalance(ctx context.Context, q *db.Queries, r db.Account) (*financev1.Account, error) {
	ts := pgtype.Timestamptz{Time: r.InitialDate.Time, Valid: true}

	expTotal, err := q.GetAccountExpenses(ctx, db.GetAccountExpensesParams{
		AccountID:       r.ID,
		ChargedCurrency: pgtype.Text{String: r.Currency, Valid: true},
		Date:            ts,
	})
	if err != nil {
		return nil, err
	}
	outTotal, err := q.GetAccountTransfersOut(ctx, db.GetAccountTransfersOutParams{
		FromAccountID: r.ID,
		Date:          ts,
	})
	if err != nil {
		return nil, err
	}
	inTotal, err := q.GetAccountTransfersIn(ctx, db.GetAccountTransfersInParams{
		ToAccountID: r.ID,
		Date:        ts,
	})
	if err != nil {
		return nil, err
	}

	return accountToProto(r,
		numericToFloat(expTotal),
		numericToFloat(outTotal),
		numericToFloat(inTotal),
	), nil
}

func accountToProto(r db.Account, totalExpenses, transfersOut, transfersIn float64) *financev1.Account {
	initial := numericToFloat(r.InitialBalance)
	balance := initial - totalExpenses - transfersOut + transfersIn

	return &financev1.Account{
		Id:             uuidToString(r.ID),
		UserId:         uuidToString(r.UserID),
		Name:           r.Name,
		Currency:       r.Currency,
		PaymentMethod:  nullTextToPtr(r.PaymentMethod),
		InitialBalance: numericToString(r.InitialBalance),
		InitialDate:    r.InitialDate.Time.Format("2006-01-02"),
		IsActive:       r.IsActive,
		Notes:          nullTextToPtr(r.Notes),
		CreatedAt:      timestamppb.New(r.CreatedAt.Time),
		Balance:        balance,
		TotalExpenses:  totalExpenses,
		TransfersOut:   transfersOut,
		TransfersIn:    transfersIn,
	}
}
