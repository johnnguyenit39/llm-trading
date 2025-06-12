package biz

import (
	"context"
	"j-ai-trade/modules/j_bot/model"
)

type GetJbotByIdStorage interface {
	GetJbotById(ctx context.Context, cond map[string]interface{}) (*model.Jbot, error)
}

func NewGetJbotByIdBiz(store GetJbotByIdStorage) *getJbotByIdBiz {
	return &getJbotByIdBiz{store: store}
}

type getJbotByIdBiz struct {
	store GetJbotByIdStorage
}

func (biz *getJbotByIdBiz) GetJbotById(ctx context.Context, id string) (*model.Jbot, error) {
	data, err := biz.store.GetJbotById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
