package biz

import (
	"context"
	dto "j-ai-trade/modules/okx/model/dto"
	"j-ai-trade/modules/okx/storage"
)

type CancelOrderBiz struct {
	store storage.Store
}

func NewCancelOrderBiz(store storage.Store) *CancelOrderBiz {
	return &CancelOrderBiz{
		store: store,
	}
}

func (biz *CancelOrderBiz) CancelOrder(ctx context.Context, req *dto.CancelOrderRequest) ([]byte, error) {
	return biz.store.CancelOrder(ctx, req)
}
