package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/permission/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) UpdatePermission(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Permission) error {

	if err := postgresStore.db.Where(cond).Updates(dataUpdate).Error; err != nil {
		return err
	}

	if err := postgresStore.db.Where(cond).First(&dataUpdate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return err
	}

	return nil
}
