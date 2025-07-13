package biz

import (
	"context"
	"j_ai_trade/modules/j_bot/model"
)

type UpdateJbotStorage interface {
	GetJbotById(ctx context.Context, cond map[string]interface{}) (*model.Jbot, error)
	UpdateJbot(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Jbot) error
}

func NewUpdateJbotBiz(store UpdateJbotStorage) *updateJbotBiz {
	return &updateJbotBiz{store: store}
}

type updateJbotBiz struct {
	store UpdateJbotStorage
}

func (biz *updateJbotBiz) UpdateJbot(ctx context.Context, id string, dataUpdate *model.Jbot) error {
	_, err := biz.store.GetJbotById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateJbot(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
