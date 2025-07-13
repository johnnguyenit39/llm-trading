package biz

import (
	"context"
	"j_ai_trade/modules/permission/model"
)

type UpdatePermissionStorage interface {
	GetPermissionById(ctx context.Context, cond map[string]interface{}) (*model.Permission, error)
	UpdatePermission(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Permission) error
}

func NewUpdatePermissionBiz(store UpdatePermissionStorage) *updatePermissionBiz {
	return &updatePermissionBiz{store: store}
}

type updatePermissionBiz struct {
	store UpdatePermissionStorage
}

func (biz *updatePermissionBiz) UpdatePermission(ctx context.Context, id string, dataUpdate *model.Permission) error {
	_, err := biz.store.GetPermissionById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdatePermission(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
