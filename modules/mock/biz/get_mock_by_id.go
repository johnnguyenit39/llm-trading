package biz

import (
	"context"
	"j-okx-ai/modules/mock/model"
)

type GetMockByIdStorage interface {
	GetMockById(ctx context.Context, cond map[string]interface{}) (*model.Mock, error)
}

func NewGetMockByIdBiz(store GetMockByIdStorage) *getMockByIdBiz {
	return &getMockByIdBiz{store: store}
}

type getMockByIdBiz struct {
	store GetMockByIdStorage
}

func (biz *getMockByIdBiz) GetMockById(ctx context.Context, id string) (*model.Mock, error) {
	data, err := biz.store.GetMockById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
