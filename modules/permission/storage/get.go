package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/permission/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetPermissionById(ctx context.Context, cond map[string]interface{}) (*model.Permission, error) {
	var data model.Permission
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &data, nil
}
