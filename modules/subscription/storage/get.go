package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/subscription/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetSubscriptionById(ctx context.Context, cond map[string]interface{}) (*model.Subscription, error) {
	var data model.Subscription
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &data, nil
}
