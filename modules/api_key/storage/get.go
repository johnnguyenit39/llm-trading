package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/api_key/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetApiKeyById(ctx context.Context, cond map[string]interface{}) (*model.ApiKey, error) {
	var data model.ApiKey
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &data, nil
}
