package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type CurrencyHandler struct {
	pool *pgxpool.Pool
}

func NewCurrencyHandler(pool *pgxpool.Pool) *CurrencyHandler {
	return &CurrencyHandler{pool: pool}
}

func (h *CurrencyHandler) ListCurrencies(ctx context.Context, req *connect.Request[financev1.ListCurrenciesRequest]) (*connect.Response[financev1.ListCurrenciesResponse], error) {
	q := db.New(h.pool)
	rows, err := q.ListCurrencies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	currencies := make([]*financev1.Currency, 0, len(rows))
	for _, r := range rows {
		currencies = append(currencies, &financev1.Currency{
			Code:   r.Code,
			Name:   r.Name,
			Symbol: r.Symbol,
		})
	}

	return connect.NewResponse(&financev1.ListCurrenciesResponse{Currencies: currencies}), nil
}
