package biz

import (
	"context"
	"j-okx-ai/modules/user/model"
)

type UserStorage interface {
	CreateUser(ctx context.Context, data *model.User) error
}

func NewCreateUserBiz(store UserStorage) *createUserBiz {
	return &createUserBiz{store: store}
}

type createUserBiz struct {
	store UserStorage
}

func (biz *createUserBiz) CreateUser(ctx context.Context, data *model.User) error {
	if err := biz.store.CreateUser(ctx, data); err != nil {
		return err
	}
	return nil
}
