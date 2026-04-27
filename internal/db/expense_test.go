package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/testutil"
)

func createCashUSD(t *testing.T, q *db.Queries, initialBalance string, initialDate string) db.Account {
	t.Helper()
	acc, err := q.CreateAccount(context.Background(), db.CreateAccountParams{
		UserID:         testutil.CreateTestUser(t, q),
		Name:           "Cash USD",
		Currency:       "USD",
		InitialBalance: testutil.Numeric(t, initialBalance),
		InitialDate:    testutil.Date(t, initialDate),
	})
	require.NoError(t, err)
	return acc
}

func calcBalance(t *testing.T, q *db.Queries, acc db.Account) float64 {
	t.Helper()
	ctx := context.Background()

	expenses, err := q.GetAccountExpenses(ctx, db.GetAccountExpensesParams{
		AccountID:       acc.ID,
		ChargedCurrency: testutil.NullText(acc.Currency),
		Date:            testutil.Timestamptz(t, acc.InitialDate.Time.Format("2006-01-02")),
	})
	require.NoError(t, err)

	out, err := q.GetAccountTransfersOut(ctx, db.GetAccountTransfersOutParams{
		FromAccountID: acc.ID,
		Date:          testutil.Timestamptz(t, acc.InitialDate.Time.Format("2006-01-02")),
	})
	require.NoError(t, err)

	in, err := q.GetAccountTransfersIn(ctx, db.GetAccountTransfersInParams{
		ToAccountID: acc.ID,
		Date:        testutil.Timestamptz(t, acc.InitialDate.Time.Format("2006-01-02")),
	})
	require.NoError(t, err)

	return testutil.NumericToFloat(t, acc.InitialBalance) -
		testutil.NumericToFloat(t, expenses) -
		testutil.NumericToFloat(t, out) +
		testutil.NumericToFloat(t, in)
}

// T04: Spend $1, balance drops from $11 to $10.
func TestExpenseReducesBalance(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc := createCashUSD(t, q, "11", "2024-01-01")

	_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:     acc.UserID,
		Date:       testutil.Timestamptz(t, "2024-01-10"),
		Amount:     testutil.Numeric(t, "1"),
		Currency:   "USD",
		AccountID:  acc.ID,
		CategoryID: testutil.SystemCategoryID("food"),
	})
	require.NoError(t, err)

	assert.InDelta(t, 10.0, calcBalance(t, q, acc), 0.0001)
}

// T05: Multiple expenses accumulate correctly.
func TestMultipleExpenses(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc := createCashUSD(t, q, "100", "2024-01-01")

	for _, amt := range []string{"10", "20", "5"} {
		_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:     acc.UserID,
			Date:       testutil.Timestamptz(t, "2024-02-01"),
			Amount:     testutil.Numeric(t, amt),
			Currency:   "USD",
			AccountID:  acc.ID,
			CategoryID: testutil.SystemCategoryID("food"),
		})
		require.NoError(t, err)
	}

	assert.InDelta(t, 65.0, calcBalance(t, q, acc), 0.0001)
}

// T06: Multi-currency expense uses chargedAmount when chargedCurrency matches account.
func TestChargedAmountUsedForForeignExpense(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc := createCashUSD(t, q, "100", "2024-01-01")

	// Paid 5000 KZT, charged 11.50 USD from account
	_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:          acc.UserID,
		Date:            testutil.Timestamptz(t, "2024-02-01"),
		Amount:          testutil.Numeric(t, "5000"),
		Currency:        "KZT",
		ChargedAmount:   testutil.Numeric(t, "11.50"),
		ChargedCurrency: testutil.NullText("USD"),
		AccountID:       acc.ID,
		CategoryID:      testutil.SystemCategoryID("food"),
	})
	require.NoError(t, err)

	assert.InDelta(t, 88.50, calcBalance(t, q, acc), 0.0001)
}

// T07: Delete expense restores balance.
func TestDeleteExpenseRestoresBalance(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc := createCashUSD(t, q, "50", "2024-01-01")

	exp, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:     acc.UserID,
		Date:       testutil.Timestamptz(t, "2024-01-15"),
		Amount:     testutil.Numeric(t, "15"),
		Currency:   "USD",
		AccountID:  acc.ID,
		CategoryID: testutil.SystemCategoryID("food"),
	})
	require.NoError(t, err)
	assert.InDelta(t, 35.0, calcBalance(t, q, acc), 0.0001)

	require.NoError(t, q.DeleteExpense(ctx, exp.ID))
	assert.InDelta(t, 50.0, calcBalance(t, q, acc), 0.0001)
}

// T08: Update expense changes balance accordingly.
func TestUpdateExpenseChangesBalance(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc := createCashUSD(t, q, "100", "2024-01-01")

	exp, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:     acc.UserID,
		Date:       testutil.Timestamptz(t, "2024-01-20"),
		Amount:     testutil.Numeric(t, "10"),
		Currency:   "USD",
		AccountID:  acc.ID,
		CategoryID: testutil.SystemCategoryID("food"),
	})
	require.NoError(t, err)
	assert.InDelta(t, 90.0, calcBalance(t, q, acc), 0.0001)

	newAmt := testutil.Numeric(t, "25")
	_, err = q.UpdateExpense(ctx, db.UpdateExpenseParams{
		ID:     exp.ID,
		Amount: newAmt,
	})
	require.NoError(t, err)
	assert.InDelta(t, 75.0, calcBalance(t, q, acc), 0.0001)
}
