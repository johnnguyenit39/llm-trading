package biz

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

type DeleteNewSubscriptionStorage interface {
	GetSubscriptionById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error)
	DeleteSubscription(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteSubscriptionBiz(store DeleteNewSubscriptionStorage) *deleteSubscriptionBiz {
	return &deleteSubscriptionBiz{store: store}
}

type deleteSubscriptionBiz struct {
	store DeleteNewSubscriptionStorage
}

func (biz *deleteSubscriptionBiz) DeleteSubscription(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetSubscriptionById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteSubscription(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
