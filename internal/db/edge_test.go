package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/testutil"
)

// T14: Account with zero initial balance and no ops has balance = 0.
func TestZeroInitialBalance(t *testing.T) {
	q, _ := testutil.NewQueries(t)

	acc := createAccount(t, q, "Empty", "USD", "0", "2024-01-01")
	assert.InDelta(t, 0.0, calcBalance(t, q, acc), 0.0001)
}

// T15: Deactivated account is excluded from ListActiveAccounts.
func TestDeactivatedAccountHidden(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	active := createAccount(t, q, "Active", "USD", "100", "2024-01-01")
	inactive := createAccount(t, q, "Inactive", "USD", "50", "2024-01-01")

	require.NoError(t, q.DeactivateAccount(ctx, inactive.ID))

	accounts, err := q.ListActiveAccounts(ctx)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, active.ID, accounts[0].ID)
}

// T16: Expenses on other accounts do not affect this account's balance.
func TestExpensesIsolatedByAccount(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	accA := createAccount(t, q, "Account A", "USD", "100", "2024-01-01")
	accB := createAccount(t, q, "Account B", "USD", "200", "2024-01-01")

	// Expense on B should not affect A
	_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		Date:       testutil.Timestamptz(t, "2024-02-01"),
		Amount:     testutil.Numeric(t, "50"),
		Currency:   "USD",
		AccountID:  accB.ID,
		CategoryID: testutil.SystemCategoryID("food"),
	})
	require.NoError(t, err)

	assert.InDelta(t, 100.0, calcBalance(t, q, accA), 0.0001)
	assert.InDelta(t, 150.0, calcBalance(t, q, accB), 0.0001)
}
