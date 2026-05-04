package handler_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/handler"
	"github.com/kiridovg/lifepilot-finance-service/internal/testutil"
)

func ts(date string) *timestamppb.Timestamp {
	t, _ := time.Parse("2006-01-02", date)
	return timestamppb.New(t)
}

func ptr(s string) *string { return &s }

func uuidToStr(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseFloat(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	require.NoError(t, err)
	return f
}

func buildAccount(t *testing.T, q *db.Queries, name, currency string) db.Account {
	t.Helper()
	acc, err := q.CreateAccount(context.Background(), db.CreateAccountParams{
		UserID:         testutil.CreateTestUser(t, q),
		Name:           name,
		Currency:       currency,
		InitialBalance: testutil.Numeric(t, "0"),
		InitialDate:    testutil.Date(t, "2026-01-01"),
	})
	require.NoError(t, err)
	return acc
}

func totalLotRemaining(t *testing.T, q *db.Queries, accountID pgtype.UUID) float64 {
	t.Helper()
	lots, err := q.ListAvailableLots(context.Background(), accountID)
	require.NoError(t, err)
	var sum float64
	for _, l := range lots {
		sum += testutil.NumericToFloat(t, l.Remaining)
	}
	return sum
}

// CreateTransfer EUR→KZT with rate creates a FIFO lot for the destination account.
func TestCreateTransfer_CreatesLot(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	wise := buildAccount(t, q, "Wise EUR", "EUR")
	cash := buildAccount(t, q, "Cash KZT", "KZT")

	h := handler.NewTransferHandler(pool)
	_, err := h.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2026-04-21"),
		FromAccountId: ptr(uuidToStr(wise.ID)),
		FromAmount:    ptr("200.00"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(uuidToStr(cash.ID)),
		ToAmount:      "91000.00",
		ToCurrency:    "KZT",
		Rate:          ptr("0.0021978"),
	}))
	require.NoError(t, err)

	lots, err := q.ListAvailableLots(ctx, cash.ID)
	require.NoError(t, err)
	require.Len(t, lots, 1)
	assert.InDelta(t, 91000.0, testutil.NumericToFloat(t, lots[0].OriginalAmount), 0.01)
	assert.InDelta(t, 0.0021978, testutil.NumericToFloat(t, lots[0].RateToBase), 1e-7)
	assert.Equal(t, "EUR", lots[0].BaseCurrency)
}

// Commission2 (KZT ATM fee) is immediately consumed from the newly created lot.
func TestCreateTransfer_Commission2ConsumesLot(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	wise := buildAccount(t, q, "Wise EUR", "EUR")
	cash := buildAccount(t, q, "Cash KZT", "KZT")

	h := handler.NewTransferHandler(pool)
	_, err := h.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:                ts("2026-04-21"),
		FromAccountId:       ptr(uuidToStr(wise.ID)),
		FromAmount:          ptr("200.16"),
		FromCurrency:        ptr("EUR"),
		ToAccountId:         ptr(uuidToStr(cash.ID)),
		ToAmount:            "91313.27",
		ToCurrency:          "KZT",
		Commission2:         ptr("313.27"),
		Commission2Currency: ptr("KZT"),
		Rate:                ptr("0.0021978"),
	}))
	require.NoError(t, err)

	// Lot was 91313.27; commission2 consumed 313.27 KZT → remaining = 91000
	remaining := totalLotRemaining(t, q, cash.ID)
	assert.InDelta(t, 91000.0, remaining, 0.01)
}

// No rate provided → no lot created, but transfer still works.
func TestCreateTransfer_NoRate_NoLot(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	wise := buildAccount(t, q, "Wise EUR", "EUR")
	cash := buildAccount(t, q, "Cash KZT", "KZT")

	h := handler.NewTransferHandler(pool)
	_, err := h.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2026-04-21"),
		FromAccountId: ptr(uuidToStr(wise.ID)),
		FromAmount:    ptr("200.00"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(uuidToStr(cash.ID)),
		ToAmount:      "91000.00",
		ToCurrency:    "KZT",
		// Rate not provided
	}))
	require.NoError(t, err)

	lots, err := q.ListAvailableLots(ctx, cash.ID)
	require.NoError(t, err)
	assert.Empty(t, lots)
}

