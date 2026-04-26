package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/testutil"
)

// T01: Create account, list it back.
func TestCreateAndListAccount(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
		Name:           "Cash USD",
		Currency:       "USD",
		InitialBalance: testutil.Numeric(t, "100"),
		InitialDate:    testutil.Date(t, "2024-01-01"),
	})
	require.NoError(t, err)
	assert.Equal(t, "Cash USD", acc.Name)
	assert.Equal(t, "USD", acc.Currency)
	assert.True(t, acc.IsActive)

	accounts, err := q.ListActiveAccounts(ctx)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, acc.ID, accounts[0].ID)
}

// T02: Balance without any operations = initialBalance.
func TestBalanceNoOperations(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
		Name:           "Cash USD",
		Currency:       "USD",
		InitialBalance: testutil.Numeric(t, "11"),
		InitialDate:    testutil.Date(t, "2024-01-01"),
	})
	require.NoError(t, err)

	expenses, err := q.GetAccountExpenses(ctx, db.GetAccountExpensesParams{
		AccountID:       acc.ID,
		ChargedCurrency: testutil.NullText("USD"),
		Date:            testutil.Timestamptz(t, "2024-01-01"),
	})
	require.NoError(t, err)

	out, err := q.GetAccountTransfersOut(ctx, db.GetAccountTransfersOutParams{
		FromAccountID: acc.ID,
		Date:          testutil.Timestamptz(t, "2024-01-01"),
	})
	require.NoError(t, err)

	in, err := q.GetAccountTransfersIn(ctx, db.GetAccountTransfersInParams{
		ToAccountID: acc.ID,
		Date:        testutil.Timestamptz(t, "2024-01-01"),
	})
	require.NoError(t, err)

	initialBalance := testutil.NumericToFloat(t, acc.InitialBalance)
	totalExpenses := testutil.NumericToFloat(t, expenses)
	transfersOut := testutil.NumericToFloat(t, out)
	transfersIn := testutil.NumericToFloat(t, in)

	balance := initialBalance - totalExpenses - transfersOut + transfersIn
	assert.InDelta(t, 11.0, balance, 0.0001)
}

// T03: Balance ignores expenses created before initialDate.
func TestBalanceIgnoresOldExpenses(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
		Name:           "Old Account",
		Currency:       "USD",
		InitialBalance: testutil.Numeric(t, "50"),
		InitialDate:    testutil.Date(t, "2024-06-01"),
	})
	require.NoError(t, err)

	// Expense before initialDate — should be ignored in balance
	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		Date:       testutil.Timestamptz(t, "2024-01-15"),
		Amount:     testutil.Numeric(t, "20"),
		Currency:   "USD",
		AccountID:  acc.ID,
		CategoryID: testutil.SystemCategoryID("food"),
	})
	require.NoError(t, err)

	expenses, err := q.GetAccountExpenses(ctx, db.GetAccountExpensesParams{
		AccountID:       acc.ID,
		ChargedCurrency: testutil.NullText("USD"),
		Date:            testutil.Timestamptz(t, "2024-06-01"),
	})
	require.NoError(t, err)

	out, err := q.GetAccountTransfersOut(ctx, db.GetAccountTransfersOutParams{
		FromAccountID: acc.ID,
		Date:          testutil.Timestamptz(t, "2024-06-01"),
	})
	require.NoError(t, err)

	in, err := q.GetAccountTransfersIn(ctx, db.GetAccountTransfersInParams{
		ToAccountID: acc.ID,
		Date:        testutil.Timestamptz(t, "2024-06-01"),
	})
	require.NoError(t, err)

	initialBalance := testutil.NumericToFloat(t, acc.InitialBalance)
	totalExpenses := testutil.NumericToFloat(t, expenses)
	transfersOut := testutil.NumericToFloat(t, out)
	transfersIn := testutil.NumericToFloat(t, in)

	balance := initialBalance - totalExpenses - transfersOut + transfersIn
	assert.InDelta(t, 50.0, balance, 0.0001, "expense before initialDate must not affect balance")
}
