package biz

import (
	"context"
	"j-okx-ai/modules/okx/model"
)

type UpdateMockStorage interface {
	GetMockById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error)
	UpdateMock(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Okx) error
}

func NewUpdateMockBiz(store UpdateMockStorage) *updateMockBiz {
	return &updateMockBiz{store: store}
}

type updateMockBiz struct {
	store UpdateMockStorage
}

func (biz *updateMockBiz) UpdateMock(ctx context.Context, id string, dataUpdate *model.Okx) error {
	_, err := biz.store.GetMockById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateMock(ctx, map[string]interface{}{"_id": id}, dataUpdate); err != nil {
		return err
	}

	newData, err := biz.store.GetMockById(ctx, map[string]interface{}{"_id": id})
	if err != nil {
		return err
	}

	*dataUpdate = *newData

	return nil
}
