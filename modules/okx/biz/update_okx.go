package biz

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

type UpdateSubscriptionStorage interface {
	GetSubscriptionById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error)
	UpdateSubscription(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Okx) error
}

func NewUpdateSubscriptionBiz(store UpdateSubscriptionStorage) *updateSubscriptionBiz {
	return &updateSubscriptionBiz{store: store}
}

type updateSubscriptionBiz struct {
	store UpdateSubscriptionStorage
}

func (biz *updateSubscriptionBiz) UpdateSubscription(ctx context.Context, id string, dataUpdate *model.Okx) error {
	_, err := biz.store.GetSubscriptionById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateSubscription(ctx, map[string]interface{}{"_id": id}, dataUpdate); err != nil {
		return err
	}

	newData, err := biz.store.GetSubscriptionById(ctx, map[string]interface{}{"_id": id})
	if err != nil {
		return err
	}

	*dataUpdate = *newData

	return nil
}
