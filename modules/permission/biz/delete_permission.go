package biz

import (
	"context"
	"j_ai_trade/modules/permission/model"
)

type DeleteNewPermissionStorage interface {
	GetPermissionById(ctx context.Context, cond map[string]interface{}) (*model.Permission, error)
	DeletePermission(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeletePermissionBiz(store DeleteNewPermissionStorage) *deletePermissionBiz {
	return &deletePermissionBiz{store: store}
}

type deletePermissionBiz struct {
	store DeleteNewPermissionStorage
}

func (biz *deletePermissionBiz) DeletePermission(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetPermissionById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeletePermission(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
