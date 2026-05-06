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

// --- helpers ---

func ts(date string) *timestamppb.Timestamp {
	t, _ := time.Parse("2006-01-02", date)
	return timestamppb.New(t)
}

func ptr(s string) *string { return &s }

func pf(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	require.NoError(t, err)
	return f
}

func uuidStr(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// hs bundles all service handlers sharing one DB pool.
type hs struct {
	account  *handler.AccountHandler
	expense  *handler.ExpenseHandler
	income   *handler.IncomeHandler
	transfer *handler.TransferHandler
	stats    *handler.StatsHandler
}

// setup spins up a fresh Postgres container, creates all handlers, and returns a test user ID.
// The only direct DB call here is CreateUser — there is no gRPC endpoint for it.
func setup(t *testing.T) (context.Context, hs, string) {
	t.Helper()
	q, pool := testutil.NewQueries(t)
	ctx := context.Background()
	userID := uuidStr(testutil.CreateTestUser(t, q))
	h := hs{
		account:  handler.NewAccountHandler(pool),
		expense:  handler.NewExpenseHandler(pool),
		income:   handler.NewIncomeHandler(db.New(pool)),
		transfer: handler.NewTransferHandler(pool),
		stats:    handler.NewStatsHandler(pool),
	}
	return ctx, h, userID
}

// newAccount creates an account via the service and returns the proto response.
func newAccount(t *testing.T, ctx context.Context, h hs, userID, name, currency, balance, date string) *financev1.Account {
	t.Helper()
	res, err := h.account.CreateAccount(ctx, connect.NewRequest(&financev1.CreateAccountRequest{
		UserId:         userID,
		Name:           name,
		Currency:       currency,
		InitialBalance: balance,
		InitialDate:    date,
	}))
	require.NoError(t, err)
	return res.Msg.Account
}

// balance fetches the current balance of an account via ListAccounts.
func balance(t *testing.T, ctx context.Context, h hs, userID, accountID string) float64 {
	t.Helper()
	res, err := h.account.ListAccounts(ctx, connect.NewRequest(&financev1.ListAccountsRequest{
		UserId: &userID,
	}))
	require.NoError(t, err)
	for _, a := range res.Msg.Accounts {
		if a.Id == accountID {
			return a.Balance
		}
	}
	t.Fatalf("account %s not found in ListAccounts", accountID)
	return 0
}

// --- Account balance tests ---

// Balance without any operations equals initialBalance.
func TestBalance_NoOperations(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash USD", "USD", "100", "2024-01-01")
	assert.InDelta(t, 100.0, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Expense reduces balance by the expense amount.
func TestBalance_ExpenseReduces(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash USD", "USD", "100", "2024-01-01")

	_, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: acc.Id,
		Date:      ts("2024-02-01"),
		Amount:    "10",
		Currency:  "USD",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 90.0, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Income increases balance.
func TestBalance_IncomeIncreases(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash USD", "USD", "100", "2024-01-01")

	_, err := h.income.CreateIncome(ctx, connect.NewRequest(&financev1.CreateIncomeRequest{
		UserId:    uid,
		AccountId: acc.Id,
		Date:      ts("2024-02-01"),
		Amount:    "50",
		Currency:  "USD",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 150.0, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Multi-currency expense: charged amount in account currency is used for balance.
func TestBalance_MultiCurrencyExpense(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Wise USD", "USD", "100", "2024-01-01")

	// Paid 5000 KZT, but 11.50 USD was debited from the account.
	_, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId:       acc.Id,
		Date:            ts("2024-02-01"),
		Amount:          "5000",
		Currency:        "KZT",
		ChargedAmount:   ptr("11.50"),
		ChargedCurrency: ptr("USD"),
	}))
	require.NoError(t, err)

	assert.InDelta(t, 88.50, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Multi-currency income: charged amount in account currency is used for balance.
func TestBalance_MultiCurrencyIncome(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Wise USD", "USD", "100", "2024-01-01")

	// Received 5000 KZT, but 11.50 USD credited to the account.
	_, err := h.income.CreateIncome(ctx, connect.NewRequest(&financev1.CreateIncomeRequest{
		UserId:          uid,
		AccountId:       acc.Id,
		Date:            ts("2024-02-01"),
		Amount:          "5000",
		Currency:        "KZT",
		ChargedAmount:   ptr("11.50"),
		ChargedCurrency: ptr("USD"),
	}))
	require.NoError(t, err)

	assert.InDelta(t, 111.50, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Transfer debits from-account and credits to-account.
func TestBalance_TransferDebitCredit(t *testing.T) {
	ctx, h, uid := setup(t)
	from := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "100", "2024-01-01")
	to := newAccount(t, ctx, h, uid, "Cash EUR", "EUR", "0", "2024-01-01")

	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-02-01"),
		FromAccountId: ptr(from.Id),
		FromAmount:    ptr("30"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(to.Id),
		ToAmount:      "30",
		ToCurrency:    "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 70.0, balance(t, ctx, h, uid, from.Id), 0.0001)
	assert.InDelta(t, 30.0, balance(t, ctx, h, uid, to.Id), 0.0001)
}

// Cross-currency transfer: amounts differ, each account uses its own currency.
func TestBalance_CrossCurrencyTransfer(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "100", "2024-01-01")
	kaspi := newAccount(t, ctx, h, uid, "Kaspi KZT", "KZT", "0", "2024-01-01")

	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-02-01"),
		FromAccountId: ptr(wise.Id),
		FromAmount:    ptr("18.51"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(kaspi.Id),
		ToAmount:      "9000",
		ToCurrency:    "KZT",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 81.49, balance(t, ctx, h, uid, wise.Id), 0.0001)
	assert.InDelta(t, 9000.0, balance(t, ctx, h, uid, kaspi.Id), 0.0001)
}

// Transfer with commission: commission is auto-created as expense.
// from_amount stored = request_from_amount - commission (net).
// Commission expense is counted separately → no double-count.
func TestBalance_TransferWithCommission(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "100", "2024-01-01")
	kaspi := newAccount(t, ctx, h, uid, "Kaspi KZT", "KZT", "0", "2024-01-01")

	// 18.34 EUR transfer + 0.17 commission expense = 18.51 total out of Wise
	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:               ts("2024-02-01"),
		FromAccountId:      ptr(wise.Id),
		FromAmount:         ptr("18.34"),
		FromCurrency:       ptr("EUR"),
		ToAccountId:        ptr(kaspi.Id),
		ToAmount:           "10000",
		ToCurrency:         "KZT",
		Commission:         ptr("0.17"),
		CommissionCurrency: ptr("EUR"),
	}))
	require.NoError(t, err)

	// wise: 100 - 18.34 (transfer out) - 0.17 (commission expense) = 81.49
	assert.InDelta(t, 81.49, balance(t, ctx, h, uid, wise.Id), 0.0001)
	assert.InDelta(t, 10000.0, balance(t, ctx, h, uid, kaspi.Id), 0.0001)
}

// External incoming transfer (from_account = nil): balance increases from external source.
func TestBalance_ExternalIncomingTransfer(t *testing.T) {
	ctx, h, uid := setup(t)
	kaspi := newAccount(t, ctx, h, uid, "Kaspi KZT", "KZT", "0", "2024-01-01")

	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:       ts("2024-03-01"),
		ToAccountId: ptr(kaspi.Id),
		ToAmount:   "10000",
		ToCurrency: "KZT",
		// FromAccountId nil = external source
	}))
	require.NoError(t, err)

	assert.InDelta(t, 10000.0, balance(t, ctx, h, uid, kaspi.Id), 0.0001)
}

// Delete expense restores balance.
func TestBalance_DeleteExpenseRestores(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash USD", "USD", "50", "2024-01-01")

	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: acc.Id,
		Date:      ts("2024-01-15"),
		Amount:    "15",
		Currency:  "USD",
	}))
	require.NoError(t, err)
	assert.InDelta(t, 35.0, balance(t, ctx, h, uid, acc.Id), 0.0001)

	_, err = h.expense.DeleteExpense(ctx, connect.NewRequest(&financev1.DeleteExpenseRequest{
		Id: res.Msg.Expense.Id,
	}))
	require.NoError(t, err)
	assert.InDelta(t, 50.0, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Delete income restores balance.
func TestBalance_DeleteIncomeRestores(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash USD", "USD", "100", "2024-01-01")

	res, err := h.income.CreateIncome(ctx, connect.NewRequest(&financev1.CreateIncomeRequest{
		UserId:    uid,
		AccountId: acc.Id,
		Date:      ts("2024-02-01"),
		Amount:    "20",
		Currency:  "USD",
	}))
	require.NoError(t, err)
	assert.InDelta(t, 120.0, balance(t, ctx, h, uid, acc.Id), 0.0001)

	_, err = h.income.DeleteIncome(ctx, connect.NewRequest(&financev1.DeleteIncomeRequest{
		Id: res.Msg.Income.Id,
	}))
	require.NoError(t, err)
	assert.InDelta(t, 100.0, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Delete transfer restores both account balances.
func TestBalance_DeleteTransferRestores(t *testing.T) {
	ctx, h, uid := setup(t)
	from := newAccount(t, ctx, h, uid, "From", "USD", "200", "2024-01-01")
	to := newAccount(t, ctx, h, uid, "To", "USD", "50", "2024-01-01")

	res, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-03-01"),
		FromAccountId: ptr(from.Id),
		FromAmount:    ptr("100"),
		FromCurrency:  ptr("USD"),
		ToAccountId:   ptr(to.Id),
		ToAmount:      "100",
		ToCurrency:    "USD",
	}))
	require.NoError(t, err)
	assert.InDelta(t, 100.0, balance(t, ctx, h, uid, from.Id), 0.0001)
	assert.InDelta(t, 150.0, balance(t, ctx, h, uid, to.Id), 0.0001)

	_, err = h.transfer.DeleteTransfer(ctx, connect.NewRequest(&financev1.DeleteTransferRequest{
		Id: res.Msg.Transfer.Id,
	}))
	require.NoError(t, err)
	assert.InDelta(t, 200.0, balance(t, ctx, h, uid, from.Id), 0.0001)
	assert.InDelta(t, 50.0, balance(t, ctx, h, uid, to.Id), 0.0001)
}

