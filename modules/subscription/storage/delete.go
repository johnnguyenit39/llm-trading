package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/subscription/model"
	"time"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) DeleteSubscription(ctx context.Context, cond map[string]interface{}) (bool, error) {
	var data model.Subscription
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return false, err
	}
	now := time.Now().UTC()
	data.DeletedAt = &now

	if err := postgresStore.db.Save(&data).Error; err != nil {
		return false, err
	}
	return true, nil
}
