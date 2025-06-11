package biz

import (
	"context"
	"j-ai-trade/modules/subscription/model"
)

type SubscriptionStorage interface {
	CreateSubscription(ctx context.Context, data *model.Subscription) error
}

func NewCreateSubscriptionBiz(store SubscriptionStorage) *createSubscriptionBiz {
	return &createSubscriptionBiz{store: store}
}

type createSubscriptionBiz struct {
	store SubscriptionStorage
}

func (biz *createSubscriptionBiz) CreateSubscription(ctx context.Context, data *model.Subscription) error {
	if err := biz.store.CreateSubscription(ctx, data); err != nil {
		return err
	}
	return nil
}
