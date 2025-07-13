package biz

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/permission/model"
)

type GetPermissionsStorage interface {
	GetPermissions(ctx context.Context, paging *common.Pagination) ([]model.Permission, error)
}

func NewGetPermissionsBiz(store GetPermissionsStorage) *getPermissionsBiz {
	return &getPermissionsBiz{store: store}
}

type getPermissionsBiz struct {
	store GetPermissionsStorage
}

func (biz *getPermissionsBiz) GetPermissions(ctx context.Context, paging *common.Pagination) ([]model.Permission, error) {
	data, err := biz.store.GetPermissions(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
