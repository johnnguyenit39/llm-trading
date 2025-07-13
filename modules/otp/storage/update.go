package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/otp/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) UpdateOtp(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Otp) error {

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