// Update expense amount changes balance accordingly.
func TestBalance_UpdateExpenseAmount(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash USD", "USD", "100", "2024-01-01")

	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: acc.Id,
		Date:      ts("2024-01-20"),
		Amount:    "10",
		Currency:  "USD",
	}))
	require.NoError(t, err)
	assert.InDelta(t, 90.0, balance(t, ctx, h, uid, acc.Id), 0.0001)

	_, err = h.expense.UpdateExpense(ctx, connect.NewRequest(&financev1.UpdateExpenseRequest{
		Id:     res.Msg.Expense.Id,
		Amount: ptr("25"),
	}))
	require.NoError(t, err)
	assert.InDelta(t, 75.0, balance(t, ctx, h, uid, acc.Id), 0.0001)
}

// Update expense account moves the debit to the new account.
func TestBalance_UpdateExpenseMoveAccount(t *testing.T) {
	ctx, h, uid := setup(t)
	accA := newAccount(t, ctx, h, uid, "Account A", "USD", "100", "2024-01-01")
	accB := newAccount(t, ctx, h, uid, "Account B", "USD", "200", "2024-01-01")

	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: accA.Id,
		Date:      ts("2024-01-10"),
		Amount:    "30",
		Currency:  "USD",
	}))
	require.NoError(t, err)
	assert.InDelta(t, 70.0, balance(t, ctx, h, uid, accA.Id), 0.0001)
	assert.InDelta(t, 200.0, balance(t, ctx, h, uid, accB.Id), 0.0001)

	_, err = h.expense.UpdateExpense(ctx, connect.NewRequest(&financev1.UpdateExpenseRequest{
		Id:        res.Msg.Expense.Id,
		AccountId: ptr(accB.Id),
	}))
	require.NoError(t, err)
	assert.InDelta(t, 100.0, balance(t, ctx, h, uid, accA.Id), 0.0001, "accA balance restored")
	assert.InDelta(t, 170.0, balance(t, ctx, h, uid, accB.Id), 0.0001, "accB balance reduced")
}

