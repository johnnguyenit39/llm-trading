package biz

import (
	"context"
	"j-ai-trade/modules/j_bot/model"
)

type JbotStorage interface {
	CreateJbot(ctx context.Context, data *model.Jbot) error
}

func NewCreateJbotBiz(store JbotStorage) *createJbotBiz {
	return &createJbotBiz{store: store}
}

type createJbotBiz struct {
	store JbotStorage
}

func (biz *createJbotBiz) CreateJbot(ctx context.Context, data *model.Jbot) error {
	if err := biz.store.CreateJbot(ctx, data); err != nil {
		return err
	}
	return nil
}
