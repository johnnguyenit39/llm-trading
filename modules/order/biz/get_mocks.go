package biz

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/order/model"
)

type GetOrdersStorage interface {
	GetOrders(ctx context.Context, paging *common.Pagination) ([]model.Order, error)
}

func NewGetOrdersBiz(store GetOrdersStorage) *getOrdersBiz {
	return &getOrdersBiz{store: store}
}

type getOrdersBiz struct {
	store GetOrdersStorage
}

func (biz *getOrdersBiz) GetOrders(ctx context.Context, paging *common.Pagination) ([]model.Order, error) {
	data, err := biz.store.GetOrders(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
