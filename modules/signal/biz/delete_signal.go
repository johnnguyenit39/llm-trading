package biz

import (
	"context"
	"j_ai_trade/modules/signal/model"
)

type DeleteNewSignalStorage interface {
	GetSignalById(ctx context.Context, cond map[string]interface{}) (*model.Signal, error)
	DeleteSignal(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteSignalBiz(store DeleteNewSignalStorage) *deleteSignalBiz {
	return &deleteSignalBiz{store: store}
}

type deleteSignalBiz struct {
	store DeleteNewSignalStorage
}

func (biz *deleteSignalBiz) DeleteSignal(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetSignalById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteSignal(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
