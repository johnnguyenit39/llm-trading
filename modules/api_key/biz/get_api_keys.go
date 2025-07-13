package biz

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/api_key/model"
)

type GetApiKeysStorage interface {
	GetApiKeys(ctx context.Context, paging *common.Pagination) ([]model.ApiKey, error)
}

func NewGetApiKeysBiz(store GetApiKeysStorage) *getApiKeysBiz {
	return &getApiKeysBiz{store: store}
}

type getApiKeysBiz struct {
	store GetApiKeysStorage
}

func (biz *getApiKeysBiz) GetApiKeys(ctx context.Context, paging *common.Pagination) ([]model.ApiKey, error) {
	data, err := biz.store.GetApiKeys(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
