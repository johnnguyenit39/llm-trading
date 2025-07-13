package biz

import (
	"context"
	"j_ai_trade/modules/order/model"
)

type GetOrderByIdStorage interface {
	GetOrderById(ctx context.Context, cond map[string]interface{}) (*model.Order, error)
}

func NewGetOrderByIdBiz(store GetOrderByIdStorage) *getOrderByIdBiz {
	return &getOrderByIdBiz{store: store}
}

type getOrderByIdBiz struct {
	store GetOrderByIdStorage
}

func (biz *getOrderByIdBiz) GetOrderById(ctx context.Context, id string) (*model.Order, error) {
	data, err := biz.store.GetOrderById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
