package biz

import (
	"context"
	"j-okx-ai/modules/user/model"
)

type UpdateUserStorage interface {
	GetUserById(ctx context.Context, cond map[string]interface{}) (*model.User, error)
	UpdateUser(ctx context.Context, cond map[string]interface{}, dataUpdate *model.User) error
}

func NewUpdateUserBiz(store UpdateUserStorage) *updateUserBiz {
	return &updateUserBiz{store: store}
}

type updateUserBiz struct {
	store UpdateUserStorage
}

func (biz *updateUserBiz) UpdateUser(ctx context.Context, id string, dataUpdate *model.User) error {
	_, err := biz.store.GetUserById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateUser(ctx, map[string]interface{}{"_id": id}, dataUpdate); err != nil {
		return err
	}

	newData, err := biz.store.GetUserById(ctx, map[string]interface{}{"_id": id})
	if err != nil {
		return err
	}

	*dataUpdate = *newData

	return nil
}
