package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/api_key/model"
	"time"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) DeleteApiKey(ctx context.Context, cond map[string]interface{}) (bool, error) {
	var data model.ApiKey
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
