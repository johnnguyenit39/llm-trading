package biz

import (
	"context"
	"j-ai-trade/modules/j_bot/model"
)

type DeleteNewJbotStorage interface {
	GetJbotById(ctx context.Context, cond map[string]interface{}) (*model.Jbot, error)
	DeleteJbot(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteJbotBiz(store DeleteNewJbotStorage) *deleteJbotBiz {
	return &deleteJbotBiz{store: store}
}

type deleteJbotBiz struct {
	store DeleteNewJbotStorage
}

func (biz *deleteJbotBiz) DeleteJbot(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetJbotById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteJbot(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
