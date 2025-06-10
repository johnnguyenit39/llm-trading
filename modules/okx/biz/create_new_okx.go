package biz

import (
	"context"
	"j-okx-ai/modules/okx/model"
)

type MockStorage interface {
	CreateMock(ctx context.Context, data *model.Okx) error
}

func NewCreateMockBiz(store MockStorage) *createMockBiz {
	return &createMockBiz{store: store}
}

type createMockBiz struct {
	store MockStorage
}

func (biz *createMockBiz) CreateMock(ctx context.Context, data *model.Okx) error {
	if err := biz.store.CreateMock(ctx, data); err != nil {
		return err
	}
	return nil
}
