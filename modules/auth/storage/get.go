package storage

import (
	"context"
	"errors"
	"j_ai_trade/common"
	"j_ai_trade/modules/user/model"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) GetUserByEmail(ctx context.Context, cond map[string]interface{}) (*model.User, error) {
	var data model.User
	if err := postgresStore.db.Where("email = ?", cond["email"]).First(&data).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}
	return &data, nil
}
