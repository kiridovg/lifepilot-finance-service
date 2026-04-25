package handler

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1"
	"github.com/kiridovg/lifepilot-finance-service/internal/repository"
)

type TransferHandler struct {
	repo *repository.Repository
}

func NewTransferHandler(repo *repository.Repository) *TransferHandler {
	return &TransferHandler{repo: repo}
}

func (h *TransferHandler) ListTransfers(ctx context.Context, req *connect.Request[financev1.ListTransfersRequest]) (*connect.Response[financev1.ListTransfersResponse], error) {
	transfers, err := h.repo.ListTransfers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var proto []*financev1.Transfer
	for _, t := range transfers {
		proto = append(proto, transferToProto(t))
	}
	return connect.NewResponse(&financev1.ListTransfersResponse{Transfers: proto}), nil
}

func (h *TransferHandler) CreateTransfer(ctx context.Context, req *connect.Request[financev1.CreateTransferRequest]) (*connect.Response[financev1.CreateTransferResponse], error) {
	p := repository.CreateTransferParams{
		FromAmount:         req.Msg.FromAmount,
		FromCurrency:       req.Msg.FromCurrency,
		ToAmount:           req.Msg.ToAmount,
		ToCurrency:         req.Msg.ToCurrency,
		Commission:         req.Msg.Commission,
		CommissionCurrency: req.Msg.CommissionCurrency,
		FromPaymentMethod:  req.Msg.FromPaymentMethod,
		ToPaymentMethod:    req.Msg.ToPaymentMethod,
		Note:               req.Msg.Note,
		Date:               req.Msg.Date.AsTime(),
	}
	t, err := h.repo.CreateTransfer(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.CreateTransferResponse{Transfer: transferToProto(t)}), nil
}

func (h *TransferHandler) DeleteTransfer(ctx context.Context, req *connect.Request[financev1.DeleteTransferRequest]) (*connect.Response[financev1.DeleteTransferResponse], error) {
	if err := h.repo.DeleteTransfer(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&financev1.DeleteTransferResponse{}), nil
}

func transferToProto(t repository.Transfer) *financev1.Transfer {
	return &financev1.Transfer{
		Id:                 t.ID,
		FromAmount:         t.FromAmount,
		FromCurrency:       t.FromCurrency,
		ToAmount:           t.ToAmount,
		ToCurrency:         t.ToCurrency,
		Commission:         t.Commission,
		CommissionCurrency: t.CommissionCurrency,
		FromPaymentMethod:  t.FromPaymentMethod,
		ToPaymentMethod:    t.ToPaymentMethod,
		Note:               t.Note,
		Date:               timestamppb.New(t.Date),
		CreatedAt:          timestamppb.New(t.CreatedAt),
	}
}
