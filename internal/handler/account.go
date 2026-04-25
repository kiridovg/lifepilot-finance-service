package handler

import (
	"context"
	"strconv"
	"time"

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
	rows, err := q.ListActiveAccounts(ctx)
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
		Name:              m.Name,
		Currency:          m.Currency,
		PaymentMethodCode: nullText(m.PaymentMethodCode),
		InitialBalance:    numericFromString(strconv.FormatFloat(m.InitialBalance, 'f', 2, 64)),
		InitialDate:       m.InitialDate,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateAccountResponse{
		Account: accountToProto(r.ID, r.Name, r.Currency, r.PaymentMethodCode, r.InitialBalance, r.InitialDate, r.IsActive, r.CreatedAt, 0, 0, 0),
	}), nil
}

func (h *AccountHandler) UpdateAccount(ctx context.Context, req *connect.Request[financev1.UpdateAccountRequest]) (*connect.Response[financev1.UpdateAccountResponse], error) {
	m := req.Msg
	var balanceNull pgtype.Numeric
	if m.InitialBalance != nil {
		balanceNull = numericFromString(strconv.FormatFloat(*m.InitialBalance, 'f', 2, 64))
	}

	q := db.New(h.pool)
	r, err := q.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:             m.Id,
		Name:           nullText(m.Name),
		InitialBalance: balanceNull,
		InitialDate:    nullText(m.InitialDate),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	row := db.ListActiveAccountsRow{
		ID:                r.ID,
		Name:              r.Name,
		Currency:          r.Currency,
		PaymentMethodCode: r.PaymentMethodCode,
		InitialBalance:    r.InitialBalance,
		InitialDate:       r.InitialDate,
		IsActive:          r.IsActive,
		CreatedAt:         r.CreatedAt,
	}
	a, err := h.withBalance(ctx, q, row)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.UpdateAccountResponse{Account: a}), nil
}

func (h *AccountHandler) DeleteAccount(ctx context.Context, req *connect.Request[financev1.DeleteAccountRequest]) (*connect.Response[financev1.DeleteAccountResponse], error) {
	if err := db.New(h.pool).DeleteAccount(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteAccountResponse{}), nil
}

func (h *AccountHandler) withBalance(ctx context.Context, q *db.Queries, r db.ListActiveAccountsRow) (*financev1.Account, error) {
	if !r.PaymentMethodCode.Valid {
		return accountToProto(r.ID, r.Name, r.Currency, r.PaymentMethodCode, r.InitialBalance, r.InitialDate, r.IsActive, r.CreatedAt, 0, 0, 0), nil
	}

	initialDate, err := time.Parse("2006-01-02", r.InitialDate)
	if err != nil {
		return nil, err
	}
	ts := pgtype.Timestamptz{Time: initialDate, Valid: true}

	expTotal, err := q.GetAccountExpenses(ctx, db.GetAccountExpensesParams{
		ChargedCurrency: pgtype.Text{String: r.Currency, Valid: true}, // $1 = account currency
		PaymentMethod:   r.PaymentMethodCode.String,
		Column3:         ts,
	})
	if err != nil {
		return nil, err
	}
	outTotal, err := q.GetAccountTransfersOut(ctx, db.GetAccountTransfersOutParams{
		FromPaymentMethod: r.PaymentMethodCode,
		FromCurrency:      r.Currency,
		Column3:           ts,
	})
	if err != nil {
		return nil, err
	}
	inTotal, err := q.GetAccountTransfersIn(ctx, db.GetAccountTransfersInParams{
		ToPaymentMethod: r.PaymentMethodCode,
		ToCurrency:      r.Currency,
		Column3:         ts,
	})
	if err != nil {
		return nil, err
	}

	expenses := numericToFloat(expTotal)
	out := numericToFloat(outTotal)
	in := numericToFloat(inTotal)

	return accountToProto(r.ID, r.Name, r.Currency, r.PaymentMethodCode, r.InitialBalance, r.InitialDate, r.IsActive, r.CreatedAt, expenses, out, in), nil
}

func accountToProto(
	id int32, name, currency string, pm pgtype.Text, initialBalance pgtype.Numeric,
	initialDate string, isActive bool, createdAt pgtype.Timestamptz,
	totalExpenses, transfersOut, transfersIn float64,
) *financev1.Account {
	initial := numericToFloat(initialBalance)
	balance := initial - totalExpenses - transfersOut + transfersIn

	return &financev1.Account{
		Id:                id,
		Name:              name,
		Currency:          currency,
		PaymentMethodCode: nullTextToPtr(pm),
		InitialBalance:    numericToString(initialBalance),
		InitialDate:       initialDate,
		IsActive:          isActive,
		CreatedAt:         timestamppb.New(createdAt.Time),
		Balance:           balance,
		TotalExpenses:     totalExpenses,
		TransfersOut:      transfersOut,
		TransfersIn:       transfersIn,
	}
}

func numericToFloat(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}
