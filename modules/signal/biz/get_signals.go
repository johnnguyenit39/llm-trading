package biz

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/signal/model"
)

type GetSignalsStorage interface {
	GetSignals(ctx context.Context, paging *common.Pagination) ([]model.Signal, error)
}

func NewGetSignalsBiz(store GetSignalsStorage) *getSignalsBiz {
	return &getSignalsBiz{store: store}
}

type getSignalsBiz struct {
	store GetSignalsStorage
}

func (biz *getSignalsBiz) GetSignals(ctx context.Context, paging *common.Pagination) ([]model.Signal, error) {
	data, err := biz.store.GetSignals(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