// CreateExpense from KZT account consumes lot and sets base_amount.
func TestCreateExpense_FIFO_SetsBaseAmount(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	cash := buildAccount(t, q, "Cash KZT", "KZT")

	// Seed lot: 91000 KZT at 0.0021978 EUR/KZT
	_, err := q.CreateLot(ctx, db.CreateLotParams{
		AccountID:      cash.ID,
		OriginalAmount: testutil.Numeric(t, "91000"),
		RateToBase:     testutil.Numeric(t, "0.0021978"),
		Remaining:      testutil.Numeric(t, "91000"),
		BaseCurrency:   "EUR",
		Date:           testutil.Timestamptz(t, "2026-04-21"),
	})
	require.NoError(t, err)

	eh := handler.NewExpenseHandler(pool)
	res, err := eh.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		UserId:    uuidToStr(cash.UserID),
		AccountId: uuidToStr(cash.ID),
		Date:      ts("2026-04-21"),
		Amount:    "1590.00",
		Currency:  "KZT",
	}))
	require.NoError(t, err)

	exp := res.Msg.Expense
	require.NotNil(t, exp.BaseAmount)
	require.NotNil(t, exp.BaseCurrency)
	assert.Equal(t, "EUR", *exp.BaseCurrency)
	// 1590 * 0.0021978 ≈ 3.4945 EUR
	assert.InDelta(t, 3.4945, parseFloat(t, *exp.BaseAmount), 0.01)

	// Lot remaining: 91000 - 1590 = 89410
	assert.InDelta(t, 89410.0, totalLotRemaining(t, q, cash.ID), 0.01)
}

// CreateExpense from EUR account sets base_amount = amount directly.
func TestCreateExpense_EUR_SetsBaseAmount(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	revolut := buildAccount(t, q, "Revolut EUR", "EUR")

	eh := handler.NewExpenseHandler(pool)
	res, err := eh.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		UserId:    uuidToStr(revolut.UserID),
		AccountId: uuidToStr(revolut.ID),
		Date:      ts("2026-04-21"),
		Amount:    "65.50",
		Currency:  "EUR",
	}))
	require.NoError(t, err)

	exp := res.Msg.Expense
	require.NotNil(t, exp.BaseAmount)
	assert.InDelta(t, 65.50, parseFloat(t, *exp.BaseAmount), 0.001)
	assert.Equal(t, "EUR", *exp.BaseCurrency)
}

// GetMonthStats sums base_amount for the month, excludes other months.
func TestGetMonthStats(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	acc := buildAccount(t, q, "Revolut EUR", "EUR")

	insert := func(date, amt string) {
		t.Helper()
		_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:       acc.UserID,
			AccountID:    acc.ID,
			Date:         testutil.Timestamptz(t, date),
			Amount:       testutil.Numeric(t, amt),
			Currency:     "EUR",
			BaseAmount:   testutil.Numeric(t, amt),
			BaseCurrency: testutil.NullText("EUR"),
		})
		require.NoError(t, err)
	}

	insert("2026-04-05", "100.00") // ✓ included
	insert("2026-04-20", "50.00")  // ✓ included
	insert("2026-03-15", "999.00") // ✗ different month

	sh := handler.NewStatsHandler(pool)
	res, err := sh.GetMonthStats(ctx, connect.NewRequest(&financev1.GetMonthStatsRequest{
		Year: 2026, Month: 4, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 150.0, parseFloat(t, res.Msg.Total), 0.001)
	assert.EqualValues(t, 2, res.Msg.Count)
	assert.EqualValues(t, 30, res.Msg.DaysInMonth)
	assert.InDelta(t, 5.0, parseFloat(t, res.Msg.AveragePerDay), 0.001) // 150/30
}

// GetYearStats breaks down expenses by month for the year.
func TestGetYearStats_MonthlyBreakdown(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	acc := buildAccount(t, q, "Revolut EUR", "EUR")

	for _, tc := range []struct{ date, amt string }{
		{"2026-01-15", "100.00"},
		{"2026-01-20", "50.00"},
		{"2026-03-10", "200.00"},
		{"2026-04-15", "75.00"},
		{"2025-12-01", "999.00"}, // different year — excluded
	} {
		_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:       acc.UserID,
			AccountID:    acc.ID,
			Date:         testutil.Timestamptz(t, tc.date),
			Amount:       testutil.Numeric(t, tc.amt),
			Currency:     "EUR",
			BaseAmount:   testutil.Numeric(t, tc.amt),
			BaseCurrency: testutil.NullText("EUR"),
		})
		require.NoError(t, err)
	}

	sh := handler.NewStatsHandler(pool)
	res, err := sh.GetYearStats(ctx, connect.NewRequest(&financev1.GetYearStatsRequest{
		Year: 2026, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 425.0, parseFloat(t, res.Msg.Total), 0.001)
	require.Len(t, res.Msg.MonthlyStats, 3, "Jan, Mar, Apr")

	byMonth := map[int32]float64{}
	for _, ms := range res.Msg.MonthlyStats {
		byMonth[ms.Month] = parseFloat(t, ms.Total)
	}
	assert.InDelta(t, 150.0, byMonth[1], 0.001)
	assert.InDelta(t, 200.0, byMonth[3], 0.001)
	assert.InDelta(t, 75.0, byMonth[4], 0.001)
}

// GetAllTimeStats returns all months chronologically across years.
func TestGetAllTimeStats_MultipleYears(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	acc := buildAccount(t, q, "Revolut EUR", "EUR")

	for _, tc := range []struct{ date, amt string }{
		{"2025-11-10", "300.00"},
		{"2026-01-15", "100.00"},
		{"2026-04-15", "75.00"},
	} {
		_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:       acc.UserID,
			AccountID:    acc.ID,
			Date:         testutil.Timestamptz(t, tc.date),
			Amount:       testutil.Numeric(t, tc.amt),
			Currency:     "EUR",
			BaseAmount:   testutil.Numeric(t, tc.amt),
			BaseCurrency: testutil.NullText("EUR"),
		})
		require.NoError(t, err)
	}

	sh := handler.NewStatsHandler(pool)
	res, err := sh.GetAllTimeStats(ctx, connect.NewRequest(&financev1.GetAllTimeStatsRequest{
		BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 475.0, parseFloat(t, res.Msg.Total), 0.001)
	require.Len(t, res.Msg.MonthlyStats, 3)
	assert.EqualValues(t, 2025, res.Msg.MonthlyStats[0].Year)
	assert.EqualValues(t, 11, res.Msg.MonthlyStats[0].Month)
	assert.InDelta(t, 300.0, parseFloat(t, res.Msg.MonthlyStats[0].Total), 0.001)
}

// DeleteTransfer cascades: associated lot is deleted.
func TestDeleteTransfer_CascadesLot(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	wise := buildAccount(t, q, "Wise EUR", "EUR")
	cash := buildAccount(t, q, "Cash KZT", "KZT")

	h := handler.NewTransferHandler(pool)
	res, err := h.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2026-04-21"),
		FromAccountId: ptr(uuidToStr(wise.ID)),
		FromAmount:    ptr("200.00"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(uuidToStr(cash.ID)),
		ToAmount:      "91000.00",
		ToCurrency:    "KZT",
		Rate:          ptr("0.0021978"),
	}))
	require.NoError(t, err)

	lots, err := q.ListAvailableLots(ctx, cash.ID)
	require.NoError(t, err)
	require.Len(t, lots, 1)

	_, err = h.DeleteTransfer(ctx, connect.NewRequest(&financev1.DeleteTransferRequest{
		Id: res.Msg.Transfer.Id,
	}))
	require.NoError(t, err)

	lots, err = q.ListAvailableLots(ctx, cash.ID)
	require.NoError(t, err)
	assert.Empty(t, lots, "lot must be deleted when transfer is deleted")
}

