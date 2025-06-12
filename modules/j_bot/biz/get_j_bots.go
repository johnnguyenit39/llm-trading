package biz

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/model"
)

type GetJbotsStorage interface {
	GetJbots(ctx context.Context, paging *common.Pagination) ([]model.Jbot, error)
}

func NewGetJbotsBiz(store GetJbotsStorage) *getJbotsBiz {
	return &getJbotsBiz{store: store}
}

type getJbotsBiz struct {
	store GetJbotsStorage
}

func (biz *getJbotsBiz) GetJbots(ctx context.Context, paging *common.Pagination) ([]model.Jbot, error) {
	data, err := biz.store.GetJbots(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
