package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
)

type UserHandler struct {
	pool *pgxpool.Pool
}

func NewUserHandler(pool *pgxpool.Pool) *UserHandler {
	return &UserHandler{pool: pool}
}

func (h *UserHandler) ListUsers(ctx context.Context, req *connect.Request[financev1.ListUsersRequest]) (*connect.Response[financev1.ListUsersResponse], error) {
	rows, err := db.New(h.pool).ListUsers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	users := make([]*financev1.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, &financev1.User{
			Id:   uuidToString(r.ID),
			Name: r.Name,
		})
	}
	return connect.NewResponse(&financev1.ListUsersResponse{Users: users}), nil
}
