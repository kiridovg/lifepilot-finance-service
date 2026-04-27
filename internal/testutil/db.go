package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

// NewDB spins up a Postgres container, applies schema, returns a pool.
// Container is terminated when the test ends.
func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts("../../internal/db/schema/schema.sql"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err, "start postgres container")
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "get connection string")

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "connect to test db")
	t.Cleanup(pool.Close)

	return pool
}

// NewQueries returns sqlc Queries and the underlying pool.
func NewQueries(t *testing.T) (*db.Queries, *pgxpool.Pool) {
	t.Helper()
	pool := NewDB(t)
	return db.New(pool), pool
}

// --- Helpers for building pgtype values ---

func Numeric(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	require.NoError(t, n.Scan(s))
	return n
}

func Timestamptz(t *testing.T, date string) pgtype.Timestamptz {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", date)
	require.NoError(t, err)
	return pgtype.Timestamptz{Time: parsed, Valid: true}
}

func Date(t *testing.T, date string) pgtype.Date {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", date)
	require.NoError(t, err)
	return pgtype.Date{Time: parsed, Valid: true}
}

func NullText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func NullNumeric(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	return Numeric(t, s)
}

func NumericToFloat(t *testing.T, n pgtype.Numeric) float64 {
	t.Helper()
	f, err := n.Float64Value()
	require.NoError(t, err)
	return f.Float64
}

// CreateTestUser inserts a user and returns its UUID for use in account creation.
func CreateTestUser(t *testing.T, q *db.Queries) pgtype.UUID {
	t.Helper()
	u, err := q.CreateUser(context.Background(), "Test User")
	require.NoError(t, err)
	return u.ID
}

// SystemCategoryID returns one of the seeded system category UUIDs.
func SystemCategoryID(name string) pgtype.UUID {
	ids := map[string]string{
		"bank-fees": "00000000-0000-0000-0000-000000000001",
		"exchange":  "00000000-0000-0000-0000-000000000002",
		"food":      "00000000-0000-0000-0000-000000000003",
		"transport": "00000000-0000-0000-0000-000000000004",
		"income":    "00000000-0000-0000-0000-000000000005",
	}
	var u pgtype.UUID
	_ = u.Scan(ids[name])
	return u
}
