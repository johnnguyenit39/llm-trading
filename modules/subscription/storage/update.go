package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/subscription/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) UpdateSubscription(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Subscription) error {

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
