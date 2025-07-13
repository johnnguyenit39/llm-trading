package biz

import (
	"context"
	"j_ai_trade/modules/api_key/model"
)

type UpdateApiKeyStorage interface {
	GetApiKeyById(ctx context.Context, cond map[string]interface{}) (*model.ApiKey, error)
	UpdateApiKey(ctx context.Context, cond map[string]interface{}, dataUpdate *model.ApiKey) error
}

func NewUpdateApiKeyBiz(store UpdateApiKeyStorage) *updateApiKeyBiz {
	return &updateApiKeyBiz{store: store}
}

type updateApiKeyBiz struct {
	store UpdateApiKeyStorage
}

func (biz *updateApiKeyBiz) UpdateApiKey(ctx context.Context, id string, dataUpdate *model.ApiKey) error {
	_, err := biz.store.GetApiKeyById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateApiKey(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
