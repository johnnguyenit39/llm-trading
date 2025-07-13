package biz

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/subscription/model"
)

type GetSubscriptionsStorage interface {
	GetSubscriptions(ctx context.Context, paging *common.Pagination) ([]model.Subscription, error)
}

func NewGetSubscriptionsBiz(store GetSubscriptionsStorage) *getSubscriptionsBiz {
	return &getSubscriptionsBiz{store: store}
}

type getSubscriptionsBiz struct {
	store GetSubscriptionsStorage
}

func (biz *getSubscriptionsBiz) GetSubscriptions(ctx context.Context, paging *common.Pagination) ([]model.Subscription, error) {
	data, err := biz.store.GetSubscriptions(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
