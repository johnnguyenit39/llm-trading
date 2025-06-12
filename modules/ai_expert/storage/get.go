package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/ai_expert/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetAiExpertById(ctx context.Context, cond map[string]interface{}) (*model.AiExpert, error) {
	var data model.AiExpert
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &data, nil
}