// --- Account visibility ---

// Deactivated account is not returned by ListAccounts.
func TestDeactivatedAccount_NotInList(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Old Account", "USD", "100", "2024-01-01")

	res, err := h.account.ListAccounts(ctx, connect.NewRequest(&financev1.ListAccountsRequest{UserId: &uid}))
	require.NoError(t, err)
	require.Len(t, res.Msg.Accounts, 1)

	_, err = h.account.DeactivateAccount(ctx, connect.NewRequest(&financev1.DeactivateAccountRequest{
		Id: acc.Id,
	}))
	require.NoError(t, err)

	res, err = h.account.ListAccounts(ctx, connect.NewRequest(&financev1.ListAccountsRequest{UserId: &uid}))
	require.NoError(t, err)
	assert.Empty(t, res.Msg.Accounts, "deactivated account must not appear in active list")
}

// --- FIFO base_amount (tested through CreateExpense response) ---

// EUR account expense → base_amount equals amount directly (no lots needed).
func TestFIFO_EURExpenseHasBaseAmount(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Revolut EUR", "EUR", "500", "2024-01-01")

	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: acc.Id,
		Date:      ts("2024-02-01"),
		Amount:    "65.50",
		Currency:  "EUR",
	}))
	require.NoError(t, err)

	exp := res.Msg.Expense
	require.NotNil(t, exp.BaseAmount)
	assert.InDelta(t, 65.50, pf(t, *exp.BaseAmount), 0.001)
	assert.Equal(t, "EUR", *exp.BaseCurrency)
}

