package biz

import (
	"context"
	"j_ai_trade/modules/api_key/model"
)

type GetApiKeyByIdStorage interface {
	GetApiKeyById(ctx context.Context, cond map[string]interface{}) (*model.ApiKey, error)
}

func NewGetApiKeyByIdBiz(store GetApiKeyByIdStorage) *getApiKeyByIdBiz {
	return &getApiKeyByIdBiz{store: store}
}

type getApiKeyByIdBiz struct {
	store GetApiKeyByIdStorage
}

func (biz *getApiKeyByIdBiz) GetApiKeyById(ctx context.Context, id string) (*model.ApiKey, error) {
	data, err := biz.store.GetApiKeyById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
