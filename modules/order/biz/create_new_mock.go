package biz

import (
	"context"
	"j-ai-trade/modules/order/model"
)

type OrderStorage interface {
	CreateOrder(ctx context.Context, data *model.Order) error
}

func NewCreateOrderBiz(store OrderStorage) *createOrderBiz {
	return &createOrderBiz{store: store}
}

type createOrderBiz struct {
	store OrderStorage
}

func (biz *createOrderBiz) CreateOrder(ctx context.Context, data *model.Order) error {
	if err := biz.store.CreateOrder(ctx, data); err != nil {
		return err
	}
	return nil
}
