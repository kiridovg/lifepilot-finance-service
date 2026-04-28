package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/testutil"
)

func createAccount(t *testing.T, q *db.Queries, name, currency, balance, date string) db.Account {
	t.Helper()
	userID := testutil.CreateTestUser(t, q)
	acc, err := q.CreateAccount(context.Background(), db.CreateAccountParams{
		UserID:         userID,
		Name:           name,
		Currency:       currency,
		InitialBalance: testutil.Numeric(t, balance),
		InitialDate:    testutil.Date(t, date),
	})
	require.NoError(t, err)
	return acc
}

// T09: Transfer debits fromAccount and credits toAccount.
func TestTransferDebitCredit(t *testing.T) {
	q, _ := testutil.NewQueries(t)

	from := createAccount(t, q, "Wise EUR", "EUR", "100", "2024-01-01")
	to := createAccount(t, q, "Cash EUR", "EUR", "0", "2024-01-01")

	_, err := q.CreateTransfer(context.Background(), db.CreateTransferParams{
		Date:          testutil.Timestamptz(t, "2024-02-01"),
		FromAccountID: from.ID,
		FromAmount:    testutil.Numeric(t, "30"),
		FromCurrency:  testutil.NullText("EUR"),
		ToAccountID:   to.ID,
		ToAmount:      testutil.Numeric(t, "30"),
		ToCurrency:    "EUR",
	})
	require.NoError(t, err)

	assert.InDelta(t, 70.0, calcBalance(t, q, from), 0.0001)
	assert.InDelta(t, 30.0, calcBalance(t, q, to), 0.0001)
}

// T10: Cross-currency transfer (EUR → KZT), amounts differ.
func TestTransferCrossCurrency(t *testing.T) {
	q, _ := testutil.NewQueries(t)

	wise := createAccount(t, q, "Wise EUR", "EUR", "100", "2024-01-01")
	kaspi := createAccount(t, q, "Kaspi KZT", "KZT", "0", "2024-01-01")

	_, err := q.CreateTransfer(context.Background(), db.CreateTransferParams{
		Date:          testutil.Timestamptz(t, "2024-02-01"),
		FromAccountID: wise.ID,
		FromAmount:    testutil.Numeric(t, "18.51"),
		FromCurrency:  testutil.NullText("EUR"),
		ToAccountID:   kaspi.ID,
		ToAmount:      testutil.Numeric(t, "9000"),
		ToCurrency:    "KZT",
	})
	require.NoError(t, err)

	assert.InDelta(t, 81.49, calcBalance(t, q, wise), 0.0001)
	assert.InDelta(t, 9000.0, calcBalance(t, q, kaspi), 0.0001)
}

// T11: Commission is stored for stats but does NOT double-count in balance.
// Wise ATM: withdrew 10000 KZT, debited 18.51 EUR (18.34 net + 0.17 commission).
// Commission expense has transfer_id set → excluded from GetAccountExpenses.
func TestTransferWithCommissionNoDoubleCount(t *testing.T) {
	q, _ := testutil.NewQueries(t)

	wise := createAccount(t, q, "Wise EUR", "EUR", "100", "2024-01-01")
	kaspi := createAccount(t, q, "Kaspi KZT", "KZT", "0", "2024-01-01")

	transfer, err := q.CreateTransfer(context.Background(), db.CreateTransferParams{
		Date:               testutil.Timestamptz(t, "2024-02-01"),
		FromAccountID:      wise.ID,
		FromAmount:         testutil.Numeric(t, "18.51"),
		FromCurrency:       testutil.NullText("EUR"),
		ToAccountID:        kaspi.ID,
		ToAmount:           testutil.Numeric(t, "10000"),
		ToCurrency:         "KZT",
		Commission:         testutil.Numeric(t, "0.17"),
		CommissionCurrency: testutil.NullText("EUR"),
	})
	require.NoError(t, err)

	_, err = q.CreateExpense(context.Background(), db.CreateExpenseParams{
		UserID:     wise.UserID,
		Date:       testutil.Timestamptz(t, "2024-02-01"),
		Amount:     testutil.Numeric(t, "0.17"),
		Currency:   "EUR",
		AccountID:  wise.ID,
		CategoryID: testutil.SystemCategoryID("bank-fees"),
		TransferID: transfer.ID,
	})
	require.NoError(t, err)

	assert.InDelta(t, 81.49, calcBalance(t, q, wise), 0.0001)
}

