package biz

import (
	"context"
	"j-ai-trade/modules/signal/model"
)

type SignalStorage interface {
	CreateSignal(ctx context.Context, data *model.Signal) error
}

func NewCreateSignalBiz(store SignalStorage) *createSignalBiz {
	return &createSignalBiz{store: store}
}

type createSignalBiz struct {
	store SignalStorage
}

func (biz *createSignalBiz) CreateSignal(ctx context.Context, data *model.Signal) error {
	if err := biz.store.CreateSignal(ctx, data); err != nil {
		return err
	}
	return nil
}
