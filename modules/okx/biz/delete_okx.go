package biz

import (
	"context"
	"j-okx-ai/modules/okx/model"
)

type DeleteNewMockStorage interface {
	GetMockById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error)
	DeleteMock(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteMockBiz(store DeleteNewMockStorage) *deleteMockBiz {
	return &deleteMockBiz{store: store}
}

type deleteMockBiz struct {
	store DeleteNewMockStorage
}

func (biz *deleteMockBiz) DeleteMock(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetMockById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteMock(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