// T12: Delete transfer restores both account balances.
func TestDeleteTransferRestoresBalances(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	from := createAccount(t, q, "From", "USD", "200", "2024-01-01")
	to := createAccount(t, q, "To", "USD", "50", "2024-01-01")

	tr, err := q.CreateTransfer(ctx, db.CreateTransferParams{
		Date:          testutil.Timestamptz(t, "2024-03-01"),
		FromAccountID: from.ID,
		FromAmount:    testutil.Numeric(t, "100"),
		FromCurrency:  testutil.NullText("USD"),
		ToAccountID:   to.ID,
		ToAmount:      testutil.Numeric(t, "100"),
		ToCurrency:    "USD",
	})
	require.NoError(t, err)

	assert.InDelta(t, 100.0, calcBalance(t, q, from), 0.0001)
	assert.InDelta(t, 150.0, calcBalance(t, q, to), 0.0001)

	require.NoError(t, q.DeleteTransfer(ctx, tr.ID))

	assert.InDelta(t, 200.0, calcBalance(t, q, from), 0.0001)
	assert.InDelta(t, 50.0, calcBalance(t, q, to), 0.0001)
}

// T13: Delete transfer cascades to commission expense.
func TestDeleteTransferCascadesCommission(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	from := createAccount(t, q, "Wise EUR", "EUR", "100", "2024-01-01")
	to := createAccount(t, q, "Kaspi KZT", "KZT", "0", "2024-01-01")

	transfer, err := q.CreateTransfer(ctx, db.CreateTransferParams{
		Date:               testutil.Timestamptz(t, "2024-02-01"),
		FromAccountID:      from.ID,
		FromAmount:         testutil.Numeric(t, "18.51"),
		FromCurrency:       testutil.NullText("EUR"),
		ToAccountID:        to.ID,
		ToAmount:           testutil.Numeric(t, "10000"),
		ToCurrency:         "KZT",
		Commission:         testutil.Numeric(t, "0.17"),
		CommissionCurrency: testutil.NullText("EUR"),
	})
	require.NoError(t, err)

	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:     from.UserID,
		Date:       testutil.Timestamptz(t, "2024-02-01"),
		Amount:     testutil.Numeric(t, "0.17"),
		Currency:   "EUR",
		AccountID:  from.ID,
		CategoryID: testutil.SystemCategoryID("bank-fees"),
		TransferID: transfer.ID,
	})
	require.NoError(t, err)

	require.NoError(t, q.DeleteTransfer(ctx, transfer.ID))

	expenses, err := q.ListExpenses(ctx)
	require.NoError(t, err)
	assert.Empty(t, expenses, "commission expense must be deleted with transfer")

	assert.InDelta(t, 100.0, calcBalance(t, q, from), 0.0001)
}

