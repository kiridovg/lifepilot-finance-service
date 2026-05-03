package handler

import (
	"context"
	"math"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

const BaseCurrency = "EUR"

type ExpenseHandler struct {
	pool *pgxpool.Pool
}

func NewExpenseHandler(pool *pgxpool.Pool) *ExpenseHandler {
	return &ExpenseHandler{pool: pool}
}

func (h *ExpenseHandler) ListExpenses(ctx context.Context, req *connect.Request[financev1.ListExpensesRequest]) (*connect.Response[financev1.ListExpensesResponse], error) {
	q := db.New(h.pool)
	m := req.Msg
	var rows []db.Expense
	var err error

	if m.AccountId != nil {
		rows, err = q.ListExpensesByAccount(ctx, db.ListExpensesByAccountParams{
			AccountID: uuidFromString(*m.AccountId),
			DateFrom:  nullTimestamptz(m.DateFrom),
			DateTo:    nullTimestamptz(m.DateTo),
		})
	} else if m.UserId != nil {
		rows, err = q.ListExpensesByUser(ctx, uuidFromString(*m.UserId))
	} else if m.DateFrom != nil && m.DateTo != nil {
		rows, err = q.ListExpensesByDateRange(ctx, db.ListExpensesByDateRangeParams{
			Date:   pgtype.Timestamptz{Time: m.DateFrom.AsTime(), Valid: true},
			Date_2: pgtype.Timestamptz{Time: m.DateTo.AsTime(), Valid: true},
		})
	} else {
		rows, err = q.ListExpenses(ctx)
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
	var result *financev1.Expense

	err := pgx.BeginFunc(ctx, h.pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		account, err := q.GetAccount(ctx, uuidFromString(m.AccountId))
		if err != nil {
			return err
		}

		baseAmount, baseCurrency := computeBaseAmount(ctx, q, account, m.ChargedAmount, m.ChargedCurrency, m.Amount, m.Currency)

		r, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:          account.UserID,
			AccountID:       uuidFromString(m.AccountId),
			Amount:          numericFromString(m.Amount),
			Currency:        m.Currency,
			ChargedAmount:   nullNumericFromPtr(m.ChargedAmount),
			ChargedCurrency: nullTextFromPtr(m.ChargedCurrency),
			CategoryID:      nullUUIDFromPtr(m.CategoryId),
			Description:     nullTextFromPtr(m.Description),
			Date:            pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true},
			BaseAmount:      baseAmount,
			BaseCurrency:    baseCurrency,
		})
		if err != nil {
			return err
		}
		result = expenseToProto(r)
		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateExpenseResponse{Expense: result}), nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *connect.Request[financev1.UpdateExpenseRequest]) (*connect.Response[financev1.UpdateExpenseResponse], error) {
	q := db.New(h.pool)
	m := req.Msg
	dateTs := pgtype.Timestamptz{}
	if m.Date != nil {
		dateTs = pgtype.Timestamptz{Time: m.Date.AsTime(), Valid: true}
	}
	params := db.UpdateExpenseParams{
		ID:              uuidFromString(m.Id),
		Amount:          nullNumericFromPtr(m.Amount),
		Currency:        nullTextFromPtr(m.Currency),
		ChargedAmount:   nullNumericFromPtr(m.ChargedAmount),
		ChargedCurrency: nullTextFromPtr(m.ChargedCurrency),
		CategoryID:      nullUUIDFromPtr(m.CategoryId),
		Description:     nullTextFromPtr(m.Description),
		Date:            dateTs,
	}
	if m.AccountId != nil {
		acc, err := q.GetAccount(ctx, uuidFromString(*m.AccountId))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		params.AccountID = nullUUIDFromPtr(m.AccountId)
		params.UserID = pgtype.UUID{Bytes: acc.UserID.Bytes, Valid: true}
	}
	r, err := q.UpdateExpense(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.UpdateExpenseResponse{Expense: expenseToProto(r)}), nil
}

func (h *ExpenseHandler) DeleteExpense(ctx context.Context, req *connect.Request[financev1.DeleteExpenseRequest]) (*connect.Response[financev1.DeleteExpenseResponse], error) {
	if err := db.New(h.pool).DeleteExpense(ctx, uuidFromString(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteExpenseResponse{}), nil
}

// computeBaseAmount resolves the base currency amount (EUR) for an expense using FIFO lots.
// Priority: charged_amount if available, otherwise amount. If already in EUR, uses directly.
func computeBaseAmount(ctx context.Context, q *db.Queries, account db.Account, chargedAmount *string, chargedCurrency *string, amount string, currency string) (pgtype.Numeric, pgtype.Text) {
	effectiveAmount := numericToFloat(numericFromString(amount))
	effectiveCurrency := currency

	if chargedAmount != nil && chargedCurrency != nil && *chargedAmount != "" {
		effectiveAmount = numericToFloat(numericFromString(*chargedAmount))
		effectiveCurrency = *chargedCurrency
	}

	if effectiveCurrency == BaseCurrency {
		return numericFromFloat64(effectiveAmount), pgtype.Text{String: BaseCurrency, Valid: true}
	}

	// Consume FIFO lots for this account
	lots, err := q.ListAvailableLots(ctx, account.ID)
	if err != nil || len(lots) == 0 {
		return pgtype.Numeric{}, pgtype.Text{}
	}

	remaining := effectiveAmount
	var baseTotal float64

	for _, lot := range lots {
		if remaining < 1e-9 {
			break
		}
		lotRemaining := numericToFloat(lot.Remaining)
		rate := numericToFloat(lot.RateToBase)
		consume := math.Min(remaining, lotRemaining)
		baseTotal += consume * rate
		remaining -= consume

		newRemaining := math.Max(0, lotRemaining-consume)
		_, _ = q.UpdateLotRemaining(ctx, db.UpdateLotRemainingParams{
			ID:        lot.ID,
			Remaining: numericFromFloat64(newRemaining),
		})
	}

	return numericFromFloat64(baseTotal), pgtype.Text{String: BaseCurrency, Valid: true}
}

// createExpenseWithBase creates a commission/fee expense inside an existing transaction,
// computing base_amount from an explicit rate (used by transfer handler).
func createExpenseWithBase(ctx context.Context, q *db.Queries, params db.CreateExpenseParams, rateToBase float64) (db.Expense, error) {
	if rateToBase > 0 {
		amount := numericToFloat(params.Amount)
		params.BaseAmount = numericFromFloat64(amount * rateToBase)
		params.BaseCurrency = pgtype.Text{String: BaseCurrency, Valid: true}
	}
	return q.CreateExpense(ctx, params)
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
		IsRefund:        r.IsRefund,
		Date:            timestamppb.New(r.Date.Time),
		CreatedAt:       timestamppb.New(r.CreatedAt.Time),
		BaseAmount:      nullNumericToPtr(r.BaseAmount),
		BaseCurrency:    nullTextToPtr(r.BaseCurrency),
	}
}