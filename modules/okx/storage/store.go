package storage

import (
	"context"
	dto "j-ai-trade/modules/okx/model/dto"
)

type Store interface {
	GetOkxInfo(ctx context.Context, cond map[string]interface{}) (*dto.OkxInfoResponse, error)
	CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) ([]byte, error)
	CancelOrder(ctx context.Context, req *dto.CancelOrderRequest) ([]byte, error)
}
