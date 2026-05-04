package handler

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/testutil"
)

// --- helpers ---

func makeAccount(t *testing.T, q *db.Queries, name, currency string) db.Account {
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

func seedLot(t *testing.T, q *db.Queries, accountID pgtype.UUID, amount, rateToBase, date string) db.AccountLot {
	t.Helper()
	lot, err := q.CreateLot(context.Background(), db.CreateLotParams{
		AccountID:      accountID,
		OriginalAmount: testutil.Numeric(t, amount),
		RateToBase:     testutil.Numeric(t, rateToBase),
		Remaining:      testutil.Numeric(t, amount),
		BaseCurrency:   BaseCurrency,
		Date:           testutil.Timestamptz(t, date),
	})
	require.NoError(t, err)
	return lot
}

func totalRemaining(t *testing.T, q *db.Queries, accountID pgtype.UUID) float64 {
	t.Helper()
	lots, err := q.ListAvailableLots(context.Background(), accountID)
	require.NoError(t, err)
	var sum float64
	for _, l := range lots {
		sum += testutil.NumericToFloat(t, l.Remaining)
	}
	return sum
}

// --- computeBaseAmount tests ---

// EUR account expense → base_amount = amount, no lots involved.
func TestComputeBaseAmount_EUR_Direct(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	acc := makeAccount(t, q, "Revolut EUR", "EUR")

	base, cur := computeBaseAmount(context.Background(), q, acc, nil, nil, "150.00", "EUR")

	assert.InDelta(t, 150.0, testutil.NumericToFloat(t, base), 0.001)
	assert.Equal(t, BaseCurrency, cur.String)
}

// charged_amount in EUR overrides amount in foreign currency.
func TestComputeBaseAmount_ChargedEUR_Priority(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	acc := makeAccount(t, q, "Revolut EUR", "EUR")

	ca, cc := "22.50", "EUR"
	base, cur := computeBaseAmount(context.Background(), q, acc, &ca, &cc, "100.00", "RON")

	assert.InDelta(t, 22.50, testutil.NumericToFloat(t, base), 0.001)
	assert.Equal(t, BaseCurrency, cur.String)
}

// KZT account with no lots → base_amount is NULL.
func TestComputeBaseAmount_NoLots_ReturnsNull(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	acc := makeAccount(t, q, "Cash KZT", "KZT")

	base, cur := computeBaseAmount(context.Background(), q, acc, nil, nil, "1590.00", "KZT")

	assert.False(t, base.Valid)
	assert.False(t, cur.Valid)
}

// Single lot, expense smaller than lot → base_amount computed, remaining decremented.
func TestComputeBaseAmount_FIFO_SingleLot(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	acc := makeAccount(t, q, "Cash KZT", "KZT")
	// 200 EUR → 91000 KZT: rate = 200/91000 ≈ 0.0021978 EUR/KZT
	seedLot(t, q, acc.ID, "91000", "0.0021978", "2026-04-21")

	base, cur := computeBaseAmount(context.Background(), q, acc, nil, nil, "1590.00", "KZT")

	assert.InDelta(t, 1590*0.0021978, testutil.NumericToFloat(t, base), 0.001)
	assert.Equal(t, BaseCurrency, cur.String)
	assert.InDelta(t, 89410.0, totalRemaining(t, q, acc.ID), 0.01)
}

// Expense spans two lots (FIFO: oldest first).
func TestComputeBaseAmount_FIFO_SpansMultipleLots(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	acc := makeAccount(t, q, "Cash KZT", "KZT")
	seedLot(t, q, acc.ID, "50000", "0.002198", "2026-04-20") // lot 1 (older)
	seedLot(t, q, acc.ID, "50000", "0.002222", "2026-04-23") // lot 2 (newer)

	// 70000 KZT: 50000 from lot1 + 20000 from lot2
	base, _ := computeBaseAmount(context.Background(), q, acc, nil, nil, "70000.00", "KZT")

	// 50000*0.002198 + 20000*0.002222 = 109.90 + 44.44 = 154.34
	assert.InDelta(t, 154.34, testutil.NumericToFloat(t, base), 0.01)
	// Lot 1 exhausted, Lot 2 remaining = 30000
	assert.InDelta(t, 30000.0, totalRemaining(t, q, acc.ID), 0.01)
}

// Expense larger than all lots → uses available amount, rest untracked.
func TestComputeBaseAmount_FIFO_LotExhaustion(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	acc := makeAccount(t, q, "Cash KZT", "KZT")
	seedLot(t, q, acc.ID, "5000", "0.002198", "2026-04-21")

	base, _ := computeBaseAmount(context.Background(), q, acc, nil, nil, "10000.00", "KZT")

	// Only 5000 KZT converted: 5000 * 0.002198 = 10.99 EUR
	assert.InDelta(t, 10.99, testutil.NumericToFloat(t, base), 0.01)
	assert.InDelta(t, 0.0, totalRemaining(t, q, acc.ID), 0.01)
}

// Sequential expenses correctly consume the same lot across calls.
func TestComputeBaseAmount_FIFO_Sequential(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()
	acc := makeAccount(t, q, "Cash KZT", "KZT")
	seedLot(t, q, acc.ID, "3000", "0.002198", "2026-04-21")

	b1, _ := computeBaseAmount(ctx, q, acc, nil, nil, "1590.00", "KZT")
	assert.InDelta(t, 1590*0.002198, testutil.NumericToFloat(t, b1), 0.001)

	b2, _ := computeBaseAmount(ctx, q, acc, nil, nil, "1000.00", "KZT")
	assert.InDelta(t, 1000*0.002198, testutil.NumericToFloat(t, b2), 0.001)

	// 3000 - 1590 - 1000 = 410 remaining
	assert.InDelta(t, 410.0, totalRemaining(t, q, acc.ID), 0.01)
}