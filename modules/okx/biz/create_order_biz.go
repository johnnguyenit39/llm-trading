package biz

import (
	"context"
	dto "j-okx-ai/modules/okx/model/dto"
	"j-okx-ai/modules/okx/storage"
)

type CreateOrderBiz struct {
	store storage.Store
}

func NewCreateOrderBiz(store storage.Store) *CreateOrderBiz {
	return &CreateOrderBiz{
		store: store,
	}
}

func (biz *CreateOrderBiz) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) ([]byte, error) {
	return biz.store.CreateOrder(ctx, req)
}
