package biz

import (
	"context"
	"j-okx-ai/common"
	"j-okx-ai/modules/user/model"
)

type GetUsersStorage interface {
	GetUsers(ctx context.Context, paging *common.Pagination) ([]model.User, error)
}

func NewGetUsersBiz(store GetUsersStorage) *getUsersBiz {
	return &getUsersBiz{store: store}
}

type getUsersBiz struct {
	store GetUsersStorage
}

func (biz *getUsersBiz) GetUsers(ctx context.Context, paging *common.Pagination) ([]model.User, error) {
	data, err := biz.store.GetUsers(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
