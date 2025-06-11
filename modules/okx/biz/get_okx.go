package biz

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/okx/model"
)

type GetSubscriptionsStorage interface {
	GetSubscriptions(ctx context.Context, paging *common.Pagination) ([]model.Okx, error)
}

func NewGetSubscriptionsBiz(store GetSubscriptionsStorage) *getSubscriptionsBiz {
	return &getSubscriptionsBiz{store: store}
}

type getSubscriptionsBiz struct {
	store GetSubscriptionsStorage
}

func (biz *getSubscriptionsBiz) GetSubscriptions(ctx context.Context, paging *common.Pagination) ([]model.Okx, error) {
	data, err := biz.store.GetSubscriptions(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
