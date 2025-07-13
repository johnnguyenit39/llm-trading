package storage

import (
	"context"
	"j_ai_trade/modules/api_key/model"
)

func (postgresStore *postgresStore) CreateApiKey(ctx context.Context, data *model.ApiKey) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
