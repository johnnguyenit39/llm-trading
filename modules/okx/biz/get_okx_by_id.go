package biz

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

type GetSubscriptionByIdStorage interface {
	GetSubscriptionById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error)
}

func NewGetSubscriptionByIdBiz(store GetSubscriptionByIdStorage) *getSubscriptionByIdBiz {
	return &getSubscriptionByIdBiz{store: store}
}

type getSubscriptionByIdBiz struct {
	store GetSubscriptionByIdStorage
}

func (biz *getSubscriptionByIdBiz) GetSubscriptionById(ctx context.Context, id string) (*model.Okx, error) {
	data, err := biz.store.GetSubscriptionById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
