package biz

import (
	"context"
	"j_ai_trade/modules/permission/model"
)

type GetPermissionByIdStorage interface {
	GetPermissionById(ctx context.Context, cond map[string]interface{}) (*model.Permission, error)
}

func NewGetPermissionByIdBiz(store GetPermissionByIdStorage) *getPermissionByIdBiz {
	return &getPermissionByIdBiz{store: store}
}

type getPermissionByIdBiz struct {
	store GetPermissionByIdStorage
}

func (biz *getPermissionByIdBiz) GetPermissionById(ctx context.Context, id string) (*model.Permission, error) {
	data, err := biz.store.GetPermissionById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
