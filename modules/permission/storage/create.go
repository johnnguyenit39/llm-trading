package storage

import (
	"context"
	"j-ai-trade/modules/permission/model"
)

func (postgresStore *postgresStore) CreatePermission(ctx context.Context, data *model.Permission) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
