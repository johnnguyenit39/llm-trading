package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/j_bot/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) UpdateJbot(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Jbot) error {

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
