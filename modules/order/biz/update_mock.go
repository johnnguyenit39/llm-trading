package biz

import (
	"context"
	"j_ai_trade/modules/order/model"
)

type UpdateOrderStorage interface {
	GetOrderById(ctx context.Context, cond map[string]interface{}) (*model.Order, error)
	UpdateOrder(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Order) error
}

func NewUpdateOrderBiz(store UpdateOrderStorage) *updateOrderBiz {
	return &updateOrderBiz{store: store}
}

type updateOrderBiz struct {
	store UpdateOrderStorage
}

func (biz *updateOrderBiz) UpdateOrder(ctx context.Context, id string, dataUpdate *model.Order) error {
	_, err := biz.store.GetOrderById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateOrder(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
