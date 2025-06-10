package biz

import (
	"context"
	"j-okx-ai/modules/user/model"
)

type GetUserByIdStorage interface {
	GetUserById(ctx context.Context, cond map[string]interface{}) (*model.User, error)
}

func NewGetUserByIdBiz(store GetUserByIdStorage) *getUserByIdBiz {
	return &getUserByIdBiz{store: store}
}

type getUserByIdBiz struct {
	store GetUserByIdStorage
}

func (biz *getUserByIdBiz) GetUserById(ctx context.Context, id string) (*model.User, error) {
	data, err := biz.store.GetUserById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