// KZT account expense with no lots → base_amount is not set.
func TestFIFO_KZTExpenseWithoutLotNoBaseAmount(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Cash KZT", "KZT", "10000", "2024-01-01")

	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: acc.Id,
		Date:      ts("2024-02-01"),
		Amount:    "1590",
		Currency:  "KZT",
	}))
	require.NoError(t, err)

	assert.Nil(t, res.Msg.Expense.BaseAmount, "base_amount must be nil when no FIFO lot exists")
}

// Transfer EUR→KZT with rate creates a lot; subsequent KZT expense gets base_amount.
func TestFIFO_KZTExpenseWithLotHasBaseAmount(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "500", "2024-01-01")
	cash := newAccount(t, ctx, h, uid, "Cash KZT", "KZT", "0", "2024-01-01")

	// Transfer creates lot: 91000 KZT at rate 0.0021978 EUR/KZT
	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-04-01"),
		FromAccountId: ptr(wise.Id),
		FromAmount:    ptr("200"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(cash.Id),
		ToAmount:      "91000",
		ToCurrency:    "KZT",
		Rate:          ptr("0.0021978"),
	}))
	require.NoError(t, err)

	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: cash.Id,
		Date:      ts("2024-04-02"),
		Amount:    "1590",
		Currency:  "KZT",
	}))
	require.NoError(t, err)

	exp := res.Msg.Expense
	require.NotNil(t, exp.BaseAmount, "base_amount must be set when FIFO lot exists")
	// 1590 * 0.0021978 ≈ 3.4945 EUR
	assert.InDelta(t, 1590*0.0021978, pf(t, *exp.BaseAmount), 0.01)
	assert.Equal(t, "EUR", *exp.BaseCurrency)
}