// Expenses without base_amount are excluded from GetMonthStats.
func TestGetMonthStats_ExcludesNullBaseAmount(t *testing.T) {
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()

	acc := buildAccount(t, q, "Cash KZT", "KZT")

	// Expense with base_amount
	_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:       acc.UserID,
		AccountID:    acc.ID,
		Date:         testutil.Timestamptz(t, "2026-04-10"),
		Amount:       testutil.Numeric(t, "1000.00"),
		Currency:     "KZT",
		BaseAmount:   testutil.Numeric(t, "2.20"),
		BaseCurrency: testutil.NullText("EUR"),
	})
	require.NoError(t, err)

	// Expense without base_amount (no lot available)
	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:    acc.UserID,
		AccountID: acc.ID,
		Date:      testutil.Timestamptz(t, "2026-04-15"),
		Amount:    testutil.Numeric(t, "500.00"),
		Currency:  "KZT",
	})
	require.NoError(t, err)

	sh := handler.NewStatsHandler(pool)
	res, err := sh.GetMonthStats(ctx, connect.NewRequest(&financev1.GetMonthStatsRequest{
		Year: 2026, Month: 4, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.EqualValues(t, 1, res.Msg.Count, "only expense with base_amount is counted")
	assert.InDelta(t, 2.20, parseFloat(t, res.Msg.Total), 0.001)
}

// GetMonthStats for a month with no expenses returns zeros.
func TestGetMonthStats_EmptyMonth(t *testing.T) {
	_, pool := testutil.NewQueries(t)
	ctx := context.Background()

	sh := handler.NewStatsHandler(pool)
	res, err := sh.GetMonthStats(ctx, connect.NewRequest(&financev1.GetMonthStatsRequest{
		Year: 2026, Month: 6, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 0.0, parseFloat(t, res.Msg.Total), 0.001)
	assert.EqualValues(t, 0, res.Msg.Count)
	assert.EqualValues(t, 30, res.Msg.DaysInMonth)
}