// T16: ATM withdrawal with two commissions — EUR fee from from_account, KZT fee from to_account.
// Wise EUR → Cash KZT: debited 17.32 EUR (incl. 0.16 EUR conversion fee),
// received 9000 KZT cash, ATM charged 313.27 KZT from dispensed cash.
// Commission2 expense linked to transfer → excluded from GetAccountExpenses.
func TestTransferWithCommission2ATMFee(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	wise := createAccount(t, q, "Wise EUR", "EUR", "100", "2024-01-01")
	cash := createAccount(t, q, "Cash KZT", "KZT", "0", "2024-01-01")

	transfer, err := q.CreateTransfer(ctx, db.CreateTransferParams{
		Date:                testutil.Timestamptz(t, "2024-03-01"),
		FromAccountID:       wise.ID,
		FromAmount:          testutil.Numeric(t, "17.32"),
		FromCurrency:        testutil.NullText("EUR"),
		ToAccountID:         cash.ID,
		ToAmount:            testutil.Numeric(t, "9000"),
		ToCurrency:          "KZT",
		Commission:          testutil.Numeric(t, "0.16"),
		CommissionCurrency:  testutil.NullText("EUR"),
		Commission2:         testutil.Numeric(t, "313.27"),
		Commission2Currency: testutil.NullText("KZT"),
	})
	require.NoError(t, err)

	// Commission1 expense (EUR, from wise)
	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:     wise.UserID,
		Date:       testutil.Timestamptz(t, "2024-03-01"),
		Amount:     testutil.Numeric(t, "0.16"),
		Currency:   "EUR",
		AccountID:  wise.ID,
		CategoryID: testutil.SystemCategoryID("bank-fees"),
		TransferID: transfer.ID,
	})
	require.NoError(t, err)

	// Commission2 expense (KZT, from cash — ATM fee)
	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:     cash.UserID,
		Date:       testutil.Timestamptz(t, "2024-03-01"),
		Amount:     testutil.Numeric(t, "313.27"),
		Currency:   "KZT",
		AccountID:  cash.ID,
		CategoryID: testutil.SystemCategoryID("bank-fees"),
		TransferID: transfer.ID,
	})
	require.NoError(t, err)

	// wise: 100 - 17.32 = 82.68 (commission1 excluded — already in fromAmount)
	assert.InDelta(t, 82.68, calcBalance(t, q, wise), 0.0001)
	// cash: 0 + 9313.27 (transfersIn) - 313.27 (commission2, excluded from GetAccountExpenses) = 9000
	assert.InDelta(t, 9000.0, calcBalance(t, q, cash), 0.0001)
}

// T14ext: Incoming external transfer (income/deposit return) — from_account_id NULL.
func TestExternalIncomingTransfer(t *testing.T) {
	q, _ := testutil.NewQueries(t)

	kaspi := createAccount(t, q, "Kaspi KZT", "KZT", "0", "2024-01-01")

	_, err := q.CreateTransfer(context.Background(), db.CreateTransferParams{
		Date:     testutil.Timestamptz(t, "2024-03-01"),
		ToAmount: testutil.Numeric(t, "10000"),
		ToCurrency: "KZT",
		ToAccountID: kaspi.ID,
		// FromAccountID is zero value (NULL) — external source
	})
	require.NoError(t, err)

	assert.InDelta(t, 10000.0, calcBalance(t, q, kaspi), 0.0001)
}

// T15ext: Deposit sent + returned — linked_transfer_id connects them.
func TestDepositLinkedToReturn(t *testing.T) {
	q, _ := testutil.NewQueries(t)
	ctx := context.Background()

	mono := createAccount(t, q, "Monobank EUR", "EUR", "500", "2024-01-01")
	kaspi := createAccount(t, q, "Kaspi KZT", "KZT", "0", "2024-01-01")

	// Deposit sent: mono → external
	deposit, err := q.CreateTransfer(ctx, db.CreateTransferParams{
		Date:          testutil.Timestamptz(t, "2024-01-15"),
		FromAccountID: mono.ID,
		FromAmount:    testutil.Numeric(t, "500"),
		FromCurrency:  testutil.NullText("EUR"),
		ToAmount:      testutil.Numeric(t, "500"),
		ToCurrency:    "EUR",
		// ToAccountID NULL — external recipient
	})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, calcBalance(t, q, mono), 0.0001)

	// Deposit returned: external → kaspi, linked to original deposit
	_, err = q.CreateTransfer(ctx, db.CreateTransferParams{
		Date:              testutil.Timestamptz(t, "2024-04-20"),
		ToAccountID:       kaspi.ID,
		ToAmount:          testutil.Numeric(t, "250000"),
		ToCurrency:        "KZT",
		LinkedTransferID:  deposit.ID,
		// FromAccountID NULL — external source
	})
	require.NoError(t, err)

	assert.InDelta(t, 0.0, calcBalance(t, q, mono), 0.0001)
	assert.InDelta(t, 250000.0, calcBalance(t, q, kaspi), 0.0001)
}
