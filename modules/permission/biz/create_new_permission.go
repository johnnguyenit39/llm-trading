package biz

import (
	"context"
	"j_ai_trade/modules/permission/model"
)

type PermissionStorage interface {
	CreatePermission(ctx context.Context, data *model.Permission) error
}

func NewCreatePermissionBiz(store PermissionStorage) *createPermissionBiz {
	return &createPermissionBiz{store: store}
}

type createPermissionBiz struct {
	store PermissionStorage
}

func (biz *createPermissionBiz) CreatePermission(ctx context.Context, data *model.Permission) error {
	if err := biz.store.CreatePermission(ctx, data); err != nil {
		return err
	}
	return nil
}
