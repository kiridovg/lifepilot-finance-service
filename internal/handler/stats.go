package handler

import (
	"context"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type StatsHandler struct {
	pool *pgxpool.Pool
}

func NewStatsHandler(pool *pgxpool.Pool) *StatsHandler {
	return &StatsHandler{pool: pool}
}

func (h *StatsHandler) GetMonthStats(ctx context.Context, req *connect.Request[financev1.GetMonthStatsRequest]) (*connect.Response[financev1.GetMonthStatsResponse], error) {
	m := req.Msg
	q := db.New(h.pool)

	from, to := monthRange(int(m.Year), int(m.Month))
	row, err := q.GetMonthStats(ctx, db.GetMonthStatsParams{
		BaseCurrency: pgtype.Text{String: m.BaseCurrency, Valid: true},
		Date:         pgtype.Timestamptz{Time: from, Valid: true},
		Date_2:       pgtype.Timestamptz{Time: to, Valid: true},
		UserID:       nullUUIDFromPtr(m.UserId),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	total := numericToFloat(row.Total)
	daysInMonth := daysInMonth(int(m.Year), int(m.Month))
	avgPerDay := 0.0
	if daysInMonth > 0 {
		avgPerDay = total / float64(daysInMonth)
	}

	return connect.NewResponse(&financev1.GetMonthStatsResponse{
		Total:        formatAmount(total),
		Currency:     m.BaseCurrency,
		Count:        row.Count,
		DaysInMonth:  int32(daysInMonth),
		AveragePerDay: formatAmount(avgPerDay),
	}), nil
}

func (h *StatsHandler) GetYearStats(ctx context.Context, req *connect.Request[financev1.GetYearStatsRequest]) (*connect.Response[financev1.GetYearStatsResponse], error) {
	m := req.Msg
	q := db.New(h.pool)

	from := time.Date(int(m.Year), 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(int(m.Year)+1, 1, 1, 0, 0, 0, 0, time.UTC)

	rows, err := q.GetMonthStatsByYear(ctx, db.GetMonthStatsByYearParams{
		BaseCurrency: pgtype.Text{String: m.BaseCurrency, Valid: true},
		Date:         pgtype.Timestamptz{Time: from, Valid: true},
		Date_2:       pgtype.Timestamptz{Time: to, Valid: true},
		UserID:       nullUUIDFromPtr(m.UserId),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var monthlyStats []*financev1.MonthStat
	var total float64
	for _, r := range rows {
		t := numericToFloat(r.Total)
		total += t
		monthlyStats = append(monthlyStats, &financev1.MonthStat{
			Year:     m.Year,
			Month:    int32(r.Month),
			Total:    formatAmount(t),
			Currency: m.BaseCurrency,
			Count:    r.Count,
		})
	}

	avgPerMonth := 0.0
	if len(rows) > 0 {
		avgPerMonth = total / float64(len(rows))
	}

	return connect.NewResponse(&financev1.GetYearStatsResponse{
		Year:            m.Year,
		Currency:        m.BaseCurrency,
		MonthlyStats:    monthlyStats,
		Total:           formatAmount(total),
		AveragePerMonth: formatAmount(avgPerMonth),
	}), nil
}

func (h *StatsHandler) GetAllTimeStats(ctx context.Context, req *connect.Request[financev1.GetAllTimeStatsRequest]) (*connect.Response[financev1.GetAllTimeStatsResponse], error) {
	m := req.Msg
	q := db.New(h.pool)

	rows, err := q.GetAllTimeMonthlyStats(ctx, db.GetAllTimeMonthlyStatsParams{
		BaseCurrency: pgtype.Text{String: m.BaseCurrency, Valid: true},
		UserID:       nullUUIDFromPtr(m.UserId),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var monthlyStats []*financev1.MonthStat
	var total float64
	for _, r := range rows {
		t := numericToFloat(r.Total)
		total += t
		monthlyStats = append(monthlyStats, &financev1.MonthStat{
			Year:     r.Year,
			Month:    r.Month,
			Total:    formatAmount(t),
			Currency: m.BaseCurrency,
			Count:    r.Count,
		})
	}

	avgPerMonth := 0.0
	if len(rows) > 0 {
		avgPerMonth = total / float64(len(rows))
	}

	return connect.NewResponse(&financev1.GetAllTimeStatsResponse{
		Currency:        m.BaseCurrency,
		MonthlyStats:    monthlyStats,
		Total:           formatAmount(total),
		AveragePerMonth: formatAmount(avgPerMonth),
	}), nil
}

func monthRange(year, month int) (time.Time, time.Time) {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	return from, to
}

func daysInMonth(year, month int) int {
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

func formatAmount(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}