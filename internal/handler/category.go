package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type CategoryHandler struct {
	pool *pgxpool.Pool
}

func NewCategoryHandler(pool *pgxpool.Pool) *CategoryHandler {
	return &CategoryHandler{pool: pool}
}

func (h *CategoryHandler) ListCategories(ctx context.Context, req *connect.Request[financev1.ListCategoriesRequest]) (*connect.Response[financev1.ListCategoriesResponse], error) {
	rows, err := db.New(h.pool).ListCategories(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cats := make([]*financev1.Category, 0, len(rows))
	for _, r := range rows {
		cats = append(cats, &financev1.Category{
			Id:       uuidToString(r.ID),
			Name:     r.Name,
			Type:     r.Type,
			ParentId: nullUUIDToPtr(r.ParentID),
		})
	}
	return connect.NewResponse(&financev1.ListCategoriesResponse{Categories: cats}), nil
}
