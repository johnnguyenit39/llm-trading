package biz

import (
	"context"
	"j-ai-trade/modules/api_key/model"
)

type ApiKeyStorage interface {
	CreateApiKey(ctx context.Context, data *model.ApiKey) error
}

func NewCreateApiKeyBiz(store ApiKeyStorage) *createApiKeyBiz {
	return &createApiKeyBiz{store: store}
}

type createApiKeyBiz struct {
	store ApiKeyStorage
}

func (biz *createApiKeyBiz) CreateApiKey(ctx context.Context, data *model.ApiKey) error {
	if err := biz.store.CreateApiKey(ctx, data); err != nil {
		return err
	}
	return nil
}
