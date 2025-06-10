package biz

import (
	"context"
	"j-okx-ai/modules/mock/model"
)

type MockStorage interface {
	CreateMock(ctx context.Context, data *model.Mock) error
}

func NewCreateMockBiz(store MockStorage) *createMockBiz {
	return &createMockBiz{store: store}
}

type createMockBiz struct {
	store MockStorage
}

func (biz *createMockBiz) CreateMock(ctx context.Context, data *model.Mock) error {
	if err := biz.store.CreateMock(ctx, data); err != nil {
		return err
	}
	return nil
}
