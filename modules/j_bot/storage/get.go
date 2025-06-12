package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetJbotById(ctx context.Context, cond map[string]interface{}) (*model.Jbot, error) {
	var data model.Jbot
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &data, nil
}
