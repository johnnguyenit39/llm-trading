package biz

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

type GetMockByIdStorage interface {
	GetMockById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error)
}

func NewGetMockByIdBiz(store GetMockByIdStorage) *getMockByIdBiz {
	return &getMockByIdBiz{store: store}
}

type getMockByIdBiz struct {
	store GetMockByIdStorage
}

func (biz *getMockByIdBiz) GetMockById(ctx context.Context, id string) (*model.Okx, error) {
	data, err := biz.store.GetMockById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
