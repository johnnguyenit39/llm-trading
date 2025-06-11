package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/api_key/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) UpdateApiKey(ctx context.Context, cond map[string]interface{}, dataUpdate *model.ApiKey) error {

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