// Expense spans two FIFO lots: base_amount is weighted average of both rates.
func TestFIFO_ExpenseSpansMultipleLots(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "1000", "2024-01-01")
	cash := newAccount(t, ctx, h, uid, "Cash KZT", "KZT", "0", "2024-01-01")

	// Lot 1: 50000 KZT at 0.002198 EUR/KZT
	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-04-01"),
		FromAccountId: ptr(wise.Id),
		FromAmount:    ptr("109.90"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(cash.Id),
		ToAmount:      "50000",
		ToCurrency:    "KZT",
		Rate:          ptr("0.002198"),
	}))
	require.NoError(t, err)

	// Lot 2: 50000 KZT at 0.002222 EUR/KZT (newer lot)
	_, err = h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-04-05"),
		FromAccountId: ptr(wise.Id),
		FromAmount:    ptr("111.10"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(cash.Id),
		ToAmount:      "50000",
		ToCurrency:    "KZT",
		Rate:          ptr("0.002222"),
	}))
	require.NoError(t, err)

	// Expense 70000 KZT: consumes 50000 from lot1 + 20000 from lot2
	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: cash.Id,
		Date:      ts("2024-04-10"),
		Amount:    "70000",
		Currency:  "KZT",
	}))
	require.NoError(t, err)

	// 50000*0.002198 + 20000*0.002222 = 109.90 + 44.44 = 154.34 EUR
	assert.InDelta(t, 154.34, pf(t, *res.Msg.Expense.BaseAmount), 0.01)
}

// Delete transfer removes the associated lot; next expense on that account has no base_amount.
func TestFIFO_DeleteTransferRemovesLot(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "500", "2024-01-01")
	cash := newAccount(t, ctx, h, uid, "Cash KZT", "KZT", "0", "2024-01-01")

	trRes, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-04-01"),
		FromAccountId: ptr(wise.Id),
		FromAmount:    ptr("200"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(cash.Id),
		ToAmount:      "91000",
		ToCurrency:    "KZT",
		Rate:          ptr("0.0021978"),
	}))
	require.NoError(t, err)

	_, err = h.transfer.DeleteTransfer(ctx, connect.NewRequest(&financev1.DeleteTransferRequest{
		Id: trRes.Msg.Transfer.Id,
	}))
	require.NoError(t, err)

	// After lot is gone, expense has no base_amount
	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: cash.Id,
		Date:      ts("2024-04-02"),
		Amount:    "1000",
		Currency:  "KZT",
	}))
	require.NoError(t, err)
	assert.Nil(t, res.Msg.Expense.BaseAmount, "lot was deleted with transfer — no base_amount")
}

// Commission2 (to-account fee) is consumed from the freshly created lot.
// After commission2 the remaining lot is smaller; next expense base_amount reflects that.
func TestFIFO_Commission2ReducesLot(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "500", "2024-01-01")
	cash := newAccount(t, ctx, h, uid, "Cash KZT", "KZT", "0", "2024-01-01")

	// Transfer: 91313.27 KZT received, 313.27 KZT ATM fee taken from cash
	_, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:                ts("2024-04-01"),
		FromAccountId:       ptr(wise.Id),
		FromAmount:          ptr("200.16"),
		FromCurrency:        ptr("EUR"),
		ToAccountId:         ptr(cash.Id),
		ToAmount:            "91313.27",
		ToCurrency:          "KZT",
		Commission2:         ptr("313.27"),
		Commission2Currency: ptr("KZT"),
		Rate:                ptr("0.0021978"),
	}))
	require.NoError(t, err)

	// Lot was 91313.27; commission2 consumed 313.27 → remaining 91000
	// Expense 1000 KZT consumed from remaining 91000
	res, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: cash.Id,
		Date:      ts("2024-04-02"),
		Amount:    "1000",
		Currency:  "KZT",
	}))
	require.NoError(t, err)

	require.NotNil(t, res.Msg.Expense.BaseAmount)
	assert.InDelta(t, 1000*0.0021978, pf(t, *res.Msg.Expense.BaseAmount), 0.01)
}

