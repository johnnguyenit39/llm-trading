package biz

import (
	"context"
	"j-ai-trade/modules/api_key/model"
)

type DeleteNewApiKeyStorage interface {
	GetApiKeyById(ctx context.Context, cond map[string]interface{}) (*model.ApiKey, error)
	DeleteApiKey(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteApiKeyBiz(store DeleteNewApiKeyStorage) *deleteApiKeyBiz {
	return &deleteApiKeyBiz{store: store}
}

type deleteApiKeyBiz struct {
	store DeleteNewApiKeyStorage
}

func (biz *deleteApiKeyBiz) DeleteApiKey(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetApiKeyById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteApiKey(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
