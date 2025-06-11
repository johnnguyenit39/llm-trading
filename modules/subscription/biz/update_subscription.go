package biz

import (
	"context"
	"j-ai-trade/modules/subscription/model"
)

type UpdateSubscriptionStorage interface {
	GetSubscriptionById(ctx context.Context, cond map[string]interface{}) (*model.Subscription, error)
	UpdateSubscription(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Subscription) error
}

func NewUpdateSubscriptionBiz(store UpdateSubscriptionStorage) *updateSubscriptionBiz {
	return &updateSubscriptionBiz{store: store}
}

type updateSubscriptionBiz struct {
	store UpdateSubscriptionStorage
}

func (biz *updateSubscriptionBiz) UpdateSubscription(ctx context.Context, id string, dataUpdate *model.Subscription) error {
	_, err := biz.store.GetSubscriptionById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateSubscription(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