// --- Transfer exchange rate ---

// Cross-currency transfer stores the exchange rate in the transfer record.
func TestTransfer_RateStoredOnCreate(t *testing.T) {
	ctx, h, uid := setup(t)
	wise := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "100", "2024-01-01")
	kaspi := newAccount(t, ctx, h, uid, "Kaspi KZT", "KZT", "0", "2024-01-01")

	res, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-02-01"),
		FromAccountId: ptr(wise.Id),
		FromAmount:    ptr("18.51"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(kaspi.Id),
		ToAmount:      "9000",
		ToCurrency:    "KZT",
		Rate:          ptr("0.0021978"),
	}))
	require.NoError(t, err)

	tr := res.Msg.Transfer
	require.NotNil(t, tr.Rate, "rate must be stored for cross-currency transfer")
	assert.InDelta(t, 0.0021978, pf(t, *tr.Rate), 1e-7)
}

// Same-currency transfer has no exchange rate.
func TestTransfer_RateNullForSameCurrency(t *testing.T) {
	ctx, h, uid := setup(t)
	from := newAccount(t, ctx, h, uid, "Wise EUR", "EUR", "100", "2024-01-01")
	to := newAccount(t, ctx, h, uid, "Cash EUR", "EUR", "0", "2024-01-01")

	res, err := h.transfer.CreateTransfer(ctx, connect.NewRequest(&financev1.CreateTransferRequest{
		Date:          ts("2024-02-01"),
		FromAccountId: ptr(from.Id),
		FromAmount:    ptr("50"),
		FromCurrency:  ptr("EUR"),
		ToAccountId:   ptr(to.Id),
		ToAmount:      "50",
		ToCurrency:    "EUR",
	}))
	require.NoError(t, err)

	assert.Nil(t, res.Msg.Transfer.Rate, "same-currency transfer must not have a rate")
}

// --- Stats tests ---

