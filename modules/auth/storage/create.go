package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/user/model"
)

func (postgresStore *postgresStore) Register(ctx context.Context, data *model.User) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return common.ErrDB(err)
	}
	return nil
}
