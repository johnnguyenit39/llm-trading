package biz

import (
	"context"
	"j_ai_trade/modules/order/model"
)

type DeleteNewOrderStorage interface {
	GetOrderById(ctx context.Context, cond map[string]interface{}) (*model.Order, error)
	DeleteOrder(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteOrderBiz(store DeleteNewOrderStorage) *deleteOrderBiz {
	return &deleteOrderBiz{store: store}
}

type deleteOrderBiz struct {
	store DeleteNewOrderStorage
}

func (biz *deleteOrderBiz) DeleteOrder(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetOrderById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteOrder(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
