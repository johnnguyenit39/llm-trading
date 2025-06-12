package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/order/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) UpdateOrder(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Order) error {

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
