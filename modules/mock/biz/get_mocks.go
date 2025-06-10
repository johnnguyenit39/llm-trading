package biz

import (
	"context"
	"j-okx-ai/common"
	"j-okx-ai/modules/mock/model"
)

type GetMocksStorage interface {
	GetMocks(ctx context.Context, paging *common.Pagination) ([]model.Mock, error)
}

func NewGetMocksBiz(store GetMocksStorage) *getMocksBiz {
	return &getMocksBiz{store: store}
}

type getMocksBiz struct {
	store GetMocksStorage
}

func (biz *getMocksBiz) GetMocks(ctx context.Context, paging *common.Pagination) ([]model.Mock, error) {
	data, err := biz.store.GetMocks(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
