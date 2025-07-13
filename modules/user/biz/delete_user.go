package biz

import (
	"context"
	"j_ai_trade/modules/user/model"
)

type DeleteNewUserStorage interface {
	GetUserById(ctx context.Context, cond map[string]interface{}) (*model.User, error)
	DeleteUser(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteUserBiz(store DeleteNewUserStorage) *deleteUserBiz {
	return &deleteUserBiz{store: store}
}

type deleteUserBiz struct {
	store DeleteNewUserStorage
}

func (biz *deleteUserBiz) DeleteUser(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetUserById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteUser(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
