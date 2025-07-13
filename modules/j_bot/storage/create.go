package storage

import (
	"context"
	"j_ai_trade/modules/j_bot/model"
)

func (postgresStore *postgresStore) CreateJbot(ctx context.Context, data *model.Jbot) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
