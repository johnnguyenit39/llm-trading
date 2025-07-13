package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/signal/model"
	"time"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) DeleteSignal(ctx context.Context, cond map[string]interface{}) (bool, error) {
	var data model.Signal
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