// GetMonthStats sums base_amount for the given month only.
func TestGetMonthStats_Basic(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Revolut EUR", "EUR", "1000", "2024-01-01")

	for _, tc := range []struct {
		date, amt string
	}{
		{"2026-04-05", "100"},  // ✓ April
		{"2026-04-20", "50"},   // ✓ April
		{"2026-03-15", "999"},  // ✗ different month
	} {
		_, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
			AccountId: acc.Id,
			Date:      ts(tc.date),
			Amount:    tc.amt,
			Currency:  "EUR",
		}))
		require.NoError(t, err)
	}

	res, err := h.stats.GetMonthStats(ctx, connect.NewRequest(&financev1.GetMonthStatsRequest{
		Year: 2026, Month: 4, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 150.0, pf(t, res.Msg.Total), 0.001)
	assert.EqualValues(t, 2, res.Msg.Count)
	assert.EqualValues(t, 30, res.Msg.DaysInMonth)
	assert.InDelta(t, 5.0, pf(t, res.Msg.AveragePerDay), 0.001) // 150/30
}

// Expenses without base_amount (KZT account, no lot) are excluded from stats.
func TestGetMonthStats_ExcludesNullBaseAmount(t *testing.T) {
	ctx, h, uid := setup(t)

	// EUR account: expense auto-gets base_amount → counted
	eurAcc := newAccount(t, ctx, h, uid, "Revolut EUR", "EUR", "1000", "2024-01-01")
	_, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: eurAcc.Id,
		Date:      ts("2026-04-10"),
		Amount:    "2.20",
		Currency:  "EUR",
	}))
	require.NoError(t, err)

	// KZT account with no lot: base_amount = NULL → not counted
	kztAcc := newAccount(t, ctx, h, uid, "Cash KZT", "KZT", "10000", "2024-01-01")
	_, err = h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
		AccountId: kztAcc.Id,
		Date:      ts("2026-04-15"),
		Amount:    "500",
		Currency:  "KZT",
	}))
	require.NoError(t, err)

	res, err := h.stats.GetMonthStats(ctx, connect.NewRequest(&financev1.GetMonthStatsRequest{
		Year: 2026, Month: 4, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.EqualValues(t, 1, res.Msg.Count, "only EUR expense with base_amount is counted")
	assert.InDelta(t, 2.20, pf(t, res.Msg.Total), 0.001)
}

// GetMonthStats for empty month returns zeros.
func TestGetMonthStats_EmptyMonth(t *testing.T) {
	ctx, h, _ := setup(t)

	res, err := h.stats.GetMonthStats(ctx, connect.NewRequest(&financev1.GetMonthStatsRequest{
		Year: 2026, Month: 6, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 0.0, pf(t, res.Msg.Total), 0.001)
	assert.EqualValues(t, 0, res.Msg.Count)
	assert.EqualValues(t, 30, res.Msg.DaysInMonth)
}

// GetYearStats breaks down expenses by month for the year.
func TestGetYearStats_MonthlyBreakdown(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Revolut EUR", "EUR", "5000", "2024-01-01")

	for _, tc := range []struct{ date, amt string }{
		{"2026-01-15", "100"},
		{"2026-01-20", "50"},
		{"2026-03-10", "200"},
		{"2026-04-15", "75"},
		{"2025-12-01", "999"}, // different year — excluded
	} {
		_, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
			AccountId: acc.Id,
			Date:      ts(tc.date),
			Amount:    tc.amt,
			Currency:  "EUR",
		}))
		require.NoError(t, err)
	}

	res, err := h.stats.GetYearStats(ctx, connect.NewRequest(&financev1.GetYearStatsRequest{
		Year: 2026, BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 425.0, pf(t, res.Msg.Total), 0.001)
	require.Len(t, res.Msg.MonthlyStats, 3, "Jan, Mar, Apr")

	byMonth := map[int32]float64{}
	for _, ms := range res.Msg.MonthlyStats {
		byMonth[ms.Month] = pf(t, ms.Total)
	}
	assert.InDelta(t, 150.0, byMonth[1], 0.001)
	assert.InDelta(t, 200.0, byMonth[3], 0.001)
	assert.InDelta(t, 75.0, byMonth[4], 0.001)
}

// GetAllTimeStats returns all months chronologically across years.
func TestGetAllTimeStats_MultipleYears(t *testing.T) {
	ctx, h, uid := setup(t)
	acc := newAccount(t, ctx, h, uid, "Revolut EUR", "EUR", "5000", "2024-01-01")

	for _, tc := range []struct{ date, amt string }{
		{"2025-11-10", "300"},
		{"2026-01-15", "100"},
		{"2026-04-15", "75"},
	} {
		_, err := h.expense.CreateExpense(ctx, connect.NewRequest(&financev1.CreateExpenseRequest{
			AccountId: acc.Id,
			Date:      ts(tc.date),
			Amount:    tc.amt,
			Currency:  "EUR",
		}))
		require.NoError(t, err)
	}

	res, err := h.stats.GetAllTimeStats(ctx, connect.NewRequest(&financev1.GetAllTimeStatsRequest{
		BaseCurrency: "EUR",
	}))
	require.NoError(t, err)

	assert.InDelta(t, 475.0, pf(t, res.Msg.Total), 0.001)
	require.Len(t, res.Msg.MonthlyStats, 3)
	assert.EqualValues(t, 2025, res.Msg.MonthlyStats[0].Year)
	assert.EqualValues(t, 11, res.Msg.MonthlyStats[0].Month)
	assert.InDelta(t, 300.0, pf(t, res.Msg.MonthlyStats[0].Total), 0.001)
}