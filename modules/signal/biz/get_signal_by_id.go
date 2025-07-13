package biz

import (
	"context"
	"j_ai_trade/modules/signal/model"
)

type GetSignalByIdStorage interface {
	GetSignalById(ctx context.Context, cond map[string]interface{}) (*model.Signal, error)
}

func NewGetSignalByIdBiz(store GetSignalByIdStorage) *getSignalByIdBiz {
	return &getSignalByIdBiz{store: store}
}

type getSignalByIdBiz struct {
	store GetSignalByIdStorage
}

func (biz *getSignalByIdBiz) GetSignalById(ctx context.Context, id string) (*model.Signal, error) {
	data, err := biz.store.GetSignalById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
