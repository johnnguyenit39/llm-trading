package biz

import (
	"context"
	"j-ai-trade/modules/signal/model"
)

type UpdateSignalStorage interface {
	GetSignalById(ctx context.Context, cond map[string]interface{}) (*model.Signal, error)
	UpdateSignal(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Signal) error
}

func NewUpdateSignalBiz(store UpdateSignalStorage) *updateSignalBiz {
	return &updateSignalBiz{store: store}
}

type updateSignalBiz struct {
	store UpdateSignalStorage
}

func (biz *updateSignalBiz) UpdateSignal(ctx context.Context, id string, dataUpdate *model.Signal) error {
	_, err := biz.store.GetSignalById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateSignal(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
