package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/signal/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetSignalById(ctx context.Context, cond map[string]interface{}) (*model.Signal, error) {
	var data model.Signal
	if err := postgresStore.db.Where(cond).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &data, nil
}
